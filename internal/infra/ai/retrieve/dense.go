package retrieve

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"learn/internal/domain/document"
	"learn/internal/infra/ai/query"
)

type Searcher struct {
	db          *gorm.DB
	embedder    document.Embedder
	reranker    document.Reranker
	transformer *query.Transformer
}

func NewSearcher(db *gorm.DB, e document.Embedder) *Searcher {
	return &Searcher{db: db, embedder: e}
}

func NewSearcherWithRerank(db *gorm.DB, e document.Embedder, r document.Reranker) *Searcher {
	return &Searcher{db: db, embedder: e, reranker: r}
}

func NewSearcherFull(db *gorm.DB, e document.Embedder, r document.Reranker, t *query.Transformer) *Searcher {
	return &Searcher{db: db, embedder: e, reranker: r, transformer: t}
}

func (s *Searcher) Hybrid(ctx context.Context, userID int64, q string, topN int) (hits []document.Hit, err error) {
	if topN <= 0 {
		topN = 10
	}
	totalStart := time.Now()
	defer func() {
		logPhase(ctx, "total", totalStart,
			"user_id", userID, "top_n", topN,
			"hits", len(hits), "status", errStatus(err),
		)
	}()

	tfStart := time.Now()
	queries := s.queriesFor(ctx, q)
	logPhase(ctx, "query_transform", tfStart, "user_id", userID, "variants", len(queries))

	lists := make([][]document.Hit, 0, len(queries))
	for i, qq := range queries {
		d, derr := s.denseOnce(ctx, userID, qq, topN*3)
		if derr != nil {
			return nil, derr
		}
		b, berr := s.bm25Once(ctx, userID, qq, topN*3)
		if berr != nil {
			return nil, berr
		}
		rrfStart := time.Now()
		merged := rrfMerge([][]document.Hit{d, b}, topN*3)
		logPhase(ctx, "rrf_within", rrfStart,
			"variant", i,
			"dense_hits", len(d), "bm25_hits", len(b), "merged", len(merged),
		)
		lists = append(lists, merged)
	}
	crossStart := time.Now()
	merged := rrfMerge(lists, topN*3)
	logPhase(ctx, "rrf_cross", crossStart, "variants", len(lists), "merged", len(merged))

	withParents, hErr := s.hydrateParents(ctx, userID, merged)
	if hErr != nil {
		err = hErr
		return nil, err
	}
	return s.maybeRerank(ctx, q, withParents, topN)
}

func (s *Searcher) queriesFor(ctx context.Context, q string) []string {
	if s.transformer == nil {
		return []string{q}
	}
	out := s.transformer.Apply(ctx, q)
	if len(out) == 0 {
		return []string{q}
	}
	return out
}

func (s *Searcher) denseOnce(ctx context.Context, userID int64, q string, topN int) ([]document.Hit, error) {
	embedStart := time.Now()
	vec, err := s.embedder.Embed(ctx, []string{q})
	logPhase(ctx, "embed", embedStart, "user_id", userID, "err", err)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vec) == 0 {
		return nil, nil
	}
	searchStart := time.Now()
	var raw []hitRow
	err = s.db.WithContext(ctx).Raw(`
SELECT dc.id AS child_id, dc.parent_id, dc.document_id, d.title, d.source,
       1.0 - (dc.embedding <=> ?::vector) AS score,
       dc.content, dc.header, dc.created_at
FROM document_chunks dc
JOIN documents d ON d.id = dc.document_id
WHERE d.user_id = ? AND d.chunk_status = 1 AND dc.embedding IS NOT NULL
ORDER BY dc.embedding <=> ?::vector
LIMIT ?
`, vecToPg(vec[0]), userID, vecToPg(vec[0]), topN).Scan(&raw).Error
	logPhase(ctx, "dense_search", searchStart, "user_id", userID, "rows", len(raw), "err", err)
	if err != nil {
		return nil, fmt.Errorf("dense query: %w", err)
	}
	return s.hydrateParents(ctx, userID, rowsToHits(raw))
}

