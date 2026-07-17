package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/lib/pq"
)

var maxAttempts = os.Getenv("worker_max_job_retries")
var runningTimeout = os.Getenv("worker_running_timeout")

type JobType string

const (
	jobTypeChannelSearch                JobType = "CHANNEL_SEARCH"
	jobTypeKeywordSearch                JobType = "KEYWORD_SEARCH"
	jobTypeSemanticSearch               JobType = "SEMANTIC_SEARCH"
	jobTypeTopicDetectionEmbed          JobType = "TOPIC_DETECTION_EMBED"
	jobTypeTopicDetectionLLM            JobType = "TOPIC_DETECTION_LLM"
	jobTypeVideoSummarizationLLM        JobType = "VIDEO_SUMMARIZATION_LLM"
	jobTypeVideoSummarizationTranscribe JobType = "VIDEO_SUMMARIZATION_TRANSCRIBE"
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

func (w *Worker) claimNextJob(ctx context.Context, jobTypes []string) (JobInfo, error) {
	var job JobInfo

	err := w.db.QueryRowContext(ctx, `
		UPDATE jobs
		SET status = $1,
		    started_at = NOW()
		WHERE id IN (
			SELECT id
			FROM jobs
			WHERE status = $2 AND type = ANY($3)
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, type, target_id, query
	`, StatusRunning, StatusPending, pq.Array(jobTypes)).Scan(
		&job.ID,
		&job.Type,
		&job.TargetID,
		&job.Query,
	)

	if err != nil {
		return job, err
	}

	return job, nil
}

func (w *Worker) handleJobFailure(ctx context.Context, jobId int, err error) {
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
    `, jobId, err.Error(), maxAttempts, StatusFailed, StatusPending, StatusRunning).Scan(&attempts)

	if err2 != nil {
		log.Printf("Failed to update retry state for job %d: %v", jobId, err2)
	}
}

func (w *Worker) setResultAndSuccessStatus(ctx context.Context, jobId int, resultJSON []byte) error {

	_, err := w.db.ExecContext(ctx, `
        UPDATE jobs
		SET result = $2,
		status = $3
		WHERE id = $1
    `, jobId, resultJSON, StatusSucceeded)

	if err != nil {
		return fmt.Errorf("Failed to update result for job %d: %v", jobId, err)
	}

	log.Printf("Job %d completed successfully", jobId)
	w.succeeded.Inc()
	return nil
}

func (w *Worker) setResultAndSuccessStatusWithLlmInfoDelete(ctx context.Context, jobId int, resultJSON []byte) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
        UPDATE jobs
		SET result = $2,
		status = $3
		WHERE id = $1
    `, jobId, resultJSON, StatusSucceeded)

	if err != nil {
		return fmt.Errorf("Failed to update result for job %d: %v", jobId, err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM llm_job_info
		WHERE job_id = $1
	`, jobId)
	if err != nil {
		return fmt.Errorf("failed to insert llm job info for job %d: %w", jobId, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit llm job for job %d: %w", jobId, err)
	}

	log.Printf("Job %d completed successfully", jobId)
	w.succeeded.Inc()
	return nil
}

func (w *Worker) createLlmJob(ctx context.Context, jobId int, newType JobType, inputJSON []byte) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE jobs
		SET type = $1, status = $2
		WHERE id = $3
	`, newType, StatusPending, jobId)
	if err != nil {
		return fmt.Errorf("failed to update job %d: %w", jobId, err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO llm_job_info (job_id, input)
		VALUES ($1, $2)
		ON CONFLICT (job_id) 
    	DO UPDATE SET input = EXCLUDED.input
	`, jobId, inputJSON)
	if err != nil {
		return fmt.Errorf("failed to insert llm job info for job %d: %w", jobId, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit llm job for job %d: %w", jobId, err)
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
