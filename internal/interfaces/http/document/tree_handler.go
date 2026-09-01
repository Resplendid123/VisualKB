package document

import (
	"strconv"

	"github.com/gin-gonic/gin"

	app "learn/internal/application/document"
	"learn/internal/domain/document"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/response"
)

type TreeHandler struct {
	treeSvc *app.TreeService
}

func NewTreeHandler(treeSvc *app.TreeService) *TreeHandler {
	return &TreeHandler{treeSvc: treeSvc}
}

// List returns the user's whole tree.
func (h *TreeHandler) List(c *gin.Context) {
	userID := middleware.IdentityFrom(c).UserID
	nodes, err := h.treeSvc.ListTree(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]*treeNodeDTO, len(nodes))
	for i, n := range nodes {
		out[i] = toTreeNodeDTO(n)
	}
	response.OK(c, treeListResponse{Nodes: out})
}

// CreateFolder makes an empty folder.
func (h *TreeHandler) CreateFolder(c *gin.Context) {
	var req treeCreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	id, err := h.treeSvc.CreateFolder(c.Request.Context(), userID, req.ParentID, req.Name)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(201, response.Response{
		Code:    response.CodeSuccess,
		Message: "success",
		Data:    treeNodeIDResponse{ID: id, ParentID: req.ParentID},
	})
}

// RenameNode renames a folder.
func (h *TreeHandler) RenameNode(c *gin.Context) {
	nodeID, ok := parseTreeNodeID(c)
	if !ok {
		return
	}
	var req treeRenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.treeSvc.RenameNode(c.Request.Context(), userID, nodeID, req.Name); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, treeNodeIDResponse{ID: nodeID})
}

// MoveNode reparents a folder.
func (h *TreeHandler) MoveNode(c *gin.Context) {
	nodeID, ok := parseTreeNodeID(c)
	if !ok {
		return
	}
	var req treeMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.treeSvc.MoveNode(c.Request.Context(), userID, nodeID, req.ParentID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, treeNodeIDResponse{ID: nodeID, ParentID: req.ParentID})
}

// DeleteFolder deletes only empty folders.
func (h *TreeHandler) DeleteFolder(c *gin.Context) {
	nodeID, ok := parseTreeNodeID(c)
	if !ok {
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.treeSvc.DeleteFolder(c.Request.Context(), userID, nodeID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, treeNodeIDResponse{ID: nodeID})
}

func parseTreeNodeID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	nodeID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || nodeID <= 0 {
		response.AbortBadRequest(c, "invalid tree node id")
		return 0, false
	}
	return nodeID, true
}

func toTreeNodeDTO(n *document.TreeNode) *treeNodeDTO {
	return &treeNodeDTO{
		ID:        n.ID,
		UserID:    n.UserID,
		ParentID:  n.ParentID,
		Name:      n.Name,
		IsFolder:  n.IsFolder,
		DocID:     n.DocID,
		CreatedAt: n.CreatedAt,
	}
}
