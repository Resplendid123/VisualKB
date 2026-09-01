package rerank

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"learn/internal/domain/document"
	"learn/internal/infra/ai/llm"
)

type LLM struct {
	client *llm.OpenAI
	model  string
}

func New(c *llm.OpenAI, model string) *LLM {
	if model == "" {
		model = "qwen3-reranker-0.6b"
	}
	return &LLM{client: c, model: model}
}

func (r *LLM) Rerank(ctx context.Context, query string, candidates []string) ([]document.RankedItem, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	prompt := buildRerankPrompt(query, candidates)
	out, err := r.client.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("rerank llm: %w", err)
	}
	raw := strings.TrimSpace(out)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var resp []struct {
		Index int     `json:"Index"`
		Score float64 `json:"Score"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w (raw=%s)", err, out)
	}
	items := make([]document.RankedItem, 0, len(resp))
	for _, r := range resp {
		items = append(items, document.RankedItem{Index: r.Index, Score: r.Score})
	}
	return items, nil
}

func buildRerankPrompt(query string, cands []string) string {
	var sb strings.Builder
	sb.WriteString("对下列候选文本按与查询的相关度从高到低排序,仅输出 JSON 数组,每项形如 {\"Index\":<int>,\"Score\":<0-1>},不要其他说明:\n")
	sb.WriteString("查询:")
	sb.WriteString(query)
	sb.WriteString("\n候选(按编号):\n")
	for i, c := range cands {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i, truncate(c, 500)))
	}
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var _ document.Reranker = (*LLM)(nil)
