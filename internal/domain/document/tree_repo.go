package document

import "context"

type TreeRepo interface {
	FindByID(ctx context.Context, id int64) (*TreeNode, error)

	FindByDocID(ctx context.Context, userID, docID int64) (*TreeNode, error)

	ListByUser(ctx context.Context, userID int64) ([]*TreeNode, error)

	ListChildren(ctx context.Context, userID int64, parentID *int64) ([]*TreeNode, error)

	Insert(ctx context.Context, n *TreeNode) (int64, error)

	UpdateName(ctx context.Context, id int64, newName string) error

	UpdateParent(ctx context.Context, id int64, newParent *int64) error

	Delete(ctx context.Context, id int64) error

	DeleteByOwner(ctx context.Context, id, userID int64) error

	ListDocPointerIDs(ctx context.Context, userID, docID int64) ([]int64, error)

	IsAncestor(ctx context.Context, userID, ancestorID, descendantID int64) (bool, error)

	Depth(ctx context.Context, userID, id int64) (int, error)

	SubtreeMaxDepth(ctx context.Context, userID, id int64) (int, error)
}
