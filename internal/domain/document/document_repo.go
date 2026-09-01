package document

import "context"

// Folder filtering handled by adjacency table.
type ListOpts struct {
	Source          string // "" | "note" | "knowledge"
	ChunkStatus     *ChunkStatus
	IncludeArchived bool
	Limit           int
	Offset          int
}

// DB fills ID; Archive checks owner.
type DocumentRepo interface {
	ListDirty(ctx context.Context, limit int) ([]int64, error)
	ListUnchunkedByUser(ctx context.Context, userID int64, source string) ([]int64, error)
	ListByUser(ctx context.Context, userID int64, opts ListOpts) ([]*Document, int64, error)
	FindChunkStatus(ctx context.Context, id int64) (ChunkStatus, error)
	FindByID(ctx context.Context, id int64) (*Document, error)
	Create(ctx context.Context, doc *Document) error
	UpdateCurrentVersion(ctx context.Context, documentID, versionID int64) error
	UpdateTitleAndLang(ctx context.Context, documentID int64, title, lang string) error
	Archive(ctx context.Context, documentID, userID int64) error
	MarkDirty(ctx context.Context, id int64) error
	MarkChunked(ctx context.Context, id int64) error
	MarkError(ctx context.Context, id int64) error
}
