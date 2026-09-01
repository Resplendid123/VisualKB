package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"learn/internal/domain/conversation"
	"learn/internal/domain/document"
)

const (
	CreateDocumentToolName = "create_document"
	ReadDocumentToolName   = "read_document"
	EditDocumentToolName   = "edit_document"
	ListDocumentsToolName  = "list_documents"
	defaultEditTimeout     = 30 * time.Second
	defaultReadTimeout     = 10 * time.Second
	defaultListTimeout     = 10 * time.Second
	defaultListLimit       = 20
	maxListLimit           = 100
	maxEditOpsPerCall      = 16
)

type ReadDocumentTool struct {
	editor document.Editor
}

func NewReadDocumentTool(e document.Editor) *ReadDocumentTool {
	return &ReadDocumentTool{editor: e}
}

func (t *ReadDocumentTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: ReadDocumentToolName,
		Description: "Read the current content of a document owned by the user. " +
			"Returns the raw content of the latest version. " +
			"Use this before edit_document so the agent knows what anchor text to match.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"document_id": map[string]any{
					"type":        "integer",
					"description": "Document ID (documents.id).",
				},
			},
			"required": []string{"document_id"},
		},
	}
}

func (t *ReadDocumentTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	docID, ok := argsInt64(args, "document_id")
	if !ok || docID <= 0 {
		return conversation.Result{Error: "document_id is required (positive integer)"}
	}
	cctx, cancel := context.WithTimeout(ctx, defaultReadTimeout)
	defer cancel()
	userID, ok := conversation.UserIDFromContext(cctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}
	content, err := t.editor.ReadCurrent(cctx, userID, docID)
	if err != nil {
		return conversation.Result{Error: fmt.Sprintf("read_document failed: %v", err)}
	}
	return conversation.Result{Content: content}
}

func (t *ReadDocumentTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: true,
		Timeout:    defaultReadTimeout,
		Message: func(args map[string]any) string {
			id, _ := argsInt64(args, "document_id")
			return fmt.Sprintf("Reading document %d", id)
		},
	}
}

type EditDocumentTool struct {
	editor document.Editor
}

func NewEditDocumentTool(e document.Editor) *EditDocumentTool {
	return &EditDocumentTool{editor: e}
}

func (t *EditDocumentTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: EditDocumentToolName,
		Description: "Edit a document owned by the user by applying edit operations to its current content. " +
			"Each call writes a new version and returns the new version number. " +
			"This tool can also RENAME the document — pass a `title` to change it; " +
			"omit `title` to keep the current one. Use this whenever the user asks to " +
			"rename a note, fix its title, or update both title and body in one call. " +
			"`ops` runs in order on top of the previous op's output. " +
			"Supported op types: " +
			"'replace_anchor' (args: {anchor, new_text, replace_all?}; anchor must match exactly once unless replace_all=true), " +
			"'append' (args: {text, want_newline?}; want_newline=true adds a leading \\n if missing), " +
			"'whole_replace' (args: {content}; replaces the entire body). " +
			"Notes (markdown) are editable; knowledge documents are read-only.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"document_id": map[string]any{
					"type":        "integer",
					"description": "Document ID.",
				},
				"title": map[string]any{
					"type":        "string",
					"minLength":   1,
					"maxLength":   128,
					"description": "New title for the document. Pass this to rename the document; omit (or pass empty) to keep the current title unchanged.",
				},
				"ops": map[string]any{
					"type":        "array",
					"description": fmt.Sprintf("Edit operations applied in order; max %d per call.", maxEditOpsPerCall),
					"minItems":    1,
					"maxItems":    maxEditOpsPerCall,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
								"enum": []string{document.OpTypeReplaceAnchor, document.OpTypeAppend, document.OpTypeWholeReplace},
							},
							"args": map[string]any{
								"type":        "object",
								"description": "Op-specific arguments (see tool description).",
							},
						},
						"required": []string{"type", "args"},
					},
				},
			},
			"required": []string{"document_id"},
		},
	}
}

type opArgs struct {
	Type string         `json:"type"`
	Args map[string]any `json:"args"`
}

