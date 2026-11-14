package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

type dateBoundary struct {
	lo int // Last index where condition is false
	hi int // First index where condition is true
}

// get dates, search for videos in channel within date range, create new jobs of session type for each video found
func (w *Worker) channelSearch(ctx context.Context, jobId int, channelID string) error {
	var (
		beginningDateBoundary time.Time
		endingDateBoundary    time.Time
		sessionID             int
	)

	err := w.db.QueryRowContext(ctx, `
		SELECT start_date, end_date, session_id
		FROM jobs
		WHERE id = $1
	`, jobId).Scan(&beginningDateBoundary, &endingDateBoundary, &sessionID)
	if err != nil {
		return fmt.Errorf("load date range: %w", err)
	}

	log.Printf("Searching for videos in channel %s between %s and %s\n", channelID, beginningDateBoundary.Format("2006-01-02"), endingDateBoundary.Format("2006-01-02"))

	stdOut, err := w.runYTDLP(ctx,
		"--flat-playlist",
		// "--cookies-from-browser", "firefox",
		"--print", "id",
		"--no-warnings",
		fmt.Sprintf("https://www.youtube.com/@%s/videos", channelID),
	)
	if err != nil {
		if errors.Is(err, ErrChannelUnavailable) {
			return fmt.Errorf("channel @%q could not be found or is unavailable", channelID)
		}
		return err
	}

	// Split the output into a slice of IDs.
	var ids []string
	trimmed := strings.TrimSpace(string(stdOut))
	if trimmed != "" {
		ids = strings.Split(trimmed, "\n")
	}

	log.Printf("Checking: %s\n", ids)

	iStart, iEnd, err := w.dateSearch(ctx, ids, beginningDateBoundary, endingDateBoundary)
	if err != nil {
		return fmt.Errorf("binary search failed: %w", err)
	}

	if iEnd < iStart {
		log.Printf("No videos found within the specified date range.")
	} else {
		publicID, jobType, query, err := w.getSessionInfoById(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session info: %w", err)
		}

		toInsert := ids[iStart : iEnd+1]
		successCount, err := w.insertBatch(ctx, sessionID, publicID, jobType, query, toInsert)
		if err != nil {
			return fmt.Errorf("failed to insert batch: %w", err)
		}
		log.Printf("Inserted %d/%d videos\n", successCount, len(toInsert))
	}

	log.Printf("Setting channel search success")

	return w.setResultAndSuccessStatus(ctx, jobId, "", []byte("[]"))
}

func (w *Worker) dateSearch(ctx context.Context, ids []string, beginningDateBoundary time.Time, endingDateBoundary time.Time) (int, int, error) {

	if len(ids) == 0 || beginningDateBoundary.After(endingDateBoundary) {
		return 0, -1, nil
	}

	sampledIDs, sampledIndices := getSampleIDs(ids, 32)

	sampledDates, err := w.getUploadDates(ctx, sampledIDs)
	if err != nil {
		return 0, 0, err
	}

	if len(sampledDates) != len(sampledIndices) {
		return 0, 0, fmt.Errorf("expected %d upload dates, received %d", len(sampledIndices), len(sampledDates))
	}

	dates := make(map[int]time.Time, len(sampledIndices))

	for i, index := range sampledIndices {
		dates[index] = sampledDates[i]
	}

	// No videos within the requested date range.
	if dates[0].Before(beginningDateBoundary) || dates[len(ids)-1].After(endingDateBoundary) {
		log.Printf("No videos found in requested range.")
		return 0, -1, nil
	}

	return w.refineDateBoundaries(ctx, ids, dates, beginningDateBoundary, endingDateBoundary)
}

