package document

import "context"

type Ingestor interface {
	Ingest(ctx context.Context, documentID int64) error
}