func (t *EditDocumentTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	docID, ok := argsInt64(args, "document_id")
	if !ok || docID <= 0 {
		return conversation.Result{Error: "document_id is required (positive integer)"}
	}
	rawOps, _ := args["ops"].([]any)

	title, _ := args["title"].(string)
	if len(rawOps) == 0 && strings.TrimSpace(title) == "" {
		return conversation.Result{Error: "ops must be non-empty, or pass a non-empty `title` to rename the document"}
	}
	if len(rawOps) > maxEditOpsPerCall {
		return conversation.Result{Error: fmt.Sprintf("ops exceeds max %d per call", maxEditOpsPerCall)}
	}
	parsed := make([]opArgs, 0, len(rawOps))
	for i, raw := range rawOps {
		blob, _ := json.Marshal(raw)
		var one opArgs
		if err := json.Unmarshal(blob, &one); err != nil {
			return conversation.Result{Error: fmt.Sprintf("ops[%d] invalid shape: %v", i, err)}
		}
		if one.Type == "" || one.Args == nil {
			return conversation.Result{Error: fmt.Sprintf("ops[%d] requires type and args", i)}
		}
		parsed = append(parsed, one)
	}

	ops := make([]document.EditOp, 0, len(parsed))
	for _, p := range parsed {
		op, err := document.ParseEditOp(p.Type, p.Args)
		if err != nil {
			return conversation.Result{Error: err.Error()}
		}
		ops = append(ops, op)
	}

	cctx, cancel := context.WithTimeout(ctx, defaultEditTimeout)
	defer cancel()
	userID, ok := conversation.UserIDFromContext(cctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}
	version, err := t.editor.ApplyEdits(cctx, userID, docID, ops, title)
	if err != nil {
		return conversation.Result{Error: fmt.Sprintf("edit_document failed: %v", err)}
	}

	out := struct {
		DocumentID int64 `json:"document_id"`
		Version    int   `json:"version"`
	}{DocumentID: docID, Version: version}
	content, _ := json.Marshal(out)
	return conversation.Result{Content: string(content)}
}

func (t *EditDocumentTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: false,
		Timeout:    defaultEditTimeout,
		Message: func(args map[string]any) string {
			id, _ := argsInt64(args, "document_id")
			return fmt.Sprintf("Editing document %d", id)
		},
	}
}

type ListDocumentsTool struct {
	lister document.Lister
}

func NewListDocumentsTool(l document.Lister) *ListDocumentsTool {
	return &ListDocumentsTool{lister: l}
}

func (t *ListDocumentsTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: ListDocumentsToolName,
		Description: "List metadata of documents owned by the current user. " +
			"Returns id, title, source, chunk status and timestamps; does NOT include content. " +
			"Use 'source' to narrow ('note' for personal notes, 'knowledge' for knowledge base; omit for both). " +
			"After locating the target document_id, call read_document to fetch its content.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type":        "string",
					"enum":        []string{"note", "knowledge"},
					"description": "Optional filter by source; omit to list both.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"minimum":     1,
					"maximum":     maxListLimit,
					"default":     defaultListLimit,
					"description": fmt.Sprintf("Max items to return (1-%d). Default %d.", maxListLimit, defaultListLimit),
				},
				"offset": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"default":     0,
					"description": "Pagination offset.",
				},
				"include_archived": map[string]any{
					"type":        "boolean",
					"default":     false,
					"description": "Include soft-archived documents; default false.",
				},
			},
		},
	}
}

