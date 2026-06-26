package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type semanticMatch struct {
	Distance  float32
	Index     int64
	StartTime int64
}

func (w *Worker) semanticSearch(ctx context.Context, jobId int, videoURL string, query string) error {

	// fmt.Printf("trying semantic search with %s, %s\n", videoURL, query)

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0], 20)
	if err != nil {
		return err
	}

	distances, indices, err := searchChunksForQuery(myChunks, query)
	if err != nil {
		return err
	}

	semanticMatches, err := evaluateSearchResults(distances, indices, myChunks)
	if err != nil {
		return err
	}

	fmt.Printf("semantic matches: %v\n", semanticMatches)

	resultJSON, err := json.Marshal(semanticMatches)
	if err != nil {
		return fmt.Errorf("Failed to marshal semantic search result: %w", err)
	}

	return w.writeResultToDb(ctx, jobId, resultJSON)
}

func evaluateSearchResults(distances []float32, indices []int64, chunks []chunk) ([]semanticMatch, error) {

	const threshold float32 = 0.6 // to be raised after testing

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
