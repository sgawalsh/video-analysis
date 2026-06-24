package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SemanticSearchRequest struct {
	Texts []string
	Query string
}

type SemanticSearchResponse struct {
	Distances []float32
	Indices   []int64
}

type GetEmbeddingRequest struct {
	Texts      []string
	StartTimes []int64
}

type GetEmbeddingResponse struct {
	Boundaries []int
	Chapters   []Chapter
}

type Chapter struct {
	ID    int64
	Title string
	Text  string
}
type Client struct {
	BaseURL string
	Client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) embedAndSearch(ctx context.Context, texts []string, query string) ([]float32, []int64, error) {

	reqBody := SemanticSearchRequest{Texts: texts, Query: query}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/embedAndSearch", c.BaseURL),
		bytes.NewBuffer(b),
	)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var out SemanticSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}

	return out.Distances, out.Indices, nil
}

func (c *Client) getTopicWindows(ctx context.Context, texts []string) ([]int, []Chapter, error) {

	reqBody := GetEmbeddingRequest{Texts: texts}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/getTopicWindows", c.BaseURL),
		bytes.NewBuffer(b),
	)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var out GetEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}

	return out.Boundaries, out.Chapters, nil
}
