package document

import (
	"context"
	"log/slog"
	"slices"

	"learn/internal/domain"
	"learn/internal/domain/document"
)

type TreeService struct {
	tree   document.TreeRepo
	docSvc *DocumentService
}

func NewTreeService(tree document.TreeRepo, docSvc *DocumentService) *TreeService {
	return &TreeService{tree: tree, docSvc: docSvc}
}

func (s *TreeService) ListTree(ctx context.Context, userID int64) ([]*document.TreeNode, error) {
	return s.tree.ListByUser(ctx, userID)
}

// CreateFolder creates empty folder under parent.
func (s *TreeService) CreateFolder(ctx context.Context, userID int64, parentID *int64, name string) (int64, error) {
	cleaned, err := document.NormalizeName(name)
	if err != nil {
		return 0, err
	}
	if parentID != nil {
		parent, err := s.tree.FindByID(ctx, *parentID)
		if err != nil {
			return 0, err
		}
		if !parent.IsFolder {
			return 0, domain.ErrTreeNodeNotFolder
		}
		pd, err := s.tree.Depth(ctx, userID, *parentID)
		if err != nil {
			return 0, err
		}
		if pd+1 > document.MaxFolderDepth {
			return 0, domain.ErrTreeNodeMaxDepth
		}
	}
	siblings, err := s.tree.ListChildren(ctx, userID, parentID)
	if err != nil {
		return 0, err
	}
	for _, sib := range siblings {
		if sib.Name == cleaned {
			return 0, domain.ErrTreeNodeNameTaken
		}
	}
	node := &document.TreeNode{
		UserID:   userID,
		ParentID: parentID,
		Name:     cleaned,
		IsFolder: true,
	}
	id, err := s.tree.Insert(ctx, node)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// RenameNode renames folder.
func (s *TreeService) RenameNode(ctx context.Context, userID, nodeID int64, newName string) error {
	cleaned, err := document.NormalizeName(newName)
	if err != nil {
		return err
	}
	node, err := s.tree.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.UserID != userID {
		return domain.ErrTreeNodeNotFound
	}
	if !node.IsFolder {
		return domain.ErrTreeNodeNotFolder
	}
	siblings, err := s.tree.ListChildren(ctx, userID, node.ParentID)
	if err != nil {
		return err
	}
	for _, sib := range siblings {
		if sib.ID != nodeID && sib.Name == cleaned {
			return domain.ErrTreeNodeNameTaken
		}
	}
	return s.tree.UpdateName(ctx, nodeID, cleaned)
}

// MoveNode moves folder to new parent.
func (s *TreeService) MoveNode(ctx context.Context, userID, nodeID int64, newParent *int64) error {
	node, err := s.tree.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.UserID != userID {
		return domain.ErrTreeNodeNotFound
	}
	if !node.IsFolder {
		return domain.ErrTreeNodeNotFolder
	}
	if newParent != nil {
		if *newParent == nodeID {
			return domain.ErrTreeNodeCycle
		}
		parent, err := s.tree.FindByID(ctx, *newParent)
		if err != nil {
			return err
		}
		if !parent.IsFolder {
			return domain.ErrTreeNodeNotFolder
		}
		isDesc, err := s.tree.IsAncestor(ctx, userID, nodeID, *newParent)
		if err != nil {
			return err
		}
		if isDesc {
			return domain.ErrTreeNodeCycle
		}
		// Reject if depth + subtree exceeds max.
		pd, err := s.tree.Depth(ctx, userID, *newParent)
		if err != nil {
			return err
		}
		sd, err := s.tree.SubtreeMaxDepth(ctx, userID, nodeID)
		if err != nil {
			return err
		}
		if pd+1+sd > document.MaxFolderDepth {
			return domain.ErrTreeNodeMaxDepth
		}
	}

	sameParent := (node.ParentID == nil && newParent == nil) ||
		(node.ParentID != nil && newParent != nil && *node.ParentID == *newParent)
	if sameParent {
		return nil
	}

	siblings, err := s.tree.ListChildren(ctx, userID, newParent)
	if err != nil {
		return err
	}
	for _, sib := range siblings {
		if sib.ID != nodeID && sib.Name == node.Name {
			return domain.ErrTreeNodeNameTaken
		}
	}
	return s.tree.UpdateParent(ctx, nodeID, newParent)
}

// DeleteFolder archives descendants then removes nodes.
func (s *TreeService) DeleteFolder(ctx context.Context, userID, nodeID int64) error {
	node, err := s.tree.FindByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node.UserID != userID {
		return domain.ErrTreeNodeNotFound
	}
	if !node.IsFolder {
		return domain.ErrTreeNodeNotFolder
	}
	all, err := s.tree.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	subtree := collectSubtree(all, nodeID)
	if len(subtree) == 0 {
		// Treat as success if missing concurrently.
		return nil
	}

	for _, n := range subtree {
		if n.IsFolder || n.DocID == nil {
			continue
		}
		if err := s.docSvc.Archive(ctx, userID, *n.DocID); err != nil {
			slog.Error("tree: archive descendant doc failed",
				"folder_id", nodeID, "doc_id", *n.DocID, "err", err)
			return err
		}
	}
	// Delete leaves first, parents last.
	for _, s0 := range slices.Backward(subtree) {
		if err := s.tree.Delete(ctx, s0.ID); err != nil {
			slog.Error("tree: delete subtree node failed",
				"folder_id", nodeID, "node_id", s0.ID, "err", err)
			return err
		}
	}
	return nil
}

func collectSubtree(all []*document.TreeNode, rootID int64) []*document.TreeNode {
	childrenByParent := make(map[int64][]*document.TreeNode)
	for _, n := range all {
		if n.ParentID == nil {
			continue
		}
		childrenByParent[*n.ParentID] = append(childrenByParent[*n.ParentID], n)
	}
	out := make([]*document.TreeNode, 0, 8)
	var walk func(id int64)
	walk = func(id int64) {
		for _, c := range childrenByParent[id] {
			out = append(out, c)
			walk(c.ID)
		}
	}
	walk(rootID)
	return out
}
