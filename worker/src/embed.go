package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EmbedRequest struct {
	Texts []string `json:"texts"`
	Query string   `json:"query"`
}

type EmbedResponse struct {
	Distances []float32 `json:"distances"`
	Indices   []int64   `json:"indices"`
}

type Client struct {
	BaseURL string
	Client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) embedAndSearch(ctx context.Context, texts []string, query string) ([]float32, []int64, error) {

	reqBody := EmbedRequest{Texts: texts, Query: query}

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

	var out EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}

	return out.Distances, out.Indices, nil
}
