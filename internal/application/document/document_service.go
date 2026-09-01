package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"learn/internal/domain"
	"learn/internal/domain/document"
	domainS3 "learn/internal/domain/s3"
)

type DocumentService struct {
	docRepo     document.DocumentRepo
	versionRepo document.DocumentVersionRepo
	tree        document.TreeRepo
	store       domainS3.ObjectStore
	ingestor    document.Ingestor
}

func NewDocumentService(
	docRepo document.DocumentRepo,
	versionRepo document.DocumentVersionRepo,
	tree document.TreeRepo,
	store domainS3.ObjectStore,
	ingestor document.Ingestor,
) *DocumentService {
	return &DocumentService{docRepo: docRepo, versionRepo: versionRepo, tree: tree, store: store, ingestor: ingestor}
}

var extByContentType = map[document.ContentType]string{
	document.ContentTypeMarkdown: "md",
	document.ContentTypePDF:      "pdf",
}

func extFor(ct document.ContentType) string {
	if e, ok := extByContentType[ct]; ok {
		return e
	}
	return "md"
}

func noteObjectKey(docID int64, version int, ct document.ContentType) string {
	return fmt.Sprintf("documents/%d/v%d.%s", docID, version, extFor(ct))
}

func knowledgeObjectKey(docID int64, version int, ct document.ContentType) string {
	return fmt.Sprintf("documents/knowledge/%d/v%d.%s", docID, version, extFor(ct))
}

type CreateParams = document.CreateParams

const uploadMaxBytes = 50 << 20 // 50 MB

const UploadMaxBytes = uploadMaxBytes

func (s *DocumentService) CreateText(ctx context.Context, userID int64, p CreateParams) (int64, error) {
	return s.createInternal(ctx, userID, p, []byte(p.Content), "text/markdown")
}

// CreateFile stores raw bytes to S3.
func (s *DocumentService) CreateFile(
	ctx context.Context,
	userID int64,
	title, lang string,
	parentTreeID *int64,
	raw []byte,
	contentType string,
) (int64, error) {
	if len(raw) == 0 {
		return 0, domain.ErrDocumentEmptyContent
	}
	if int64(len(raw)) > uploadMaxBytes {
		return 0, fmt.Errorf("uploaded file too large: %d > %d bytes", len(raw), uploadMaxBytes)
	}
	if contentType != "application/pdf" {
		return 0, fmt.Errorf("unsupported upload content type %q (only application/pdf is accepted here)", contentType)
	}
	p := CreateParams{
		Source:       "knowledge",
		Title:        title,
		Lang:         lang,
		ContentType:  document.ContentTypePDF,
		ParentTreeID: parentTreeID,
	}
	return s.createInternal(ctx, userID, p, raw, contentType)
}

func (s *DocumentService) createInternal(
	ctx context.Context,
	userID int64,
	p CreateParams,
	raw []byte,
	mediaType string,
) (int64, error) {
	src := strings.TrimSpace(p.Source)
	title := strings.TrimSpace(p.Title)
	lang := strings.TrimSpace(p.Lang)
	if err := validateSource(src); err != nil {
		return 0, err
	}
	if title == "" {
		return 0, domain.ErrDocumentEmptyTitle
	}
	if len(raw) == 0 {
		return 0, domain.ErrDocumentEmptyContent
	}
	ct := document.NormalizeContentType(p.ContentType)
	if err := document.ValidateSourceContentMatch(src, ct); err != nil {
		return 0, err
	}
	if lang == "" {
		lang = "zh"
	}
	if src == "knowledge" && p.ParentTreeID != nil {
		parent, err := s.tree.FindByID(ctx, *p.ParentTreeID)
		if err != nil {
			return 0, err
		}
		if !parent.IsFolder {
			return 0, domain.ErrTreeNodeNotFolder
		}
	}
	doc := &document.Document{
		UserID:      userID,
		Source:      src,
		Title:       title,
		Lang:        lang,
		ContentType: ct,
		ChunkStatus: document.ChunkStatusDirty,
	}
	if err := s.docRepo.Create(ctx, doc); err != nil {
		return 0, err
	}
	var key string
	switch src {
	case "knowledge":
		key = knowledgeObjectKey(doc.ID, 1, ct)
	default:
		key = noteObjectKey(doc.ID, 1, ct)
	}
	if err := s.uploadContent(ctx, key, raw, mediaType); err != nil {
		return 0, err
	}
	ver := &document.DocumentVersion{
		DocumentID: doc.ID,
		Version:    1,
		Title:      title,
		FileKey:    key,
		FileSize:   int64(len(raw)),
	}
	if err := s.versionRepo.Create(ctx, ver); err != nil {
		return 0, err
	}
	if err := s.docRepo.UpdateCurrentVersion(ctx, doc.ID, ver.ID); err != nil {
		return 0, err
	}
	if src == "knowledge" {
		docID := doc.ID
		if _, err := s.tree.Insert(ctx, &document.TreeNode{
			UserID:   userID,
			ParentID: p.ParentTreeID,
			IsFolder: false,
			DocID:    &docID,
		}); err != nil {
			return 0, err
		}
	}
	s.maybeIngestOnWrite(src, doc.ID)
	return doc.ID, nil
}

