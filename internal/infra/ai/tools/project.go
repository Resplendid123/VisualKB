package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"learn/internal/domain/conversation"
	"learn/internal/domain/project"
)

const ProjectToolName = "create_project"

type CreateProjectTool struct {
	exec project.ProjectOps
}

func NewCreateProjectTool(exec project.ProjectOps) *CreateProjectTool {
	return &CreateProjectTool{exec: exec}
}

func (t *CreateProjectTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: ProjectToolName,
		Description: "Create a new project and bind it to the current conversation. " +
			"`name` must match ^[a-z0-9][a-z0-9-]{0,62}$ (lowercase letters, digits, hyphens).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Project name.",
				},
			},
			"required": []string{"name"},
		},
	}
}

type ProjectResult struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Cwd   string `json:"cwd"`
}

func (t *CreateProjectTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return conversation.Result{Error: "name is required"}
	}
	userID, ok := conversation.UserIDFromContext(ctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}
	convoID, _ := conversation.ConversationIDFromContext(ctx)
	msgID, _ := conversation.MessageIDFromContext(ctx)

	p, err := t.exec.CreateFromChat(ctx, userID, convoID, msgID, name)
	if err != nil {
		return conversation.Result{Error: err.Error()}
	}

	res := ProjectResult{
		ID:    p.ID,
		Name:  p.Name,
		Title: p.Title,
		Cwd:   p.ContainerPath(),
	}
	content, mErr := json.Marshal(res)
	if mErr != nil {
		return conversation.Result{Error: mErr.Error()}
	}
	return conversation.Result{Content: string(content)}
}

func (t *CreateProjectTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: false,
		Timeout:    30 * time.Second,
		Message: func(args map[string]any) string {
			name, _ := args["name"].(string)
			if name == "" {
				return "Creating project"
			}
			return fmt.Sprintf("Creating project: %s", name)
		},
	}
}

const ListProjectsToolName = "list_projects"

type ListProjectsTool struct {
	exec project.ProjectOps
}

func NewListProjectsTool(exec project.ProjectOps) *ListProjectsTool {
	return &ListProjectsTool{exec: exec}
}

func (t *ListProjectsTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: ListProjectsToolName,
		Description: "List all projects owned by the current user. Returns name, title, last update time. " +
			"Use to locate an existing project before opening a conversation bound to it.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

type ProjectSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title"`
	Active    bool   `json:"active"`
	UpdatedAt string `json:"updated_at"`
}

type ListProjectsResult struct {
	Projects []ProjectSummary `json:"projects"`
}

func (t *ListProjectsTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	userID, ok := conversation.UserIDFromContext(ctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}

	convoID, _ := conversation.ConversationIDFromContext(ctx)

	ps, err := t.exec.List(ctx, userID, convoID, 50, 0)
	if err != nil {
		return conversation.Result{Error: err.Error()}
	}

	out := ListProjectsResult{Projects: make([]ProjectSummary, 0, len(ps))}
	for _, p := range ps {
		out.Projects = append(out.Projects, ProjectSummary{
			ID:        p.ID,
			Name:      p.Name,
			Title:     p.Title,
			Active:    p.IsActive,
			UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	content, mErr := json.Marshal(out)
	if mErr != nil {
		return conversation.Result{Error: mErr.Error()}
	}
	return conversation.Result{Content: string(content)}
}

func (t *ListProjectsTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: true,
		Timeout:    30 * time.Second,
		Message: func(args map[string]any) string {
			return "Listing projects"
		},
	}
}

func RegisterProjectTools(exec project.ProjectOps) {
	Default.MustRegister(NewCreateProjectTool(exec))
	Default.MustRegister(NewListProjectsTool(exec))
}
