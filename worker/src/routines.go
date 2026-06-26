package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusPending   = "PENDING"
	StatusRunning   = "RUNNING"
	StatusSucceeded = "SUCCEEDED"
	StatusFailed    = "FAILED"
)

type JobInfo struct {
	ID       int
	Type     string
	TargetID string
	Query    sql.NullString
}

var pollingInterval = os.Getenv("worker_poll_interval")
var workerTimeoutInterval = os.Getenv("worker_execution_timeout_interval")

func waitForNotification(ctx context.Context, conn *sql.Conn, timeout time.Duration) error {
	return conn.Raw(func(driverConn any) error {
		// Use type assertion to get the stdlib.Conn wrapper
		// Then call .Conn() to get the native *pgx.Conn
		pgxConn := driverConn.(*stdlib.Conn).Conn()

		waitCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		_, err := pgxConn.WaitForNotification(waitCtx)
		return err
	})
}

func (w *Worker) executeDBJobs(ctx context.Context, channelNameCommand string, allowedJobTypes []string, handlerMap map[string]JobHandler, successCounter prometheus.Counter, failCounter prometheus.Counter) {
	log.Println("DB worker is running...")

	// Dedicated connection for LISTEN / NOTIFY
	conn, err := w.db.Conn(ctx)
	if err != nil {
		log.Fatalf("Failed to get DB conn for LISTEN: %v", err)
	}
	defer conn.Close()

	// Start listening
	if _, err := conn.ExecContext(ctx, channelNameCommand); err != nil {
		log.Fatalf("LISTEN failed: %v", err)
	}
	// Parse timeout interval
	workerTimeoutInterval, err := strconv.Atoi(workerTimeoutInterval)
	if err != nil {
		log.Fatalf("Failed to parse worker_execution_timeout_interval: %v", err)
		return
	}

	log.Printf("Listening for job notifications using %s", channelNameCommand)
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker shutting down")
			return

		default:
			// Wait for notification OR timeout
			err = waitForNotification(ctx, conn, time.Duration(workerTimeoutInterval)*time.Second)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				log.Printf("Notification wait error: %v", err)
			}

			// Drain all available jobs
			for {
				JobInfo, err := w.claimNextJob(ctx, allowedJobTypes)
				if err != nil {
					if err == sql.ErrNoRows {
						break // no available work
					}
					log.Printf("Failed to claim job: %v", err)
					break
				}

				log.Printf("Claimed job %d of type %s", JobInfo.ID, JobInfo.Type)

				handler, ok := handlerMap[JobInfo.Type]
				if !ok {
					log.Printf("Unknown job type %s", JobInfo.Type)
					w.failJob(ctx, JobInfo.ID, errors.New("unknown job type"), failCounter)
					continue
				}

				if err := handler(ctx, JobInfo); err != nil {
					w.failJob(ctx, JobInfo.ID, err, failCounter)
					continue
				}

				if _, err := w.db.ExecContext(ctx, `
					UPDATE jobs
					SET status = $1
					WHERE id = $2 AND status = $3
				`, StatusSucceeded, JobInfo.ID, StatusRunning); err != nil {
					log.Printf("Failed to set job status to SUCCEEDED for job %d: %v", JobInfo.ID, err)
					continue
				}

				log.Printf("Job %d completed successfully", JobInfo.ID)

				successCounter.Inc()
			}
		}
	}
}

func (w *Worker) pollPendingJobs(ctx context.Context) {
	interval, err := strconv.Atoi(pollingInterval)
	if err != nil {
		log.Printf("Failed to parse worker_poll_interval: %v", err)
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Poller stopping...")
			return
		case <-ticker.C:
			jobIDs, err := requeueStuckRunningJobs(ctx, w.db) // move stuck running jobs to PENDING
			if err != nil {
				log.Printf("Poller: failed to requeue stuck running jobs: %v", err)
				continue
			}
			if len(jobIDs) > 0 {
				log.Printf("Poller: enqueued %d jobs", len(jobIDs))
			}
		}
	}
}

func (w *Worker) failJob(ctx context.Context, jobID int, err error, pCounter prometheus.Counter) {
	w.handleJobFailure(ctx, jobID, err)
	pCounter.Inc()
	log.Printf("Job %d failed: %v", jobID, err)
}