func (s *DocumentService) AddVersion(
	ctx context.Context,
	userID, docID int64,
	title, content string,
) (int, error) {
	title = strings.TrimSpace(title)
	if content == "" {
		return 0, domain.ErrDocumentEmptyContent
	}
	doc, err := s.docRepo.FindByID(ctx, docID)
	if err != nil {
		return 0, err
	}
	if doc.UserID != userID {
		return 0, domain.ErrDocumentForbidden
	}
	if doc.Source == "knowledge" {
		return 0, domain.ErrDocumentNotEditable
	}
	return s.writeNewVersion(ctx, doc, content, title)
}

func (s *DocumentService) writeNewVersion(
	ctx context.Context,
	doc *document.Document,
	content, title string,
) (int, error) {

	sum := sha256.Sum256([]byte(content))
	newHash := hex.EncodeToString(sum[:])
	if title == "" || strings.TrimSpace(title) == doc.Title {
		if cur, err := s.versionRepo.FindCurrent(ctx, doc); err == nil {
			if cur.FileHash != nil && *cur.FileHash == newHash {
				return cur.Version, nil
			}
		} else if !errors.Is(err, domain.ErrDocumentVersionNotFound) {
			return 0, err
		}
	}

	nextV, err := s.versionRepo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return 0, err
	}
	var key string
	switch doc.Source {
	case "knowledge":
		key = knowledgeObjectKey(doc.ID, nextV, doc.ContentType)
	default:
		key = noteObjectKey(doc.ID, nextV, doc.ContentType)
	}
	if err := s.uploadContent(ctx, key, []byte(content), "text/markdown"); err != nil {
		return 0, err
	}
	verTitle := title
	if verTitle == "" {
		verTitle = doc.Title
	}
	ver := &document.DocumentVersion{
		DocumentID: doc.ID,
		Version:    nextV,
		Title:      verTitle,
		FileKey:    key,
		FileSize:   int64(len(content)),
		FileHash:   &newHash,
	}
	if err := s.versionRepo.Create(ctx, ver); err != nil {
		return 0, err
	}
	if err := s.docRepo.UpdateCurrentVersion(ctx, doc.ID, ver.ID); err != nil {
		return 0, err
	}
	if err := s.docRepo.MarkDirty(ctx, doc.ID); err != nil {
		return 0, err
	}
	if title != "" {
		if err := s.docRepo.UpdateTitleAndLang(ctx, doc.ID, title, doc.Lang); err != nil {
			return 0, err
		}
	}
	s.maybeIngestOnWrite(doc.Source, doc.ID)
	return nextV, nil
}

func (s *DocumentService) maybeIngestOnWrite(source string, documentID int64) {
	if source != "knowledge" {
		return
	}
	go s.runOne(documentID)
}

func (s *DocumentService) uploadContent(ctx context.Context, key string, raw []byte, mediaType string) error {
	if err := s.store.Put(ctx, key, bytes.NewReader(raw), int64(len(raw)), mediaType); err != nil {
		return fmt.Errorf("%w: put %s: %v", domain.ErrDocumentUploadFailed, key, err)
	}
	return nil
}

func validateSource(src string) error {
	switch src {
	case "note", "knowledge":
		return nil
	}
	return domain.ErrDocumentInvalidSource
}

func (s *DocumentService) ListDirty(ctx context.Context, limit int) ([]int64, error) {
	return s.docRepo.ListDirty(ctx, limit)
}

func (s *DocumentService) Get(ctx context.Context, userID, docID int64) (*document.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc.UserID != userID {
		return nil, domain.ErrDocumentForbidden
	}
	return doc, nil
}

func (s *DocumentService) ReadCurrent(ctx context.Context, userID, docID int64) (string, error) {
	doc, err := s.docRepo.FindByID(ctx, docID)
	if err != nil {
		return "", err
	}
	if doc.UserID != userID {
		return "", domain.ErrDocumentForbidden
	}
	ver, err := s.versionRepo.FindCurrent(ctx, doc)
	if err != nil {
		return "", err
	}
	rc, err := s.store.Get(ctx, ver.FileKey)
	if err != nil {
		return "", fmt.Errorf("%w: get %s: %v", domain.ErrDocumentFetchFailed, ver.FileKey, err)
	}
	defer func() { _ = rc.Close() }()
	buf, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("%w: read %s: %v", domain.ErrDocumentFetchFailed, ver.FileKey, err)
	}
	return string(buf), nil
}

func (s *DocumentService) GetFile(
	ctx context.Context,
	userID, docID int64,
) (io.ReadCloser, string, string, error) {
	doc, err := s.Get(ctx, userID, docID)
	if err != nil {
		return nil, "", "", err
	}
	if doc.ContentType != document.ContentTypePDF {
		return nil, "", "", domain.ErrDocumentNotEditable
	}
	ver, err := s.versionRepo.FindCurrent(ctx, doc)
	if err != nil {
		return nil, "", "", err
	}
	rc, err := s.store.Get(ctx, ver.FileKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: get %s: %v", domain.ErrDocumentFetchFailed, ver.FileKey, err)
	}
	return rc, "application/pdf", doc.Title + ".pdf", nil
}

