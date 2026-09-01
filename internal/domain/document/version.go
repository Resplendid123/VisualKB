package document

import "time"

// documents.current_version_id points to the active version.
type DocumentVersion struct {
	ID         int64
	DocumentID int64
	Version    int
	Title      string
	FileKey    string
	FileSize   int64
	FileHash   *string
	CreatedAt  time.Time
	UpdatedAt  *time.Time
}
