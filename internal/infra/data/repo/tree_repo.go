package repo

import (
	"context"
	"errors"

	"learn/internal/domain"
	"learn/internal/domain/document"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type treeRepo struct {
	db *gorm.DB
}

func NewTreeRepo(db *gorm.DB) document.TreeRepo {
	return &treeRepo{db: db}
}

func (r *treeRepo) FindByID(ctx context.Context, id int64) (*document.TreeNode, error) {
	var m model.KnowledgeTree
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTreeNodeNotFound
		}
		return nil, err
	}
	return treeToDomain(&m), nil
}

func (r *treeRepo) FindByDocID(ctx context.Context, userID, docID int64) (*document.TreeNode, error) {
	var m model.KnowledgeTree
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND doc_id = ? AND is_folder = false", userID, docID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTreeNodeNotFound
		}
		return nil, err
	}
	return treeToDomain(&m), nil
}

func (r *treeRepo) ListByUser(ctx context.Context, userID int64) ([]*document.TreeNode, error) {
	var rows []model.KnowledgeTree
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("parent_id ASC NULLS FIRST, name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*document.TreeNode, len(rows))
	for i := range rows {
		out[i] = treeToDomain(&rows[i])
	}
	return out, nil
}

func (r *treeRepo) ListChildren(ctx context.Context, userID int64, parentID *int64) ([]*document.TreeNode, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	var rows []model.KnowledgeTree
	if err := q.Order("is_folder DESC, name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*document.TreeNode, len(rows))
	for i := range rows {
		out[i] = treeToDomain(&rows[i])
	}
	return out, nil
}

func (r *treeRepo) Insert(ctx context.Context, n *document.TreeNode) (int64, error) {
	if n.ID != 0 {
		return 0, errors.New("insert: tree node id must be zero (DB-generated)")
	}
	m := model.KnowledgeTree{
		UserID:   n.UserID,
		ParentID: n.ParentID,
		Name:     n.Name,
		IsFolder: n.IsFolder,
		DocID:    n.DocID,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return 0, err
	}
	n.ID = m.ID
	n.CreatedAt = m.CreatedAt
	return m.ID, nil
}

func (r *treeRepo) UpdateName(ctx context.Context, id int64, newName string) error {
	res := r.db.WithContext(ctx).Model(&model.KnowledgeTree{}).
		Where("id = ?", id).
		Update("name", newName)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTreeNodeNotFound
	}
	return nil
}

func (r *treeRepo) UpdateParent(ctx context.Context, id int64, newParent *int64) error {
	res := r.db.WithContext(ctx).Model(&model.KnowledgeTree{}).
		Where("id = ?", id).
		Update("parent_id", newParent)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTreeNodeNotFound
	}
	return nil
}

func (r *treeRepo) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.KnowledgeTree{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTreeNodeNotFound
	}
	return nil
}

func (r *treeRepo) DeleteByOwner(ctx context.Context, id, userID int64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.KnowledgeTree{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTreeNodeNotFound
	}
	return nil
}

func (r *treeRepo) ListDocPointerIDs(ctx context.Context, userID, docID int64) ([]int64, error) {
	var ids []int64
	if err := r.db.WithContext(ctx).
		Model(&model.KnowledgeTree{}).
		Where("user_id = ? AND doc_id = ? AND is_folder = false", userID, docID).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *treeRepo) IsAncestor(ctx context.Context, userID, ancestorID, descendantID int64) (bool, error) {
	if ancestorID == descendantID {
		return true, nil
	}
	const q = `
WITH RECURSIVE subtree AS (
    SELECT id, parent_id, user_id FROM knowledge_tree WHERE id = ? AND user_id = ?
    UNION ALL
    SELECT t.id, t.parent_id, t.user_id
    FROM knowledge_tree t
    JOIN subtree s ON t.parent_id = s.id
)
SELECT EXISTS(SELECT 1 FROM subtree WHERE id = ?)
`
	var exists bool
	if err := r.db.WithContext(ctx).Raw(q, ancestorID, userID, descendantID).Scan(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (r *treeRepo) Depth(ctx context.Context, userID, id int64) (int, error) {
	const q = `
WITH RECURSIVE ancestors AS (
    SELECT id, parent_id, 1 AS depth FROM knowledge_tree WHERE id = ? AND user_id = ?
    UNION ALL
    SELECT t.id, t.parent_id, a.depth + 1
    FROM knowledge_tree t
    JOIN ancestors a ON t.id = a.parent_id
    WHERE a.parent_id IS NOT NULL
)
SELECT COALESCE(MAX(depth), 0) FROM ancestors
`
	var d int
	if err := r.db.WithContext(ctx).Raw(q, id, userID).Scan(&d).Error; err != nil {
		return 0, err
	}
	return d, nil
}

func (r *treeRepo) SubtreeMaxDepth(ctx context.Context, userID, id int64) (int, error) {
	const q = `
WITH RECURSIVE subtree AS (
    SELECT id, parent_id, is_folder, 0 AS rel_depth
    FROM knowledge_tree WHERE id = ? AND user_id = ?
    UNION ALL
    SELECT t.id, t.parent_id, t.is_folder, s.rel_depth + 1
    FROM knowledge_tree t
    JOIN subtree s ON t.parent_id = s.id
)
SELECT COALESCE(MAX(rel_depth), 0) FROM subtree WHERE is_folder = true
`
	var d int
	if err := r.db.WithContext(ctx).Raw(q, id, userID).Scan(&d).Error; err != nil {
		return 0, err
	}
	return d, nil
}

var _ document.TreeRepo = (*treeRepo)(nil)

func treeToDomain(m *model.KnowledgeTree) *document.TreeNode {
	return &document.TreeNode{
		ID:        m.ID,
		UserID:    m.UserID,
		ParentID:  cloneInt64Ptr(m.ParentID),
		Name:      m.Name,
		IsFolder:  m.IsFolder,
		DocID:     cloneInt64Ptr(m.DocID),
		CreatedAt: m.CreatedAt,
	}
}

func cloneInt64Ptr(p *int64) *int64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
