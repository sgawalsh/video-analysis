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
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
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

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {

	reqBody := EmbedRequest{Texts: texts}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/embed", c.BaseURL),
		bytes.NewBuffer(b),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var out EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out.Embeddings, nil
}