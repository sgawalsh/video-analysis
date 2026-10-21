package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/asticode/go-astisub"
)

const (
	chunkSize    = 100
	insertBuffer = 1
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

func (w *Worker) getSubs(ctx context.Context, videoURL string) (string, []string, string, error) {
	titleOutput, err := w.runYTDLP(ctx, []string{
		"--print", "%(title)s",
		"--skip-download",
		"--",
		videoURL,
	}...)
	if err != nil {
		log.Printf("yt-dlp failed to get title: %v", err)
		log.Printf("yt-dlp output: %s", string(titleOutput))
		return "", nil, "", fmt.Errorf("yt-dlp title read failed: %w", err)
	}

	title := strings.TrimSpace(string(titleOutput))
	log.Printf("video title: %s", title)

	tempDir, err := os.MkdirTemp("", "subs-*")
	if err != nil {
		return "", nil, "", err
	}

	// Download subtitles
	outputTemplate := filepath.Join(
		tempDir,
		"%(id)s.%(ext)s",
	)

	_, err = w.runYTDLP(ctx, []string{
		"--write-auto-sub",
		"--skip-download",
		"--sub-lang", "en-orig",
		"--convert-subs", "vtt",
		"--output", outputTemplate,
		"--",
		videoURL,
	}...)
	if err != nil {
		return "", nil, "", fmt.Errorf("yt-dlp failed to download subtitles: %w", err)
	}

	files, err := filepath.Glob(
		filepath.Join(tempDir, "*.vtt"),
	)
	if err != nil {
		return "", nil, "", err
	}

	if len(files) == 0 {
		return "", nil, "", fmt.Errorf("no transcript found")
	}

	return tempDir, files, title, nil
}

func isHyphen(f string) bool {
	return f == "-"
}

func chunkSubtitles(path string, chunkOverlap int, chunkSizeParam int) ([]chunk, error) {
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

	targetSize := max(chunkSize, chunkSizeParam) // compares global chunkSize with param

	for i, item := range subs.Items {
		words := strings.Fields(item.String())
		words = slices.DeleteFunc(words, isHyphen) // remove isolated hyphens

		if len(currChunk.Text)+len(words) <= targetSize {
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

func getTimeStampFromWordIndex(indexTimes []wordIndexTime, wordIndex int) int64 {
	for i := len(indexTimes) - 1; i >= 0; i-- {
		if indexTimes[i].wordIndex <= wordIndex {
			return indexTimes[i].timeStamp
		}
	}
	return 0
}