func (s *DocumentService) ApplyEdits(
	ctx context.Context,
	userID, docID int64,
	ops []document.EditOp,
	title string,
) (int, error) {
	if len(ops) == 0 && strings.TrimSpace(title) == "" {
		return 0, domain.ErrDocumentEmptyContent
	}
	doc, err := s.docRepo.FindByID(ctx, docID)
	if err != nil {
		return 0, err
	}
	if doc.UserID != userID {
		return 0, domain.ErrDocumentForbidden
	}
	if doc.Source == "knowledge" {
		return 0, domain.ErrDocumentNotEditable
	}
	current, err := s.ReadCurrent(ctx, userID, docID)
	if err != nil {
		return 0, err
	}
	for _, op := range ops {
		current, err = op.Apply(current)
		if err != nil {
			return 0, err
		}
	}
	return s.writeNewVersion(ctx, doc, current, title)
}

func (s *DocumentService) Ingest(ctx context.Context, documentID int64) error {
	if err := s.ingestor.Ingest(ctx, documentID); err != nil {
		slog.Error("ingest failed",
			"document_id", documentID,
			"err", err)
		return err
	}
	return nil
}

// IngestOne triggers single ingest if dirty.
func (s *DocumentService) IngestOne(ctx context.Context, userID, documentID int64) error {
	doc, err := s.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return err
	}
	if doc.UserID != userID {
		return domain.ErrDocumentForbidden
	}
	status, err := s.docRepo.FindChunkStatus(ctx, documentID)
	if err != nil {
		return err
	}
	if status == document.ChunkStatusChunked {
		return nil
	}
	go s.runOne(documentID)
	return nil
}

// IngestAll bulk-ingests user docs by source.
func (s *DocumentService) IngestAll(ctx context.Context, userID int64, source string) (int, error) {
	if err := validateSource(source); err != nil {
		return 0, err
	}
	ids, err := s.docRepo.ListUnchunkedByUser(ctx, userID, source)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	go s.runBatch(ids)
	return len(ids), nil
}

func (s *DocumentService) List(ctx context.Context, userID int64, opts document.ListOpts) ([]*document.Document, int64, error) {
	if opts.Source != "" {
		if err := validateSource(opts.Source); err != nil {
			return nil, 0, err
		}
	}
	return s.docRepo.ListByUser(ctx, userID, opts)
}

// Archive soft-deletes and cleans tree pointers.
func (s *DocumentService) Archive(ctx context.Context, userID, documentID int64) error {
	if err := s.docRepo.Archive(ctx, documentID, userID); err != nil {
		return err
	}
	ids, err := s.tree.ListDocPointerIDs(ctx, userID, documentID)
	if err != nil {
		slog.Warn("archive: list doc pointer ids failed",
			"user_id", userID, "document_id", documentID, "err", err)
		return nil
	}
	for _, id := range ids {
		if err := s.tree.DeleteByOwner(ctx, id, userID); err != nil {
			slog.Warn("archive: delete tree pointer failed",
				"user_id", userID, "tree_node_id", id, "err", err)
		}
	}
	return nil
}

func (s *DocumentService) MarkDirty(ctx context.Context, id int64) error {
	return s.docRepo.MarkDirty(ctx, id)
}

// Move re-parents knowledge doc in tree.
func (s *DocumentService) Move(ctx context.Context, userID, docID int64, newParentTreeID *int64) error {
	doc, err := s.Get(ctx, userID, docID)
	if err != nil {
		return err
	}
	if doc.Source != "knowledge" {
		return domain.ErrDocumentNotEditable
	}
	node, err := s.tree.FindByDocID(ctx, userID, docID)
	if err != nil {
		return err
	}
	if newParentTreeID != nil {
		parent, err := s.tree.FindByID(ctx, *newParentTreeID)
		if err != nil {
			return err
		}
		if !parent.IsFolder {
			return domain.ErrTreeNodeNotFolder
		}
	}
	if node.ParentID == nil && newParentTreeID == nil {
		return nil
	}
	if node.ParentID != nil && newParentTreeID != nil && *node.ParentID == *newParentTreeID {
		return nil
	}
	return s.tree.UpdateParent(ctx, node.ID, newParentTreeID)
}

// runOne ingests with own ctx + timeout.
func (s *DocumentService) runOne(documentID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := s.ingestor.Ingest(ctx, documentID); err != nil {
		slog.Warn("ingest document failed",
			"document_id", documentID,
			"err", err)
	}
}

func (s *DocumentService) runBatch(ids []int64) {
	for _, id := range ids {
		s.runOne(id)
	}
}

var (
	_ document.Editor  = (*DocumentService)(nil)
	_ document.Lister  = (*DocumentService)(nil)
	_ document.Creator = (*DocumentService)(nil)
)
