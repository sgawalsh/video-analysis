package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var ytdlpLimiter = func() time.Duration {
	limiter, err := time.ParseDuration(os.Getenv("worker_ytdlp_limiter"))
	if err != nil {
		log.Printf("Invalid worker_ytdlp_limiter: %v", err)
		return 0
	}
	return limiter
}()

const (
	ytdlp429BaseDelay          = 30 * time.Second
	ytdlp429MaxDelay           = 15 * time.Minute
	ytDLPAdvisoryLockKey int64 = 827364019
)

func (w *Worker) runYTDLP(ctx context.Context, args ...string) ([]byte, error) {
	conn, release, err := w.rateLimiter(ctx)
	if err != nil {
		if release != nil {
			release()
		}
		return nil, err
	}

	defer func() {
		if release != nil {
			release()
		}
	}()

	// Reserve this launch slot before starting yt-dlp.
	updateCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	_, updateErr := conn.ExecContext(
		updateCtx,
		`UPDATE yt_dlp_rate_limit
		SET last_launch = clock_timestamp()
		WHERE name = 'global'`,
	)
	cancel()

	if updateErr != nil {
		return nil, fmt.Errorf("record yt-dlp launch slot: %w", updateErr)
	}

	// The launch slot is recorded. Another worker may now proceed.
	release()
	release = nil

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf(
			"start yt-dlp: %w: %s",
			err,
			stderr.String(),
		)
	}

	start := time.Now()

	log.Printf("Starting yt-dlp: args=%q time=%s", args, start.Format(time.RFC3339Nano))

	if err := cmd.Wait(); err != nil {
		duration := time.Since(start)
		is429 := strings.Contains(stderr.String(), "HTTP Error 429")
		log.Printf("yt-dlp failed: duration=%s rate_limited=%t err=%v stderr=%s", duration, is429, err, stderr.String())

		if is429 {
			log.Printf("yt-dlp rate limited, recording outcome")
			outcomeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			if recordErr := w.recordYTDLPOutcome(outcomeCtx, true); recordErr != nil {
				log.Printf("record yt-dlp 429: %v", recordErr)
			}
			cancel()
		}

		return stdout.Bytes(), fmt.Errorf("yt-dlp failed: %w: %s", err, stderr.String())
	}

	log.Printf("yt-dlp completed successfully: duration=%s", time.Since(start))

	outcomeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	if err := w.recordYTDLPOutcome(outcomeCtx, false); err != nil {
		log.Printf("reset yt-dlp failure counter: %v", err)
	}
	cancel()

	return stdout.Bytes(), nil
}

func (w *Worker) rateLimiter(ctx context.Context) (*sql.Conn, func(), error) {
	for {
		maxTime, conn, release, err := w.getNextLaunchTime(ctx)
		if err != nil {
			if release != nil {
				release()
			}
			return nil, nil, fmt.Errorf("get next launch time: %w", err)
		}

		wait := time.Until(maxTime)

		// The launch time has arrived. Return with the lock held so the caller can reserve the launch before releasing it.
		if wait <= 0 {
			return conn, release, nil
		}

		// Don't hold the global lock while sleeping.
		release()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			// Loop back and re-read the shared state.
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, nil, ctx.Err()
		}
	}
}

func (w *Worker) getNextLaunchTime(ctx context.Context) (time.Time, *sql.Conn, func(), error) {
	conn, err := w.db.Conn(ctx)
	if err != nil {
		return time.Time{}, nil, nil, fmt.Errorf("get rate limiter connection: %w", err)
	}

	locked := false

	release := func() {
		if !locked {
			_ = conn.Close()
			return
		}

		var unlocked bool
		unlockCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

		err := conn.QueryRowContext(
			unlockCtx,
			`SELECT pg_advisory_unlock($1)`,
			int64(ytDLPAdvisoryLockKey),
		).Scan(&unlocked)

		cancel()

		if err != nil || !unlocked {
			log.Printf("yt-dlp advisory unlock failed: unlocked=%v err=%v", unlocked, err)
			_ = conn.Raw(func(any) error {
				return driver.ErrBadConn
			})
		}

		_ = conn.Close()
		locked = false
	}

	if _, err := conn.ExecContext(
		ctx,
		`SELECT pg_advisory_lock($1)`,
		int64(ytDLPAdvisoryLockKey),
	); err != nil {
		_ = conn.Raw(func(any) error {
			return driver.ErrBadConn
		})
		_ = conn.Close()

		return time.Time{}, nil, nil, fmt.Errorf("acquire yt-dlp advisory lock: %w", err)
	}
	locked = true

	// Find the next possible launch slot
	var nextAllowed, cooldownUntil time.Time

	err = conn.QueryRowContext(
		ctx,
		`SELECT last_launch, cooldown_until
		FROM yt_dlp_rate_limit
		WHERE name = 'global'`,
	).Scan(&nextAllowed, &cooldownUntil)
	if err != nil {
		return time.Time{}, nil, release, fmt.Errorf("read yt-dlp rate limit state: %w", err)
	}

	nextAllowed = nextAllowed.Add(ytdlpLimiter)
	if cooldownUntil.After(nextAllowed) {
		nextAllowed = cooldownUntil
	}

	return nextAllowed, conn, release, nil
}

func (w *Worker) recordYTDLPOutcome(ctx context.Context, rateLimited bool) error {
	conn, err := w.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get rate limit connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(
		ctx,
		`SELECT pg_advisory_lock($1)`,
		int64(ytDLPAdvisoryLockKey),
	); err != nil {
		return fmt.Errorf("acquire rate limit lock: %w", err)
	}

	// Always release the session-level lock.
	defer func() {
		unlockCtx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()

		var unlocked bool
		if err := conn.QueryRowContext(
			unlockCtx,
			`SELECT pg_advisory_unlock($1)`,
			int64(ytDLPAdvisoryLockKey),
		).Scan(&unlocked); err != nil || !unlocked {
			log.Printf("yt-dlp advisory unlock failed: %v", err)
			_ = conn.Raw(func(any) error {
				return driver.ErrBadConn
			})
		}
	}()

	if !rateLimited { // A successful command resets the failure streak, but does not shorten a cooldown already imposed by another worker.
		_, err := conn.ExecContext(
			ctx,
			`UPDATE yt_dlp_rate_limit
             SET consecutive_429 = 0
             WHERE name = 'global'`,
		)
		return err
	}

	var failures int
	if err := conn.QueryRowContext(
		ctx,
		`UPDATE yt_dlp_rate_limit
         SET consecutive_429 = consecutive_429 + 1
         WHERE name = 'global'
         RETURNING consecutive_429`,
	).Scan(&failures); err != nil {
		return fmt.Errorf("increment 429 counter: %w", err)
	}

	delay := ytdlp429BaseDelay
	for i := 1; i < failures && delay < ytdlp429MaxDelay; i++ {
		delay *= 2
	}
	if delay > ytdlp429MaxDelay {
		delay = ytdlp429MaxDelay
	}

	_, err = conn.ExecContext(
		ctx,
		`UPDATE yt_dlp_rate_limit
		SET cooldown_until = GREATEST(
			cooldown_until,
			clock_timestamp() + ($1 * INTERVAL '1 millisecond')
		)
		WHERE name = 'global'`,
		delay.Milliseconds(),
	)
	if err != nil {
		return fmt.Errorf("update 429 cooldown: %w", err)
	}

	log.Printf("yt-dlp HTTP 429: consecutive failures=%d, cooldown=%s", failures, delay)
	return nil
}
