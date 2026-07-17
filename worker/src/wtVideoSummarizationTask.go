package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
)

type SummaryResult struct {
	Text string
}

func (w *Worker) videoSummarizationTranscribe(ctx context.Context, jobId int, videoURL string) error {
	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0], 0, math.MaxInt) // no overlap between chunks for summarization, using MaxInt to create one large chunk
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal(strings.Join(myChunks[0].Text, " "))

	log.Printf("Writing transcript: %s", strings.Join(myChunks[0].Text, " "))

	return w.createLlmJob(ctx, jobId, jobTypeVideoSummarizationLLM, resultJSON)
}

func (w *Worker) videoSummarizationLLM(ctx context.Context, jobId int) error {

	input, err := w.getLlmJobInfo(ctx, jobId)
	if err != nil {
		return err
	}

	var transcription string
	if err := json.Unmarshal(input, &transcription); err != nil {
		return err
	}

	prompt := fmt.Sprintf(
		"Write a summary (under 100 words) for this video transcription. Respond with just the summary, do not include introductory words or punctuation.\nTranscription: %s",
		transcription,
	)

	summary, err := w.runPrompt(ctx, prompt)
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal([]SummaryResult{{Text: summary}}) // frontend expects array of results
	if err != nil {
		return fmt.Errorf("Failed to marshal video summarization result: %w", err)
	}

	return w.setResultAndSuccessStatusWithLlmInfoDelete(ctx, jobId, resultJSON)
}
