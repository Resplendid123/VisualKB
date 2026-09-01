package document

import "context"

type ChunkIndex int

type Chunk struct {
	Index   ChunkIndex
	Content string
	Header  string
}

type SplitterConfig struct {
	MaxChars   int
	Overlap    int
	Language   string
	ChainOrder []string
}

// Single-layer split.
type Splitter interface {
	Split(ctx context.Context, text string, cfg SplitterConfig) ([]Chunk, string, error)
}