type DocumentSummary struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Source      string  `json:"source"`
	Lang        string  `json:"lang,omitempty"`
	ChunkStatus int     `json:"chunk_status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   *string `json:"updated_at,omitempty"`
	ArchivedAt  *string `json:"archived_at,omitempty"`
}

type ListDocumentsResult struct {
	Items  []DocumentSummary `json:"items"`
	Total  int64             `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func (t *ListDocumentsTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	source, _ := args["source"].(string)
	if source != "" && source != "note" && source != "knowledge" {
		return conversation.Result{Error: "source must be 'note' or 'knowledge'"}
	}

	limit := defaultListLimit
	if _, hasLimit := args["limit"]; hasLimit {
		n, ok := argsInt64(args, "limit")
		if !ok {
			return conversation.Result{Error: "limit must be a positive integer"}
		}
		limit = int(n)
	}
	if limit < 1 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	offset := 0
	if _, hasOffset := args["offset"]; hasOffset {
		n, ok := argsInt64(args, "offset")
		if !ok || n < 0 {
			return conversation.Result{Error: "offset must be a non-negative integer"}
		}
		offset = int(n)
	}

	includeArchived := false
	if v, ok := args["include_archived"].(bool); ok {
		includeArchived = v
	}

	cctx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()
	userID, ok := conversation.UserIDFromContext(cctx)
	if !ok {
		return conversation.Result{Error: "list context missing user_id"}
	}

	docs, total, err := t.lister.List(cctx, userID, document.ListOpts{
		Source:          source,
		Limit:           limit,
		Offset:          offset,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		return conversation.Result{Error: fmt.Sprintf("list_documents failed: %v", err)}
	}

	out := ListDocumentsResult{
		Items:  make([]DocumentSummary, 0, len(docs)),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	for _, d := range docs {
		item := DocumentSummary{
			ID:          d.ID,
			Title:       d.Title,
			Source:      d.Source,
			Lang:        d.Lang,
			ChunkStatus: int(d.ChunkStatus),
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		}
		if d.UpdatedAt != nil {
			s := d.UpdatedAt.Format(time.RFC3339)
			item.UpdatedAt = &s
		}
		if d.ArchivedAt != nil {
			s := d.ArchivedAt.Format(time.RFC3339)
			item.ArchivedAt = &s
		}
		out.Items = append(out.Items, item)
	}

	content, mErr := json.Marshal(out)
	if mErr != nil {
		return conversation.Result{Error: mErr.Error()}
	}
	return conversation.Result{Content: string(content)}
}

func (t *ListDocumentsTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: true,
		Timeout:    defaultListTimeout,
		Message: func(args map[string]any) string {
			src, _ := args["source"].(string)
			if src == "" {
				return "Listing documents"
			}
			return fmt.Sprintf("Listing %s documents", src)
		},
	}
}

func argsInt64(args map[string]any, key string) (int64, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}

type CreateDocumentTool struct {
	creator document.Creator
}

func NewCreateDocumentTool(c document.Creator) *CreateDocumentTool {
	return &CreateDocumentTool{creator: c}
}

func (t *CreateDocumentTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: CreateDocumentToolName,
		Description: "Create a new personal note document owned by the current user. " +
			"Returns the new document_id. Use this when the conversation produces content worth keeping " +
			"(a summary, a how-to, a piece of analysis) and the user hasn't already pasted a knowledge doc. " +
			"Only supports 'note' source — knowledge documents are ingested from uploads, not authored by the agent.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"minLength":   1,
					"maxLength":   128,
					"description": "Document title; will be trimmed. Required.",
				},
				"content": map[string]any{
					"type":        "string",
					"minLength":   1,
					"maxLength":   200_000,
					"description": "Initial body content in markdown; required (empty content is rejected).",
				},
			},
			"required": []string{"title", "content"},
		},
	}
}

type createDocumentResult struct {
	DocumentID int64 `json:"document_id"`
}

func (t *CreateDocumentTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	if strings.TrimSpace(title) == "" {
		return conversation.Result{Error: "title is required (non-empty after trim)"}
	}
	if content == "" {
		return conversation.Result{Error: "content is required (non-empty)"}
	}
	cctx, cancel := context.WithTimeout(ctx, defaultEditTimeout)
	defer cancel()
	userID, ok := conversation.UserIDFromContext(cctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}
	docID, err := t.creator.CreateText(cctx, userID, document.CreateParams{
		Source:  "note",
		Title:   title,
		Content: content,
	})
	if err != nil {
		return conversation.Result{Error: fmt.Sprintf("create_document failed: %v", err)}
	}
	out, _ := json.Marshal(createDocumentResult{DocumentID: docID})
	return conversation.Result{Content: string(out)}
}

func (t *CreateDocumentTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: false,
		Timeout:    defaultEditTimeout,
		Message: func(args map[string]any) string {
			title, _ := args["title"].(string)
			if title == "" {
				return "Creating note"
			}
			return fmt.Sprintf("Creating note %q", title)
		},
	}
}

func RegisterDocumentTools(c document.Creator, e document.Editor, l document.Lister) {
	Default.MustRegister(NewCreateDocumentTool(c))
	Default.MustRegister(NewReadDocumentTool(e))
	Default.MustRegister(NewEditDocumentTool(e))
	Default.MustRegister(NewListDocumentsTool(l))
}

var (
	_ conversation.Tool = (*ReadDocumentTool)(nil)
	_ conversation.Tool = (*EditDocumentTool)(nil)
	_ conversation.Tool = (*ListDocumentsTool)(nil)
)
