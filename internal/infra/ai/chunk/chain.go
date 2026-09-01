package chunk

import (
	"context"

	"learn/internal/domain/document"
)

type Splitter struct{}

func New() *Splitter { return &Splitter{} }

var DefaultChainOrder = []string{"markdown", "heuristic", "recursive", "fixed"}

func DefaultConfig() document.SplitterConfig {
	return document.SplitterConfig{
		MaxChars:   1200,
		Overlap:    80,
		Language:   "zh",
		ChainOrder: DefaultChainOrder,
	}
}

func (s *Splitter) Split(_ context.Context, text string, cfg document.SplitterConfig) ([]document.Chunk, string, error) {
	chain := cfg.ChainOrder
	if len(chain) == 0 {
		chain = DefaultChainOrder
	}
	maxChars := cfg.MaxChars
	if maxChars <= 0 {
		maxChars = 1200
	}
	overlap := cfg.Overlap
	for _, level := range chain {
		var chunks []document.Chunk
		var header string
		var ok bool
		switch level {
		case "markdown":
			chunks, header = markdownSplit(text, maxChars, overlap)
			ok = len(chunks) > 0
		case "heuristic":
			chunks = heuristicSplit(text, maxChars, overlap)
			ok = len(chunks) > 0
		case "recursive":
			chunks = recursiveSplit(text, maxChars, overlap)
			ok = len(chunks) > 0
		case "fixed":
			chunks = fixedSplit(text, maxChars, overlap)
			ok = len(chunks) > 0
		}
		if ok {
			return chunks, header, nil
		}
	}
	return nil, "", nil
}

var _ document.Splitter = (*Splitter)(nil)
