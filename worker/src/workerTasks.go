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

type chunk struct {
	Text      []string
	StartTime int64
	EndTime   int64
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

func queryTranscripts(ctx context.Context, videoURL string) error {

	tempDir, err := os.MkdirTemp("", "subs-*")

	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)

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

	output, err := cmd.CombinedOutput()

	fmt.Println(string(output))

	if err != nil {
		return err
	}

	files, err := filepath.Glob(
		filepath.Join(tempDir, "*.vtt"),
	)

	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no transcript found")
	}

	// return logTranscript(files[0])
	myChunks, err := chunkSubtitles(files[0])
	if err != nil {
		return err
	}
	for i, c := range myChunks {
		fmt.Printf(
			"Chunk %d: [%s - %s] %s\n",
			i,
			formatTimestamp(c.StartTime),
			formatTimestamp(c.EndTime),
			strings.Join(c.Text, " "),
		)
	}
	return nil
}

var chunkSize = 100
var chunkOverlap = 20

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

func logTranscript(path string) error {

	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()

	subs, err := astisub.ReadFromWebVTT(file)

	if err != nil {
		return err
	}

	subs.Unfragment()

	// embedder := NewClient("http://model:8001")
	// var texts []string
	// for _, item := range subs.Items {
	// 	texts = append(texts, item.String())
	// }
	// embeddings, err := embedder.Embed(context.Background(), texts)

	// if err != nil {
	// 	return err
	// }

	for _, item := range subs.Items {

		fmt.Printf(
			"[%v - %v] %s\n",
			item.StartAt,
			item.EndAt,
			item.String(),
		)
	}

	// for i, embedding := range embeddings {
	// 	fmt.Printf("Embedding for subtitle %d: %v\n", i, embedding)
	// }

	return nil
}
