package document

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	app "learn/internal/application/document"
	"learn/internal/domain/document"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/response"
)

// DocumentHandler exposes document CRUD and ingest.
type DocumentHandler struct {
	docSvc *app.DocumentService
}

func NewDocumentHandler(docSvc *app.DocumentService) *DocumentHandler {
	return &DocumentHandler{docSvc: docSvc}
}

// Create creates a new document.
func (h *DocumentHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	id, err := h.docSvc.CreateText(c.Request.Context(), userID, app.CreateParams{
		Source:       req.Source,
		Title:        req.Title,
		Lang:         req.Lang,
		ContentType:  document.ContentType(req.ContentType),
		ParentTreeID: req.ParentTreeID,
		Content:      req.Content,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(201, response.Response{
		Code:    response.CodeSuccess,
		Message: "success",
		Data:    createResponse{DocumentID: id},
	})
}

func (h *DocumentHandler) UploadFile(c *gin.Context) {
	userID := middleware.IdentityFrom(c).UserID
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		response.AbortBadRequest(c, fmt.Sprintf("parse multipart: %v", err))
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		response.AbortBadRequest(c, "title is required")
		return
	}
	lang := c.PostForm("lang")
	var parentTreeID *int64
	if raw := strings.TrimSpace(c.PostForm("parent_tree_id")); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v <= 0 {
			response.AbortBadRequest(c, "parent_tree_id must be a positive integer")
			return
		}
		parentTreeID = &v
	}
	header, err := c.FormFile("file")
	if err != nil {
		response.AbortBadRequest(c, "file is required (multipart field 'file')")
		return
	}

	mime := header.Header.Get("Content-Type")
	extOK := strings.EqualFold(filepath.Ext(header.Filename), ".pdf")
	mimeOK := mime == "application/pdf" || mime == "application/octet-stream"
	if !extOK && !mimeOK {
		response.AbortBadRequest(c, "only PDF files are accepted (application/pdf or .pdf)")
		return
	}
	if header.Size > app.UploadMaxBytes {
		response.AbortBadRequest(c, fmt.Sprintf("file too large: %d > %d bytes", header.Size, app.UploadMaxBytes))
		return
	}
	src, err := header.Open()
	if err != nil {
		response.Fail(c, fmt.Errorf("open uploaded file: %w", err))
		return
	}
	defer func() { _ = src.Close() }()
	raw, err := io.ReadAll(src)
	if err != nil {
		response.Fail(c, fmt.Errorf("read uploaded file: %w", err))
		return
	}
	id, err := h.docSvc.CreateFile(c.Request.Context(), userID, title, lang, parentTreeID, raw, "application/pdf")
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(201, response.Response{
		Code:    response.CodeSuccess,
		Message: "success",
		Data:    uploadResponse{DocumentID: id},
	})
}

