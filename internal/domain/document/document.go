package document

import (
	"time"

	"learn/internal/domain"
)

type ChunkStatus int8

const (
	ChunkStatusDirty   ChunkStatus = 0 // pending
	ChunkStatusChunked ChunkStatus = 1 // chunked; retrievable
	ChunkStatusError   ChunkStatus = 2 // split failed
)

type ContentType string

const (
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypePDF      ContentType = "pdf"
)

const defaultContentType = ContentTypeMarkdown

func ValidContentTypes() map[ContentType]bool {
	return map[ContentType]bool{
		ContentTypeMarkdown: true,
		ContentTypePDF:      true,
	}
}

func NormalizeContentType(ct ContentType) ContentType {
	if ct == "" {
		return defaultContentType
	}
	return ct
}

func ValidateContentType(ct ContentType) error {
	if ct == "" {
		return nil
	}
	if !ValidContentTypes()[ct] {
		return domain.ErrDocumentInvalidContentType
	}
	return nil
}

// Notes allow markdown only; knowledge any whitelisted.
func ValidateSourceContentMatch(source string, ct ContentType) error {
	if err := ValidateContentType(ct); err != nil {
		return err
	}
	if source == "note" && ct != ContentTypeMarkdown {
		return domain.ErrDocumentNoteMarkdownOnly
	}
	return nil
}

type CreateParams struct {
	Source       string // "note" | "knowledge"
	Title        string
	Lang         string      // optional, default "zh"
	ContentType  ContentType // optional, default markdown
	ParentTreeID *int64      // knowledge only
	Content      string      // required
}

// Location lives in knowledge_tree adjacency table.
type Document struct {
	ID               int64
	CurrentVersionID *int64
	UserID           int64
	Source           string
	Title            string
	Lang             string
	ContentType      ContentType
	ChunkStatus      ChunkStatus
	ArchivedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        *time.Time
}
