package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"learn/internal/domain/conversation"
	"learn/internal/domain/document"
)

const (
	RetrieveToolName = "search_kb"
	defaultTopK      = 8
	maxTopK          = 30
	defaultTimeout   = 30 * time.Second
)

type RetrieveTool struct {
	searcher document.Searcher
}

func NewRetrieveTool(s document.Searcher) *RetrieveTool {
	return &RetrieveTool{searcher: s}
}

func (t *RetrieveTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: RetrieveToolName,
		Description: "Search the user's knowledge base for content relevant to the query. " +
			"Returns up to `top_k` hits, each with title, source, score and a content excerpt. " +
			"Use this before answering questions about notes, documents, or any factual content " +
			"the user may have ingested. Prefer multiple narrower queries over one broad query.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Natural-language search query;rewrite user intent into a self-contained question if needed.",
				},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "Number of hits to return (1-30). Default 8.",
					"minimum":     1,
					"maximum":     maxTopK,
				},
			},
			"required": []string{"query"},
		},
	}
}

func (t *RetrieveTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	query, _ := args["query"].(string)
	if query == "" {
		return conversation.Result{Error: "query is required"}
	}
	topK := defaultTopK
	if v, ok := args["top_k"]; ok {
		switch n := v.(type) {
		case float64:
			topK = int(n)
		case int:
			topK = n
		}
	}
	if topK < 1 {
		topK = defaultTopK
	}
	if topK > maxTopK {
		topK = maxTopK
	}

	cctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	userID, ok := conversation.UserIDFromContext(cctx)
	if !ok {
		return conversation.Result{Error: "search context missing user_id"}
	}

	hits, err := t.searcher.Hybrid(cctx, userID, query, topK)
	if err != nil {
		return conversation.Result{Error: fmt.Sprintf("search failed: %v", err)}
	}
	return conversation.Result{Content: formatHits(hits)}
}

func (t *RetrieveTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: true,
		Timeout:    defaultTimeout,
		Message: func(args map[string]any) string {
			q, _ := args["query"].(string)
			if q == "" {
				return "Searching knowledge base"
			}
			return fmt.Sprintf("Searching: %s", truncate(q, 60))
		},
	}
}

func formatHits(hits []document.Hit) string {
	if len(hits) == 0 {
		return "(no results)"
	}
	var sb strings.Builder
	for i, h := range hits {
		fmt.Fprintf(&sb, "[%d] %s (source=%s, score=%.3f)\n", i+1, h.Title, h.Source, h.Score)
		if h.Header != "" {
			fmt.Fprintf(&sb, "    section: %s\n", h.Header)
		}

		fmt.Fprintf(&sb, "    %s\n\n", oneLine(h.Content, 600))
	}
	return sb.String()
}

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		s = s[:n] + "…"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func MarshalArgs(args map[string]any) string {
	b, _ := json.Marshal(args)
	return string(b)
}

func RegisterRetrieveTool(s document.Searcher) {
	Default.MustRegister(NewRetrieveTool(s))
}

var _ conversation.Tool = (*RetrieveTool)(nil)
