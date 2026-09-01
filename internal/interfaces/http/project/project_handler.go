package project

import (
	"net/http"

	app "learn/internal/application/project"
	domainproject "learn/internal/domain/project"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	projectSvc *app.ProjectService
}

func NewProjectHandler(projectSvc *app.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectSvc: projectSvc}
}

func (h *ProjectHandler) GetActive(c *gin.Context) {
	userID := middleware.IdentityFrom(c).UserID
	convoID := c.Param("conversation_id")
	if convoID == "" {
		response.AbortBadRequest(c, "empty conversation id")
		return
	}
	p, err := h.projectSvc.GetActive(c.Request.Context(), userID, convoID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if p == nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, activeProjectResponse{ID: p.ID, Name: p.Name, Title: p.Title, Cwd: p.ContainerPath(), PreviewURL: p.PreviewURL, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")})
}

func (h *ProjectHandler) SetActive(c *gin.Context) {
	var req setActiveProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	convoID := c.Param("conversation_id")
	if convoID == "" {
		// Gin still matches; reject empty UUIDs.
		response.AbortBadRequest(c, "empty conversation id")
		return
	}
	p, err := h.projectSvc.SetActive(c.Request.Context(), userID, convoID, req.ProjectID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	// Return null on unbind for parity.
	if p == nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, activeProjectResponse{ID: p.ID, Name: p.Name, Title: p.Title, Cwd: p.ContainerPath(), PreviewURL: p.PreviewURL, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")})
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	convoID := c.Query("conversation_id")
	msgID := c.Query("message_id")

	var (
		p   *domainproject.Project
		err error
	)
	if convoID != "" {
		p, err = h.projectSvc.CreateFromChat(c.Request.Context(), userID, convoID, msgID, req.Name)
	} else {
		p, err = h.projectSvc.CreateForUser(c.Request.Context(), userID, req.Name)
	}
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toProjectResponse(p))
}

func (h *ProjectHandler) Rename(c *gin.Context) {
	var req renameProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	p, err := h.projectSvc.Rename(c.Request.Context(), middleware.IdentityFrom(c).UserID, c.Param("project_id"), req.Title)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toProjectResponse(p))
}

func (h *ProjectHandler) List(c *gin.Context) {
	ps, err := h.projectSvc.List(c.Request.Context(), middleware.IdentityFrom(c).UserID, c.Query("conversation_id"), 50, 0)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := projectListResponse{Projects: make([]projectListItem, 0, len(ps))}
	for _, p := range ps {
		out.Projects = append(out.Projects, toProjectListItem(p))
	}
	response.OK(c, out)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	p, err := h.projectSvc.Get(c.Request.Context(), middleware.IdentityFrom(c).UserID, c.Param("project_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toProjectResponse(p))
}

func (h *ProjectHandler) Archive(c *gin.Context) {
	if err := h.projectSvc.Archive(c.Request.Context(), middleware.IdentityFrom(c).UserID, c.Param("project_id")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
