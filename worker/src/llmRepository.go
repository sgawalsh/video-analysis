package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (w *Worker) runPrompt(ctx context.Context, prompt string) (string, error) { // calls llm generation with default hard-coded option values
	return w.generateWithOllama(ctx, "llama3.2:1b", prompt, OllamaOptions{
		Temperature: 0.1,
		NumPredict:  256,
	})
}

func (w *Worker) generateWithOllama(
	ctx context.Context,
	model string,
	prompt string,
	options OllamaOptions,
) (string, error) {

	client := &http.Client{
		Timeout: 500 * time.Second,
	}

	payload := OllamaRequest{
		Model:   model,
		Prompt:  prompt,
		Stream:  false,
		Options: options,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://llm_model:11434/api/generate",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", err
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

func (w *Worker) getLlmJobInfo(ctx context.Context, jobId int) ([]byte, error) {
	var inputJson []byte

	err := w.db.QueryRowContext(ctx, `
        SELECT input FROM llm_job_info WHERE job_id = $1
        `, jobId).Scan(&inputJson)
	if err != nil {
		return nil, err
	}

	return inputJson, nil
}