func (h *DocumentHandler) ServeFile(c *gin.Context) {
	docID, ok := parseDocID(c)
	if !ok {
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	body, contentType, filename, err := h.docSvc.GetFile(c.Request.Context(), userID, docID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer func() { _ = body.Close() }()
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	// Block MIME sniffing for iframe PDFs.
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(200)
	if _, err := io.Copy(c.Writer, body); err != nil {
		// Headers already sent; just log.
		slog.ErrorContext(c.Request.Context(), "stream pdf body failed",
			"document_id", docID, "err", err)
	}
}

// AddVersion appends a new version.
func (h *DocumentHandler) AddVersion(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || docID <= 0 {
		response.AbortBadRequest(c, "invalid document id")
		return
	}
	var req addVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	version, err := h.docSvc.AddVersion(c.Request.Context(), userID, docID, req.Title, req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(201, response.Response{
		Code:    response.CodeSuccess,
		Message: "success",
		Data:    addVersionResponse{DocumentID: docID, Version: version},
	})
}

// IngestOne triggers ingest for one document.
func (h *DocumentHandler) IngestOne(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || docID <= 0 {
		response.AbortBadRequest(c, "invalid document id")
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.docSvc.IngestOne(c.Request.Context(), userID, docID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ingestOneResponse{DocumentID: docID})
}

// IngestAll triggers ingest by source.
func (h *DocumentHandler) IngestAll(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		response.AbortBadRequest(c, "source is required (note | knowledge)")
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	n, err := h.docSvc.IngestAll(c.Request.Context(), userID, source)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ingestAllResponse{Enqueued: n})
}

func (h *DocumentHandler) List(c *gin.Context) {
	userID := middleware.IdentityFrom(c).UserID
	opts := document.ListOpts{
		Source:          c.Query("source"),
		IncludeArchived: c.Query("include_archived") == "true",
	}
	if cs := c.Query("chunk_status"); cs != "" {
		v, err := strconv.ParseInt(cs, 10, 8)
		if err != nil || v < 0 || v > 2 {
			response.AbortBadRequest(c, "chunk_status must be 0/1/2")
			return
		}
		st := document.ChunkStatus(int8(v))
		opts.ChunkStatus = &st
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil {
		opts.Limit = v
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil {
		opts.Offset = v
	}
	rows, total, err := h.docSvc.List(c.Request.Context(), userID, opts)
	if err != nil {
		response.Fail(c, err)
		return
	}
	items := make([]*documentItem, len(rows))
	for i, r := range rows {
		items[i] = toDocumentItem(r)
	}
	response.OK(c, listResponse{Items: items, Total: total})
}

// Move reparents a knowledge document.
func (h *DocumentHandler) Move(c *gin.Context) {
	docID, ok := parseDocID(c)
	if !ok {
		return
	}
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.docSvc.Move(c.Request.Context(), userID, docID, req.ParentTreeID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, moveResponse{DocumentID: docID, ParentNodeID: req.ParentTreeID})
}

// Archive soft-deletes a document.
func (h *DocumentHandler) Archive(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || docID <= 0 {
		response.AbortBadRequest(c, "invalid document id")
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.docSvc.Archive(c.Request.Context(), userID, docID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, archiveResponse{DocumentID: docID})
}

// Get returns document detail with content.
func (h *DocumentHandler) Get(c *gin.Context) {
	docID, ok := parseDocID(c)
	if !ok {
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	doc, err := h.docSvc.Get(c.Request.Context(), userID, docID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	content, err := h.docSvc.ReadCurrent(c.Request.Context(), userID, docID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toDocumentDetail(doc, content))
}

// Patch edits via ops, saves new version.
func (h *DocumentHandler) Patch(c *gin.Context) {
	docID, ok := parseDocID(c)
	if !ok {
		return
	}
	var req patchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	ops := make([]document.EditOp, 0, len(req.Ops))
	for _, o := range req.Ops {
		op, err := document.ParseEditOp(o.Type, o.Args)
		if err != nil {
			response.Fail(c, err)
			return
		}
		ops = append(ops, op)
	}
	userID := middleware.IdentityFrom(c).UserID
	version, err := h.docSvc.ApplyEdits(c.Request.Context(), userID, docID, ops, req.Title)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, patchResponse{DocumentID: docID, Version: version})
}

func parseDocID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || docID <= 0 {
		response.AbortBadRequest(c, "invalid document id")
		return 0, false
	}
	return docID, true
}

func toDocumentItem(d *document.Document) *documentItem {
	return &documentItem{
		ID:               d.ID,
		Title:            d.Title,
		Source:           d.Source,
		Lang:             d.Lang,
		ContentType:      d.ContentType,
		ChunkStatus:      int8(d.ChunkStatus),
		CurrentVersionID: d.CurrentVersionID,
		ArchivedAt:       d.ArchivedAt,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func toDocumentDetail(d *document.Document, content string) *documentDetail {
	return &documentDetail{
		ID:               d.ID,
		Title:            d.Title,
		Source:           d.Source,
		Lang:             d.Lang,
		ContentType:      d.ContentType,
		ChunkStatus:      int8(d.ChunkStatus),
		CurrentVersionID: d.CurrentVersionID,
		ArchivedAt:       d.ArchivedAt,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
		Content:          content,
	}
}