func (s *Searcher) bm25Once(ctx context.Context, userID int64, q string, topN int) ([]document.Hit, error) {
	searchStart := time.Now()
	var raw []hitRow
	err := s.db.WithContext(ctx).Raw(`
SELECT dc.id AS child_id, dc.parent_id, dc.document_id, d.title, d.source,
       pdb.score(dc.id) AS score,
       dc.content, dc.header, dc.created_at
FROM document_chunks dc
JOIN documents d ON d.id = dc.document_id
WHERE d.user_id = ? AND d.chunk_status = 1 AND dc.embedding IS NOT NULL
  AND dc.content ||| ?
ORDER BY pdb.score(dc.id) DESC
LIMIT ?
`, userID, q, topN).Scan(&raw).Error
	logPhase(ctx, "bm25_search", searchStart, "user_id", userID, "rows", len(raw), "err", err)
	if err != nil {
		return nil, fmt.Errorf("bm25 query: %w", err)
	}
	return s.hydrateParents(ctx, userID, rowsToHits(raw))
}

func (s *Searcher) hydrateParents(ctx context.Context, userID int64, hits []document.Hit) (out []document.Hit, err error) {
	hydrateStart := time.Now()
	defer func() {
		logPhase(ctx, "hydrate_parents", hydrateStart,
			"user_id", userID,
			"hits", len(out), "err", err,
		)
	}()
	if len(hits) == 0 {
		return hits, nil
	}
	parentIDs := []int64{}
	seen := map[int64]bool{}
	for _, h := range hits {
		if h.ParentID > 0 && !seen[h.ParentID] {
			parentIDs = append(parentIDs, h.ParentID)
			seen[h.ParentID] = true
		}
	}
	if len(parentIDs) == 0 {
		return hits, nil
	}
	var parents []parentRow
	err = s.db.WithContext(ctx).
		Table("document_chunks dc").
		Joins("JOIN documents d ON d.id = dc.document_id").
		Select("dc.id, dc.document_id, dc.content, dc.header").
		Where("dc.id IN ? AND d.user_id = ?", parentIDs, userID).
		Scan(&parents).Error
	if err != nil {
		return nil, fmt.Errorf("hydrate parents: %w", err)
	}
	byID := map[int64]parentRow{}
	for _, p := range parents {
		byID[p.ID] = p
	}
	out = make([]document.Hit, 0, len(hits))
	seenP := map[int64]bool{}
	for _, h := range hits {
		if h.ParentID > 0 {
			if seenP[h.ParentID] {
				continue
			}
			seenP[h.ParentID] = true
		}
		hh := h
		if h.ParentID > 0 {
			if p, ok := byID[h.ParentID]; ok {
				hh.Content = p.Content
				if p.Header != nil {
					hh.Header = *p.Header
				}
			}
		}
		out = append(out, hh)
	}
	return out, nil
}

type parentRow struct {
	ID         int64   `gorm:"column:id"`
	DocumentID int64   `gorm:"column:document_id"`
	Content    string  `gorm:"column:content"`
	Header     *string `gorm:"column:header"`
}

func (s *Searcher) maybeRerank(ctx context.Context, q string, hits []document.Hit, topN int) (out []document.Hit, err error) {
	rerankStart := time.Now()
	defer func() {
		logPhase(ctx, "rerank", rerankStart,
			"candidates", len(hits),
			"hits", len(out), "err", err,
		)
	}()
	if s.reranker == nil || len(hits) == 0 {
		if topN > 0 && topN < len(hits) {
			return hits[:topN], nil
		}
		return hits, nil
	}
	contents := make([]string, len(hits))
	for i, h := range hits {
		contents[i] = h.Content
	}
	ordered, err := s.reranker.Rerank(ctx, q, contents)
	if err != nil {

		if topN > 0 && topN < len(hits) {
			return hits[:topN], nil
		}
		return hits, nil
	}
	out = make([]document.Hit, 0, min(topN, len(ordered)))
	for _, it := range ordered {
		if it.Index < 0 || it.Index >= len(hits) {
			continue
		}
		out = append(out, hits[it.Index])
		if len(out) >= topN {
			break
		}
	}
	if len(out) == 0 {
		return hits[:min(topN, len(hits))], nil
	}
	return out, nil
}

var _ document.Searcher = (*Searcher)(nil)
