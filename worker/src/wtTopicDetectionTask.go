package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const ollamaUrl = "http://ollama:11434/api/generate"

type ChapterText struct {
	Text      string
	StartTime int64
}

type OllamaOptions struct {
	NumPredict  int     `json:"num_predict"`
	Temperature float64 `json:"temperature"`
}

type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func (w *Worker) topicDetectionEmbed(ctx context.Context, jobId int, videoURL string) error {
	// get subs, chunks, embeddings, compare via cosine similarity, break into chapters, identify topic for each chunk set, return results

	tempDir, subFiles, err := getSubs(videoURL)
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)

	myChunks, err := chunkSubtitles(subFiles[0], 0, 0) // no overlap between chunks for topic detection
	if err != nil {
		return err
	}

	embedder := NewClient("http://model:8001")
	var texts []string
	for _, chunk := range myChunks {
		texts = append(texts, strings.Join(chunk.Text, " "))
	}

	chapterBoundaries, chapterTexts, err := embedder.getTopicWindows(ctx, texts)
	if err != nil {
		return err
	}

	chapters := make([]ChapterText, 0, len(chapterTexts))

	// fmt.Printf("len(myChunks)=%d\n", len(myChunks))
	// fmt.Printf("chapterBoundaries=%v\n", chapterBoundaries)
	// fmt.Printf("len(chapters)=%d\n", len(chapters))

	for i := range chapterTexts {
		chapters = append(chapters, ChapterText{
			Text:      chapterTexts[i],
			StartTime: myChunks[chapterBoundaries[i]].StartTime / 1000,
		})
	}

	fmt.Printf("boundaries: %v\n", chapterBoundaries)
	fmt.Printf("chunk 1: %v\n", chapterTexts[0])

	resultJSON, err := json.Marshal(chapters)
	if err != nil {
		return fmt.Errorf("Failed to marshal topic detection result: %w", err)
	}

	// fmt.Printf("writing: %v\n", string(resultJSON))

	return w.createLlmJob(ctx, jobId, jobTypeTopicDetectionLLM, resultJSON)
}

func (w *Worker) topicDetectionLLM(ctx context.Context, jobId int) error {
	inputJson, err := w.getLlmJobInfo(ctx, jobId)
	if err != nil {
		return err
	}
	var chapters []ChapterText
	if err := json.Unmarshal(inputJson, &chapters); err != nil {
		return err
	}

	var chapterTitles []ChapterText

	for idx, chapter := range chapters {
		words := strings.Fields(chapter.Text)
		if len(words) > 300 {
			words = words[:300]
		}

		prompt := fmt.Sprintf(
			"Write a short, descriptive video chapter title (under 7 words) for this video transcript segment. Respond with just the title, do not include introductory words or punctuation. Text: %s",
			strings.Join(words, " "),
		)

		title, err := w.runPrompt(ctx, prompt)
		if err != nil {
			fmt.Println("Timed out or encountered error:", err)
			title = fmt.Sprintf("Chapter %d", idx+1)
		}

		chapterTitles = append(chapterTitles, ChapterText{
			Text:      title,
			StartTime: chapter.StartTime,
		})
	}

	resultJSON, err := json.Marshal(chapterTitles)
	if err != nil {
		return fmt.Errorf("Failed to marshal topic detection result: %w", err)
	}

	return w.writeResultToDb(ctx, jobId, resultJSON)
}
