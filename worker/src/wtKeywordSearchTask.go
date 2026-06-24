package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type keywordMatch struct {
	MatchCount int
	StartTime  int64
}

func (w *Worker) keywordSearch(ctx context.Context, jobId int, videoURL string, query string) error {

	// fmt.Printf("trying keyword search with %s, %s\n", videoURL, query)

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	// fmt.Printf("got the subs!\n")

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0], 20)
	if err != nil {
		return err
	}

	keywords := parseQuery(query)

	matchCounts, err := searchChunksForKeywords(myChunks, keywords)
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal(matchCounts)
	if err != nil {
		return fmt.Errorf("Failed to marshal keyword search result: %w", err)
	}

	return w.writeResultToDb(ctx, jobId, resultJSON)
}

func searchChunksForKeywords(chunks []chunk, keywords []string) ([]keywordMatch, error) {
	keywordMatches := make([]keywordMatch, 0)
	var matchCount int

	for _, chunk := range chunks {
		joined := strings.ToLower(strings.Join(chunk.Text, " "))
		matchCount = 0

		for _, keyword := range keywords {
			if containsPhrase(strings.Fields(joined), strings.Fields(keyword)) {
				matchCount++
			}
		}
		if matchCount > 0 {
			keywordMatches = append(keywordMatches, keywordMatch{
				MatchCount: matchCount,
				StartTime:  chunk.StartTime / 1000, //convert to seconds
			})
		}
	}

	return keywordMatches, nil
}

func containsPhrase(tokens []string, phrase []string) bool {
	for i := 0; i <= len(tokens)-len(phrase); i++ {
		match := true
		for j := range phrase {
			if tokens[i+j] != phrase[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func parseQuery(query string) []string {
	re := regexp.MustCompile(`"([^"]+)"|(\S+)`)

	matches := re.FindAllStringSubmatch(query, -1)

	var result []string
	for _, match := range matches {
		if match[1] != "" {
			// Quoted phrase (without quotes)
			result = append(result, match[1])
		} else {
			// Single word
			result = append(result, match[2])
		}
	}

	return result
}
