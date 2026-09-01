package document

import "time"

type treeNodeDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	ParentID  *int64    `json:"parent_id"`
	Name      string    `json:"name"`
	IsFolder  bool      `json:"is_folder"`
	DocID     *int64    `json:"doc_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type treeListResponse struct {
	Nodes []*treeNodeDTO `json:"nodes"`
}

// treeCreateFolderRequest allows nil parent_id.
type treeCreateFolderRequest struct {
	ParentID *int64 `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
}

type treeNodeIDResponse struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
}

// treeRenameRequest requires a single-segment name.
type treeRenameRequest struct {
	Name string `json:"name" binding:"required"`
}

type treeMoveRequest struct {
	ParentID *int64 `json:"parent_id"`
}
