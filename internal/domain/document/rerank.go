package document

import "context"

// Index points back into candidates.
type RankedItem struct {
	Index int
	Score float64
}

// Returned order is final.
type Reranker interface {
	Rerank(ctx context.Context, query string, candidates []string) ([]RankedItem, error)
}
