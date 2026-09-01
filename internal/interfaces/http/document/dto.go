package document

import (
	"time"

	"learn/internal/domain/document"
)

type createRequest struct {
	Source       string `json:"source"        binding:"required"`
	Title        string `json:"title"         binding:"required"`
	Lang         string `json:"lang"`
	ContentType  string `json:"content_type"`
	ParentTreeID *int64 `json:"parent_tree_id"`
	Content      string `json:"content"       binding:"required"`
}

type createResponse struct {
	DocumentID int64 `json:"document_id"`
}

type uploadResponse struct {
	DocumentID int64 `json:"document_id"`
}

// addVersionRequest excludes the version field.
type addVersionRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}

type addVersionResponse struct {
	DocumentID int64 `json:"document_id"`
	Version    int   `json:"version"`
}

type ingestOneResponse struct {
	DocumentID int64 `json:"document_id"`
}

type ingestAllResponse struct {
	Enqueued int `json:"enqueued"`
}

// documentItem is a list/detail row.
type documentItem struct {
	ID               int64                `json:"id"`
	Title            string               `json:"title"`
	Source           string               `json:"source"`
	Lang             string               `json:"lang"`
	ContentType      document.ContentType `json:"content_type"`
	ChunkStatus      int8                 `json:"chunk_status"`
	CurrentVersionID *int64               `json:"current_version_id"`
	ArchivedAt       *time.Time           `json:"archived_at"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        *time.Time           `json:"updated_at"`
}

type listResponse struct {
	Items []*documentItem `json:"items"`
	Total int64           `json:"total"`
}

type archiveResponse struct {
	DocumentID int64 `json:"document_id"`
}

// documentDetail is the GET document response.
type documentDetail struct {
	ID               int64                `json:"id"`
	Title            string               `json:"title"`
	Source           string               `json:"source"`
	Lang             string               `json:"lang"`
	ContentType      document.ContentType `json:"content_type"`
	ChunkStatus      int8                 `json:"chunk_status"`
	CurrentVersionID *int64               `json:"current_version_id"`
	ArchivedAt       *time.Time           `json:"archived_at"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        *time.Time           `json:"updated_at"`
	Content          string               `json:"content"`
}

// moveRequest allows nil parent_tree_id.
type moveRequest struct {
	ParentTreeID *int64 `json:"parent_tree_id"`
}

type moveResponse struct {
	DocumentID   int64  `json:"document_id"`
	ParentNodeID *int64 `json:"parent_node_id"`
}

// patchRequest allows optional title.
type patchRequest struct {
	Title string        `json:"title"`
	Ops   []patchOpBody `json:"ops" binding:"required"`
}

// patchOpBody dispatches by type.
type patchOpBody struct {
	Type string         `json:"type" binding:"required"` // replace_anchor | append | whole_replace
	Args map[string]any `json:"args"`
}

type patchResponse struct {
	DocumentID int64 `json:"document_id"`
	Version    int   `json:"version"`
}
