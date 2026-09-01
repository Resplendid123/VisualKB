package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"learn/internal/domain"
	"learn/internal/domain/document"
	"learn/internal/domain/s3"
	"learn/internal/infra/ai/chunk"
	"learn/internal/infra/data/model"
	"learn/internal/infra/data/repo"

	"github.com/pgvector/pgvector-go"
)

type Ingestor struct {
	docRepo     document.DocumentRepo
	versionRepo document.DocumentVersionRepo
	chunkRepo   repo.ChunkRepo
	store       s3.ObjectStore
	splitter    document.Splitter
	embedder    document.Embedder
}

func New(
	docRepo document.DocumentRepo,
	versionRepo document.DocumentVersionRepo,
	chunkRepo repo.ChunkRepo,
	store s3.ObjectStore,
	splitter document.Splitter,
	embedder document.Embedder,
) *Ingestor {
	return &Ingestor{
		docRepo:     docRepo,
		versionRepo: versionRepo,
		chunkRepo:   chunkRepo,
		store:       store,
		splitter:    splitter,
		embedder:    embedder,
	}
}

func (in *Ingestor) Ingest(ctx context.Context, documentID int64) error {
	doc, err := in.docRepo.FindByID(ctx, documentID)
	if err != nil {
		return err
	}
	ver, err := in.versionRepo.FindCurrent(ctx, doc)
	if err != nil {
		in.markError(ctx, documentID)
		return err
	}
	text, err := in.fetchText(ctx, doc.ContentType, ver.FileKey)
	if err != nil {
		in.markError(ctx, documentID)
		return err
	}
	cfg := chunk.DefaultConfig()
	if doc.Lang != "" {
		cfg.Language = doc.Lang
	}
	parentCfg := cfg
	parentCfg.MaxChars = cfg.MaxChars * 2
	parents, _, err := in.splitter.Split(ctx, text, parentCfg)
	if err != nil {
		in.markError(ctx, documentID)
		return fmt.Errorf("%w: split parents: %v", domain.ErrDocumentSplitFailed, err)
	}
	if len(parents) == 0 {

		_ = in.chunkRepo.DeleteByDocumentID(ctx, documentID)
		in.markError(ctx, documentID)
		return fmt.Errorf("%w: no parent chunks produced", domain.ErrDocumentSplitFailed)
	}
	children := make([]childChunk, 0)
	for pi, p := range parents {
		kids, _, err := in.splitter.Split(ctx, p.Content, cfg)
		if err != nil {
			in.markError(ctx, documentID)
			return fmt.Errorf("%w: split children (parent %d): %v", domain.ErrDocumentSplitFailed, pi, err)
		}
		for _, k := range kids {
			children = append(children, childChunk{parentIdx: pi, chunk: k})
		}
	}
	if len(children) == 0 {
		_ = in.chunkRepo.DeleteByDocumentID(ctx, documentID)
		in.markError(ctx, documentID)
		return fmt.Errorf("%w: no child chunks produced", domain.ErrDocumentSplitFailed)
	}
	contents := make([]string, len(children))
	for i, c := range children {
		contents[i] = c.chunk.Content
	}
	vecs, err := in.embedder.Embed(ctx, contents)
	if err != nil {
		in.markError(ctx, documentID)
		return fmt.Errorf("%w: embed: %v", domain.ErrDocumentEmbedFailed, err)
	}
	if err := in.replaceChunks(ctx, documentID, parents, children, vecs); err != nil {
		in.markError(ctx, documentID)
		return err
	}
	if err := in.docRepo.MarkChunked(ctx, documentID); err != nil {
		return fmt.Errorf("%w: mark chunked: %v", domain.ErrDocumentIngestFailed, err)
	}
	return nil
}

type childChunk struct {
	parentIdx int
	chunk     document.Chunk
}

func (in *Ingestor) fetchText(ctx context.Context, ct document.ContentType, key string) (string, error) {
	rc, err := in.store.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("s3 get %s: %w", key, domain.ErrDocumentFetchFailed)
	}
	defer rc.Close()
	text, err := document.Extract(ctx, ct, rc)
	if err != nil {
		return "", fmt.Errorf("extract text (ct=%s): %w", ct, err)
	}
	return text, nil
}

func (in *Ingestor) replaceChunks(
	ctx context.Context,
	documentID int64,
	parents []document.Chunk,
	children []childChunk,
	vecs [][]float32,
) error {
	if len(children) != len(vecs) {
		return fmt.Errorf("%w: child/vector count mismatch %d vs %d",
			domain.ErrDocumentEmbedFailed, len(children), len(vecs))
	}
	if err := in.chunkRepo.DeleteByDocumentID(ctx, documentID); err != nil {
		return fmt.Errorf("%w: clear old chunks: %v", domain.ErrDocumentIngestFailed, err)
	}
	parentUsed := make([]bool, len(parents))
	for _, c := range children {
		parentUsed[c.parentIdx] = true
	}
	parentRows := make([]model.Chunk, 0, len(parents))
	parentRowIdx := make([]int, len(parents))
	for i, p := range parents {
		if !parentUsed[i] {
			continue
		}
		parentRows = append(parentRows, model.Chunk{
			DocumentID: documentID,
			ChunkIndex: int(p.Index),
			Content:    p.Content,
			Header:     ptrString(p.Header),
		})
		parentRowIdx[i] = len(parentRows) - 1
	}
	if err := in.chunkRepo.BatchInsert(ctx, parentRows); err != nil {
		return fmt.Errorf("%w: insert parent chunks: %v", domain.ErrDocumentIngestFailed, err)
	}
	childRows := make([]model.Chunk, len(children))
	for i, c := range children {
		parentID := parentRows[parentRowIdx[c.parentIdx]].ID
		childRows[i] = model.Chunk{
			DocumentID: documentID,
			ChunkIndex: int(c.chunk.Index),
			Content:    c.chunk.Content,
			ParentID:   &parentID,
			Embedding:  pgvector.NewVector(vecs[i]),
		}
	}
	if err := in.chunkRepo.BatchInsert(ctx, childRows); err != nil {
		return fmt.Errorf("%w: insert child chunks: %v", domain.ErrDocumentIngestFailed, err)
	}
	return nil
}

func (in *Ingestor) markError(ctx context.Context, documentID int64) {
	if err := in.docRepo.MarkError(ctx, documentID); err != nil {
		slog.Error("ingest: mark error failed",
			"document_id", documentID,
			"err", err)
	}
}

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ document.Ingestor = (*Ingestor)(nil)
