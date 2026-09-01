package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"learn/internal/domain/conversation"
)

const SkillLoaderName = "load_skill"

type SkillLoaderTool struct {
	repo conversation.SkillRepo
}

func NewSkillLoaderTool(repo conversation.SkillRepo) *SkillLoaderTool {
	return &SkillLoaderTool{repo: repo}
}

func (t *SkillLoaderTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: SkillLoaderName,
		Description: "Load the full instructions for a previously announced skill. " +
			"Use this when a skill's name or description matches the user's request. ",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Skill name (as listed in the available skills section of the system prompt).",
				},
			},
			"required": []string{"name"},
		},
	}
}

func (t *SkillLoaderTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return conversation.Result{Error: "name is required"}
	}
	s, ok := t.repo.Get(name)
	if !ok {
		return conversation.Result{Error: fmt.Sprintf("skill %q not found", name)}
	}
	return conversation.Result{Content: s.Body}
}

func (t *SkillLoaderTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: true,
		Timeout:    5 * time.Second,
		Message: func(args map[string]any) string {
			name, _ := args["name"].(string)
			if name == "" {
				return "Loading skill"
			}
			return fmt.Sprintf("Loading skill: %s", name)
		},
	}
}

func RegisterSkillLoaderTool(repo conversation.SkillRepo) {
	Default.MustRegister(NewSkillLoaderTool(repo))
}
