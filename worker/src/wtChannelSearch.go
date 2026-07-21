package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// get dates, search for videos in channel within date range, create new jobs of session type for each video found
func (w *Worker) channelSearch(ctx context.Context, jobId int, channelID string) error {
	var (
		startDate time.Time
		endDate   time.Time
		sessionID int
	)

	err := w.db.QueryRowContext(ctx, `
		SELECT start_date, end_date, session_id
		FROM jobs
		WHERE id = $1
	`, jobId).Scan(&startDate, &endDate, &sessionID)
	if err != nil {
		return fmt.Errorf("load date range: %w", err)
	}

	fmt.Printf("Searching for videos in channel %s between %s and %s\n", channelID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"--flat-playlist",
		// "--cookies-from-browser firefox",
		"--print", "id",
		"--no-warnings",
		"--ignore-errors",
		fmt.Sprintf("https://www.youtube.com/%s/videos", channelID),
	)

	stdOutBuf, err := runCommand(cmd)
	if err != nil {
		return err
	}

	// Split the buffer string into a slice of IDs
	var ids []string
	if stdOutBuf.Len() > 0 {
		// Clean up trailing newline before splitting
		trimmed := strings.TrimSpace(stdOutBuf.String())
		if trimmed != "" {
			ids = strings.Split(trimmed, "\n")
		}
	}

	fmt.Printf("Checking: %s\n", ids)

	iStart, iEnd, err := binarySearchByUploadDate(ctx, ids, startDate, endDate)
	if err != nil {
		return fmt.Errorf("binary search failed: %w", err)
	}

	publicID, jobType, query, err := w.getSessionInfoById(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}

	if iEnd < iStart {
		fmt.Println("No videos found within the specified date range.")
		return nil
	}

	toInsert := ids[iStart : iEnd+1]
	successCount, err := w.insertBatch(ctx, sessionID, publicID, jobType, query, toInsert)
	if err != nil {
		return fmt.Errorf("failed to insert batch: %w", err)
	}

	// fmt.Printf("warnings:\n%s\n", errBytes)
	// fmt.Printf("video ids:\n%s\n", idsBytes)
	fmt.Printf("Inserted %d/%d videos\n", successCount, len(toInsert))

	return w.setResultAndSuccessStatus(ctx, jobId, []byte("[]"))
}

// finds the index range of videos uploaded between startDate and endDate (inclusive).
// yt-dlp outputs lists from NEWEST to OLDEST.
func binarySearchByUploadDate(ctx context.Context, ids []string, startDate, endDate time.Time) (int, int, error) {
	if len(ids) == 0 {
		return 0, -1, nil // Return empty slice boundaries
	}

	var searchErr error

	//Find the FIRST index where uploadDate <= endDate
	iStart := sort.Search(len(ids), func(i int) bool {
		date, err := getUploadDate(ctx, ids[i])
		if err != nil {
			searchErr = err
			return true
		}
		// this video is within or older than target upper bound
		return !date.After(endDate)
	})

	if searchErr != nil {
		return 0, 0, searchErr
	}

	// Find the FIRST index where uploadDate < startDate (The first video that is too old)
	// search from iStart onwards to save API calls.
	iEndOffset := sort.Search(len(ids)-iStart, func(i int) bool {
		actualIndex := iStart + i
		date, err := getUploadDate(ctx, ids[actualIndex])
		if err != nil {
			searchErr = err
			return true
		}
		// True means this video is strictly older than our allowed window
		return date.Before(startDate)
	})

	if searchErr != nil {
		return 0, 0, searchErr
	}

	iEnd := iStart + iEndOffset - 1

	// Validate if any videos were actually found in the range
	if iStart > iEnd || iStart >= len(ids) {
		return 0, -1, nil // No videos match criteria
	}

	return iStart, iEnd, nil
}

func getUploadDate(ctx context.Context, videoID string) (time.Time, error) {
	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		// "--cookies-from-browser firefox",
		"--print", "upload_date",
		"--no-warnings",
		"--ignore-errors",
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
	)

	stdOutBuf, err := runCommand(cmd)
	if err != nil {
		return time.Time{}, err
	}

	// Parse the upload date
	uploadDate, err := time.Parse("20060102", strings.TrimSpace(stdOutBuf.String()))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse upload date: %w", err)
	}

	return uploadDate, nil
}

func (w *Worker) getSessionInfoById(ctx context.Context, sessionId int) (string, JobType, string, error) {
	var (
		publicID string
		jobType  JobType
		query    sql.NullString
	)

	err := w.db.QueryRowContext(ctx, `
		SELECT public_id, type, query
		FROM sessions
		WHERE id = $1
	`, sessionId).Scan(&publicID, &jobType, &query)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get session info: %w", err)
	}

	return publicID, jobType, query.String, nil
}

func runCommand(cmd *exec.Cmd) (bytes.Buffer, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Run executes the command and waits for it to complete
	if err := cmd.Run(); err != nil {
		return bytes.Buffer{}, fmt.Errorf(
			"yt-dlp failed: %w\nstderr:\n%s",
			err,
			stderrBuf.String(),
		)
	}
	return stdoutBuf, nil
}
