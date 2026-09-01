package document

import "context"

// Service layer guards version-number concurrency.
type DocumentVersionRepo interface {
	FindCurrent(ctx context.Context, doc *Document) (*DocumentVersion, error)
	NextVersionNumber(ctx context.Context, documentID int64) (int, error)
	Create(ctx context.Context, ver *DocumentVersion) error
	UpdateFileKey(ctx context.Context, versionID int64, fileKey string) error
	ListByDocument(ctx context.Context, documentID int64) ([]*DocumentVersion, error)
}
