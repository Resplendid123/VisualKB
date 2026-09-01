package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"learn/internal/domain/document"
	"learn/internal/infra/config"
)

type openAIEmbedReq struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbedResp struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

type OpenAIEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	http    *http.Client
}

func New(cfg config.EmbeddingConfig, timeout time.Duration) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.ApiKey,
		model:   cfg.Model,
		dim:     cfg.Dimension,
		http:    &http.Client{Timeout: timeout},
	}
}

func (e *OpenAIEmbedder) Dimension() int { return e.dim }

func (e *OpenAIEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	req := openAIEmbedReq{Input: inputs, Model: e.model}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal embed: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed status %d body %s", resp.StatusCode, b)
	}
	var raw openAIEmbedResp
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode embed: %w", err)
	}
	out := make([][]float32, len(inputs))
	for _, d := range raw.Data {
		out[d.Index] = d.Embedding
	}
	return out, nil
}

var _ document.Embedder = (*OpenAIEmbedder)(nil)