func (w *Worker) refineDateBoundaries(ctx context.Context, ids []string, dates map[int]time.Time, startDate time.Time, endDate time.Time) (int, int, error) {

	endCondition := func(date time.Time) bool {
		return !date.After(endDate)
	}

	startCondition := func(date time.Time) bool {
		return date.Before(startDate)
	}

	endBoundary := findDateBoundary(
		dates, len(ids), endCondition,
	)

	startBoundary := findDateBoundary(
		dates, len(ids), startCondition,
	)

	const refinementSamples = 8

	//refinement loop
	for !endBoundary.resolved() || !startBoundary.resolved() {
		indices := make(map[int]struct{})

		if !endBoundary.resolved() {
			for _, i := range endBoundary.sampleIndices(refinementSamples) {
				indices[i] = struct{}{}
			}
		}

		if !startBoundary.resolved() {
			for _, i := range startBoundary.sampleIndices(refinementSamples) {
				indices[i] = struct{}{}
			}
		}

		var missingIndices []int
		var missingIDs []string

		for index := range indices {
			if _, exists := dates[index]; !exists {
				missingIndices = append(missingIndices, index)
				missingIDs = append(missingIDs, ids[index])
			}
		}

		// Sort to keep requests deterministic and preserve chronological order.
		sort.Ints(missingIndices)

		missingIDs = missingIDs[:0]
		for _, index := range missingIndices {
			missingIDs = append(missingIDs, ids[index])
		}

		if len(missingIDs) > 0 {
			newDates, err := w.getUploadDates(ctx, missingIDs)
			if err != nil {
				return 0, 0, err
			}

			if len(newDates) != len(missingIndices) {
				return 0, 0, fmt.Errorf(
					"expected %d upload dates, received %d",
					len(missingIndices), len(newDates),
				)
			}

			for i, index := range missingIndices {
				dates[index] = newDates[i]
			}
		}

		if !endBoundary.resolved() {
			endBoundary.refine(
				endBoundary.sampleIndices(refinementSamples),
				dates,
				endCondition,
			)
		}

		if !startBoundary.resolved() {
			startBoundary.refine(
				startBoundary.sampleIndices(refinementSamples),
				dates,
				startCondition,
			)
		}
	}

	return endBoundary.hi, startBoundary.hi - 1, nil
}

func findDateBoundary(dates map[int]time.Time, length int, condition func(time.Time) bool) dateBoundary {
	indices := make([]int, 0, len(dates))

	for index := range dates {
		indices = append(indices, index)
	}

	sort.Ints(indices)

	for i, index := range indices {
		if condition(dates[index]) {
			lo := -1
			if i > 0 {
				lo = indices[i-1]
			}

			return dateBoundary{
				lo: lo,
				hi: index,
			}
		}
	}

	// No sampled index satisfies the condition.
	return dateBoundary{
		lo: indices[len(indices)-1],
		hi: length,
	}
}

func (b *dateBoundary) refine(indices []int, dates map[int]time.Time, condition func(time.Time) bool) {
	for _, index := range indices {
		if condition(dates[index]) {
			b.hi = index
			return
		}

		b.lo = index
	}
}

func (b dateBoundary) sampleIndices(n int) []int {
	indices := evenlySpacedIndices(b.hi-b.lo+1, n+2)

	for i := range indices {
		indices[i] += b.lo
	}

	return indices
}

func (b dateBoundary) resolved() bool {
	return b.hi-b.lo <= 1
}

func evenlySpacedIndices(length, n int) []int {
	if length <= 0 || n <= 0 {
		return nil
	}

	if n >= length {
		indices := make([]int, length)
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	if n == 1 {
		return []int{(length - 1) / 2}
	}

	indices := make([]int, n)

	for i := range indices {
		indices[i] = i * (length - 1) / (n - 1)
	}

	return indices
}

func getSampleIDs(ids []string, n int) ([]string, []int) {
	indices := evenlySpacedIndices(len(ids), n)

	sampledIDs := make([]string, len(indices))

	for i, index := range indices {
		sampledIDs[i] = ids[index]
	}

	return sampledIDs, indices
}

func (w *Worker) getUploadDates(ctx context.Context, videoIDs []string) ([]time.Time, error) {
	if len(videoIDs) == 0 {
		return []time.Time{}, nil
	}

	args := []string{
		"--print", "upload_date",
		"--no-warnings",
		"--",
	}

	for _, videoID := range videoIDs {
		args = append(args, fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID))
	}

	stdOut, err := w.runYTDLP(ctx, args...)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(stdOut)), "\n")
	if len(lines) != len(videoIDs) {
		return nil, fmt.Errorf(
			"expected %d upload dates, received %d",
			len(videoIDs), len(lines),
		)
	}

	uploadDates := make([]time.Time, len(lines))
	for i, line := range lines {
		uploadDate, err := time.Parse("20060102", strings.TrimSpace(line))
		if err != nil {
			return nil, fmt.Errorf("failed to parse upload date for video %s: %w", videoIDs[i], err)
		}
		uploadDates[i] = uploadDate
	}

	return uploadDates, nil
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
