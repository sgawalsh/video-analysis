package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Topic struct {
	Title     string
	StartTime int64
}

func (w *Worker) topicDetection(ctx context.Context, jobId int, videoURL string) error {
	// get subs, chunks, embeddings, compare via cosine similarity, break into topics, identify topic for each chunk set, return results

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	// fmt.Printf("got the subs!\n")

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0], 0) // no overlap between chunks for topic detection
	if err != nil {
		return err
	}

	embedder := NewClient("http://model:8001")
	var texts []string
	for _, chunk := range myChunks {
		texts = append(texts, strings.Join(chunk.Text, " "))
	}

	windowBoundaries, chapters, err := embedder.getTopicWindows(ctx, texts)
	if err != nil {
		return err
	}

	for i, chunk := range myChunks {
		fmt.Printf(string(i) + "\n")
		fmt.Printf(string(chunk.StartTime) + "\n")
	}

	topics := make([]Topic, 0, len(chapters))

	// fmt.Printf("len(myChunks)=%d\n", len(myChunks))
	// fmt.Printf("windowBoundaries=%v\n", windowBoundaries)
	// fmt.Printf("len(chapters)=%d\n", len(chapters))

	for i := range chapters {
		topics = append(topics, Topic{
			Title:     chapters[i].Title,
			StartTime: myChunks[windowBoundaries[i]].StartTime / 1000,
		})
	}

	if len(chapters) > 0 {
		fmt.Printf("chapter title: %v\n", chapters[0].Title)
	}

	fmt.Printf("boundaries: %v\n", windowBoundaries)
	fmt.Printf("chapter titles: %v\n", chapters[0].Title)

	resultJSON, err := json.Marshal(topics)
	if err != nil {
		return fmt.Errorf("Failed to marshal topic detection result: %w", err)
	}

	// fmt.Printf("writing: %v\n", string(resultJSON))

	return w.writeResultToDb(ctx, jobId, resultJSON)
}
