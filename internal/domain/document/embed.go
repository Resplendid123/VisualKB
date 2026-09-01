package document

import "context"

// Output order aligns with inputs.
type Embedder interface {
	Dimension() int

	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}
