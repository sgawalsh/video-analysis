package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lib/pq"
)

var maxAttempts = os.Getenv("worker_max_job_retries")
var runningTimeout = os.Getenv("worker_running_timeout")

type JobType string

const (
	jobTypeChannelSearch      JobType = "CHANNEL_SEARCH"
	jobTypeSemanticSearch     JobType = "SEMANTIC_SEARCH"
	jobTypeTopicDetection     JobType = "TOPIC_DETECTION"
	jobTypeKeywordSearch      JobType = "KEYWORD_SEARCH"
	jobTypeVideoSummarization JobType = "VIDEO_SUMMARIZATION"
)

// Requeue stuck RUNNING jobs that have exceeded the time limit
func requeueStuckRunningJobs(ctx context.Context, db *sql.DB) ([]int, error) {
	rows, err := db.QueryContext(ctx, `
		UPDATE jobs
		SET status = $1,
		started_at = NULL
		WHERE id IN (
			SELECT id
			FROM jobs
			WHERE status=$2
			AND started_at < NOW() - ($3)::interval
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 10
		)
		RETURNING id
	`, StatusPending, StatusRunning, runningTimeout)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var jobIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			jobIDs = append(jobIDs, id)
		}
	}

	return jobIDs, nil
}

func (w *Worker) claimNextJob(ctx context.Context) (int, string, string, string, error) {
	var (
		jobID    int
		jobType  string
		targetID string
		query    sql.NullString
	)

	err := w.db.QueryRowContext(ctx, `
		UPDATE jobs
		SET status = $1,
		    started_at = NOW()
		WHERE id IN (
			SELECT id
			FROM jobs
			WHERE status = $2
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, type, target_id, query
	`, StatusRunning, StatusPending).Scan(
		&jobID,
		&jobType,
		&targetID,
		&query,
	)

	if err != nil {
		return 0, "", "", "", err
	}

	return jobID, jobType, targetID, query.String, nil
}

func (w *Worker) handleJobFailure(ctx context.Context, jobID int, err error) {
	var attempts int

	// increment attempts counter and set to pending or failed based on attempts < max_attempts
	err2 := w.db.QueryRowContext(ctx, `
        UPDATE jobs
        SET attempts = attempts + 1,
            last_error = $2,
            status = CASE
                WHEN attempts + 1 >= $3 THEN $4::job_status
                ELSE $5::job_status
            END
        WHERE id = $1
		AND status = $6
        RETURNING attempts
    `, jobID, err.Error(), maxAttempts, StatusFailed, StatusPending, StatusRunning).Scan(&attempts)

	if err2 != nil {
		log.Printf("Failed to update retry state for job %d: %v", jobID, err2)
	}
}

func (w *Worker) writeSemanticSearchResultToDb(ctx context.Context, jobID int, semanticMatches []semanticMatch) error {

	resultJSON, err := json.Marshal(semanticMatches)
	if err != nil {
		return fmt.Errorf("Failed to marshal semantic search result: %w", err)
	}

	_, err = w.db.ExecContext(ctx, `
        UPDATE jobs
		SET result = $2
		WHERE id = $1
    `, jobID, resultJSON)

	if err != nil {
		return fmt.Errorf("Failed to update semantic search result for job %d: %v", jobID, err)
	}

	return nil
}

func (w *Worker) insertBatch(ctx context.Context, sessionID int, publicID string, jobType JobType, query string, videoIDs []string) (int, error) {
	if len(videoIDs) == 0 {
		return 0, nil
	}

	res, err := w.db.ExecContext(
		ctx,
		`
        INSERT INTO jobs (session_id, session_public_id, type, target_id, query )
        SELECT $1, $2, $3::job_type, unnest($4::text[]), $5
        ON CONFLICT (session_id, target_id) DO NOTHING
        `,
		sessionID, publicID, jobType, pq.Array(videoIDs), query,
	)

	if err != nil {
		return 0, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch rows affected: %w", err)
	}

	return int(rows), nil
}
