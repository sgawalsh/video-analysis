package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
)

const (
	chunkSize    = 100
	chunkOverlap = 20
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

func formatTimestamp(ms int64) string {
	d := time.Duration(ms) * time.Millisecond

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (w *Worker) semanticSearch(ctx context.Context, jobId int, videoURL string, query string) error {

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

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

	var semanticMatches []semanticMatch

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
