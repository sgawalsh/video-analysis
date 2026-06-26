package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
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

	myChunks, err := chunkSubtitles(subFiles[0], 0) // no overlap between chunks for topic detection
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
	chapters, err := w.getLlmJobInfo(ctx, jobId)
	if err != nil {
		return err
	}

	fmt.Printf("summarizing: %v\n", chapters)

	var segmentedChapters []ChapterText

	// Set up an HTTP client with a 500-second timeout to match Python's timeout=500
	client := &http.Client{
		Timeout: 500 * time.Second,
	}

	for idx, chapter := range chapters {
		// 1. Python " ".join(text.split()[:300]) equivalent
		words := strings.Fields(chapter.Text)
		if len(words) > 300 {
			words = words[:300]
		}
		truncatedText := strings.Join(words, " ")

		// 2. Format the prompt string
		prompt := fmt.Sprintf(
			"Write a short, descriptive video chapter title (under 7 words) for this transcript segment. Respond with just the title, do not include introductory words or punctuation. Text: %s",
			truncatedText,
		)

		// 3. Build the Ollama JSON payload
		payload := OllamaRequest{
			Model:  "llama3.2:1b",
			Prompt: prompt,
			Stream: false,
			Options: OllamaOptions{
				NumPredict:  8,
				Temperature: 0.1,
			},
		}

		// Set default fallback title
		title := fmt.Sprintf("Chapter %d", idx+1)

		// Inline function to handle try/except block cleanly
		err := func() error {
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			// Bind request to the context for cancellation support
			req, err := http.NewRequestWithContext(ctx, "POST", ollamaUrl, bytes.NewBuffer(jsonData))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("bad status: %s", resp.Status)
			}

			var ollamaResp OllamaResponse
			if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
				return err
			}

			title = strings.TrimSpace(ollamaResp.Response)
			return nil
		}()

		// "except Exception:" handling
		if err != nil {
			fmt.Println("Timed out or encountered error:", err)
			title = fmt.Sprintf("Chapter %d", idx+1)
		}

		segmentedChapters = append(segmentedChapters, ChapterText{
			Text:      title,
			StartTime: chapters[idx].StartTime,
		})

		fmt.Printf("Title: %s\n", title)
	}

	resultJSON, err := json.Marshal(segmentedChapters)
	if err != nil {
		return fmt.Errorf("Failed to marshal semantic search result: %w", err)
	}

	fmt.Printf("writing: %v\n", segmentedChapters)

	return w.writeResultToDb(ctx, jobId, resultJSON)
}

func (w *Worker) getLlmJobInfo(ctx context.Context, jobId int) ([]ChapterText, error) {
	var chaptersJSON []byte

	err := w.db.QueryRowContext(ctx, `
        SELECT input FROM llm_job_info WHERE job_id = $1
        `, jobId).Scan(&chaptersJSON)
	if err != nil {
		return nil, err
	}

	var chapters []ChapterText
	if err := json.Unmarshal(chaptersJSON, &chapters); err != nil {
		return nil, err
	}

	return chapters, nil
}
