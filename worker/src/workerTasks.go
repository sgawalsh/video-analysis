package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
)

const (
	chunkSize    = 100
	chunkOverlap = 20
	insertBuffer = 1
)

type chunk struct {
	Text      []string
	StartTime int64
	EndTime   int64
}

type semanticMatch struct {
	Distance  float32
	Index     int64
	StartTime int64
}

type wordIndexTime struct {
	wordIndex int
	timeStamp int64
}

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

	return nil
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
		publicID, query string
		jobType         JobType
	)

	err := w.db.QueryRowContext(ctx, `
		SELECT public_id, type, query
		FROM sessions
		WHERE id = $1
	`, sessionId).Scan(&publicID, &jobType, &query)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get session info: %w", err)
	}

	return publicID, jobType, query, nil
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

func (w *Worker) semanticSearch(ctx context.Context, jobId int, videoURL string, query string) error {

	fmt.Printf("trying semantic search with %s, %s\n", videoURL, query)

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	fmt.Printf("got the subs!\n")

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0])
	if err != nil {
		return err
	}

	for i := 0; i < len(myChunks); i++ {
		myChunks[i] = formatText(myChunks[i])
	}

	distances, indices, err := searchChunksForQuery(myChunks, query)
	if err != nil {
		return err
	}

	semanticMatches, err := evaluateSearchResults(distances, indices, myChunks)
	if err != nil {
		return err
	}

	err = w.writeSemanticSearchResultToDb(ctx, jobId, semanticMatches)
	return err
}

func evaluateSearchResults(distances []float32, indices []int64, chunks []chunk) ([]semanticMatch, error) {

	const threshold float32 = 0.5

	semanticMatches := make([]semanticMatch, 0)

	// sanity check
	if len(distances) != len(indices) {
		return nil, fmt.Errorf("distances and indices length mismatch")
	}

	for i, distance := range distances {

		// skip weak matches
		if distance < threshold {
			continue
		}

		chunkIndex := indices[i]

		// FAISS can return -1 when no result is found
		if chunkIndex < 0 || int(chunkIndex) >= len(chunks) {
			continue
		}

		semanticMatches = append(semanticMatches, semanticMatch{
			Distance:  distance,
			Index:     chunkIndex,
			StartTime: chunks[chunkIndex].StartTime / 1000, // convert ms to seconds
		})
	}

	return semanticMatches, nil
}

func getSubs(videoURL string) (string, []string, error) {
	tempDir, err := os.MkdirTemp("", "subs-*")

	if err != nil {
		return "", nil, err
	}

	outputTemplate := filepath.Join(
		tempDir,
		"%(id)s.%(ext)s",
	)
	cmd := exec.Command(
		"yt-dlp",
		"--write-auto-sub",
		"--skip-download",
		"--sub-lang", "en",
		"--convert-subs", "vtt",
		"--output", outputTemplate,
		videoURL,
	)
	_, err = cmd.CombinedOutput()

	if err != nil {
		return "", nil, err
	}

	files, err := filepath.Glob(
		filepath.Join(tempDir, "*.vtt"),
	)

	if err != nil {
		return "", nil, err
	}

	if len(files) == 0 {
		return "", nil, fmt.Errorf("no transcript found")
	}

	return tempDir, files, nil
}

func formatText(c chunk) chunk { // remove isolated hyphens
	c.Text = slices.DeleteFunc(c.Text, func(f string) bool {
		return f == "-"
	})
	return c
}

func mergeWithOverlap(existing, incoming []string) []string {
	maxOverlap := min(len(existing), len(incoming))

	for overlap := maxOverlap; overlap > 0; overlap-- {
		if slices.Equal(
			existing[len(existing)-overlap:],
			incoming[:overlap],
		) {
			return append(existing, incoming[overlap:]...)
		}
	}

	return append(existing, incoming...)
}

func chunkSubtitles(path string) ([]chunk, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	subs, err := astisub.ReadFromWebVTT(file)

	if err != nil {
		return nil, err
	}

	subs.Unfragment()

	chunks := []chunk{}
	indexTimes := []wordIndexTime{}

	currChunk := chunk{
		StartTime: subs.Items[0].StartAt.Milliseconds(),
	}

	for i, item := range subs.Items {
		words := strings.Fields(item.String())

		if len(currChunk.Text)+len(words) <= chunkSize {
			indexTimes = append(indexTimes, wordIndexTime{
				wordIndex: len(currChunk.Text),
				timeStamp: item.StartAt.Milliseconds(),
			})

			currChunk.Text = mergeWithOverlap(currChunk.Text, words)
			continue
		}

		currChunk.EndTime = subs.Items[i-1].EndAt.Milliseconds()
		chunks = append(chunks, currChunk)

		// build overlap seed
		start := max(0, len(currChunk.Text)-chunkOverlap)

		currChunk = chunk{
			StartTime: getTimeStampFromWordIndex(indexTimes, start),
			Text:      append([]string{}, currChunk.Text[start:]...),
		}

		currChunk.Text = mergeWithOverlap(currChunk.Text, words)
		currChunk.EndTime = item.EndAt.Milliseconds()

		indexTimes = []wordIndexTime{
			{wordIndex: len(currChunk.Text), timeStamp: item.EndAt.Milliseconds()},
		}

	}
	currChunk.EndTime = subs.Items[len(subs.Items)-1].EndAt.Milliseconds()
	chunks = append(chunks, currChunk)

	return chunks, nil
}

func getTimeStampFromWordIndex(indexTimes []wordIndexTime, wordIndex int) int64 {
	for i := len(indexTimes) - 1; i >= 0; i-- {
		if indexTimes[i].wordIndex <= wordIndex {
			return indexTimes[i].timeStamp
		}
	}
	return 0
}

func formatTimestamp(ms int64) string {
	d := time.Duration(ms) * time.Millisecond

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func searchChunksForQuery(chunks []chunk, query string) ([]float32, []int64, error) {
	embedder := NewClient("http://model:8001")
	var texts []string
	for _, item := range chunks {
		texts = append(texts, strings.Join(item.Text, " "))
	}

	distances, indices, err := embedder.embedAndSearch(context.Background(), texts, query)
	if err != nil {
		return nil, nil, err
	}

	return distances, indices, nil
}
