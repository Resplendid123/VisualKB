package document

import "context"

// Content is the parent chunk after small-to-big.
type Hit struct {
	ChunkID    int64
	ParentID   int64
	DocumentID int64
	Title      string
	Source     string
	Content    string
	Score      float64
	Header     string
	Embedding  []float32
}

type Searcher interface {
	Hybrid(ctx context.Context, userID int64, q string, topN int) ([]Hit, error)
}
