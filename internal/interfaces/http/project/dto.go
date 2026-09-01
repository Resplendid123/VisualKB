package project

import domainproject "learn/internal/domain/project"

type activeProjectResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	Cwd        string `json:"cwd"`
	PreviewURL string `json:"preview_url,omitempty"` // "" when sandbox not ready
	UpdatedAt  string `json:"updated_at"`
}

type createProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

type projectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Cwd       string `json:"cwd"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type projectListItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type projectListResponse struct {
	Projects []projectListItem `json:"projects"`
}

type setActiveProjectRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
}

type renameProjectRequest struct {
	Title string `json:"title" binding:"required"`
}

func toProjectResponse(p *domainproject.Project) projectResponse {
	return projectResponse{
		ID: p.ID, Name: p.Name, Title: p.Title, Cwd: p.ContainerPath(),
		Status: p.Status, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProjectListItem(p *domainproject.Project) projectListItem {
	return projectListItem{
		ID: p.ID, Name: p.Name, Title: p.Title, Status: p.Status,
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
