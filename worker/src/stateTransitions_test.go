package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/sgawalsh/video-analysis/worker/src/testutil"
	"github.com/stretchr/testify/require"
)

var jobType = "SEMANTIC_SEARCH"

func refreshDB(t *testing.T) (*sql.DB, context.Context) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateJobs(t, db)
	ctx := context.Background()
	return db, ctx
}

func createTestSession(t *testing.T, db *sql.DB, jobType string) (int, string) {
	t.Helper()

	var id int
	var public_id string
	err := db.QueryRow(`
            INSERT INTO sessions (type)
            VALUES ($1)
			RETURNING id, public_id
            `,
		jobType,
	).Scan(&id, &public_id)
	require.NoError(t, err)

	return id, public_id
}

func TestRequeueStuckRunningJobs(t *testing.T) {
	db, ctx := refreshDB(t)
	sessionID, publicID := createTestSession(t, db, jobType)

	// Seed data, interval must be greater that worker_running_timeout
	_, err := db.Exec(`
		INSERT INTO jobs (
			session_id,
			session_public_id,
			type,
			target_id,
			query,
			status,
			started_at
		)
		VALUES
		($1, $2, $3, 'a', 'a', 'RUNNING', NOW() - INTERVAL '1 DAY'),
		($1, $2, $3, 'b', 'b', 'RUNNING', NOW() - INTERVAL '5 MINUTES'),
		($1, $2, $3, 'c', 'c', 'RUNNING', NOW())
	`, sessionID, publicID, jobType)
	require.NoError(t, err)

	ids, err := requeueStuckRunningJobs(ctx, db)
	require.NoError(t, err)
	require.Len(t, ids, 1)

	// Assert DB state
	rows, err := db.Query(`
		SELECT status FROM jobs ORDER BY id
	`)
	require.NoError(t, err)

	var statuses []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		statuses = append(statuses, s)
	}

	require.Equal(t,
		[]string{"PENDING", "RUNNING", "RUNNING"},
		statuses,
	)
}

func TestClaimJob(t *testing.T) {
	db, ctx := refreshDB(t)
	sessionID, publicID := createTestSession(t, db, jobType)

	_, err := db.Exec(`
		INSERT INTO jobs (
			session_id,
			session_public_id,
			type,
			target_id,
			query,
			status
		)
		VALUES
			($1, $2, $3, 'a', 'a', 'PENDING'),
			($1, $2, $3, 'b', 'b', 'PENDING'),
			($1, $2, $3, 'c', 'c', 'PENDING')
	`, sessionID, publicID, jobType)
	require.NoError(t, err)

	w, err := NewWorker()
	require.NoError(t, err)

	jobInfo, err := w.claimNextJob(ctx, []string{jobType})
	require.NoError(t, err)

	require.Equal(t, 1, jobInfo.ID)

	// Assert DB state
	rows, err := db.Query(`
		SELECT status FROM jobs ORDER BY id
	`)
	require.NoError(t, err)

	var statuses []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		statuses = append(statuses, s)
	}

	require.Equal(t,
		[]string{"RUNNING", "PENDING", "PENDING"},
		statuses,
	)
}

func TestHandleJobFailure(t *testing.T) {
	db, ctx := refreshDB(t)
	sessionID, publicID := createTestSession(t, db, jobType)
	maxAttempts, err := strconv.Atoi(os.Getenv("worker_max_job_retries"))
	require.NoError(t, err)

	rows, err := db.QueryContext(ctx, `
		INSERT INTO jobs (
			session_id,
			session_public_id,
			type,
			target_id,
			query,
			status,
			attempts
		)
		VALUES
			($1, $2, $3, 'a', 'a', 'RUNNING', $4),
			($1, $2, $3, 'b', 'b', 'RUNNING', $5),
			($1, $2, $3, 'c', 'c', 'RUNNING', $6)
		RETURNING id
	`, sessionID, publicID, jobType, maxAttempts-2, maxAttempts-1, maxAttempts)
	require.NoError(t, err)

	var jobIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			jobIDs = append(jobIDs, id)
		}
	}

	w, err := NewWorker()
	require.NoError(t, err)
	err = errors.New("a basic error message")
	for _, id := range jobIDs {
		w.handleJobFailure(ctx, id, err)
	}

	// Assert DB state
	rows, err = db.Query(`
		SELECT status FROM jobs ORDER BY id
	`)
	require.NoError(t, err)

	var statuses []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		statuses = append(statuses, s)
	}

	require.Equal(t,
		[]string{"PENDING", "FAILED", "FAILED"},
		statuses,
	)
}
