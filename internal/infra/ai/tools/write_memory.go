package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	userapp "learn/internal/application/user"
	domainconv "learn/internal/domain/conversation"
)

const WriteMemoryToolName = "write_memory"

const (
	writeMemoryMaxContent = 2000
)

type WriteMemoryTool struct {
	users *userapp.UserService
}

func NewWriteMemoryTool(users *userapp.UserService) *WriteMemoryTool {
	return &WriteMemoryTool{users: users}
}

func (t *WriteMemoryTool) Spec() domainconv.Spec {
	return domainconv.Spec{
		Name: WriteMemoryToolName,
		Description: "Persist a learned fact about the user into the mutable section of their portrait, " +
			"so future turns remember it without you needing to bring it up again. " +
			"Use this when you learn something durable about the user — a preference, a working habit, " +
			"a context they expect you to carry forward. " +
			"Do NOT use it for ephemeral context (the current task, the current repo). " +
			"mode='append' adds content as a new line; mode='replace' overwrites the whole memory — " +
			"prefer append unless the user explicitly asked to rewrite or clear.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"append", "replace"},
					"description": "append: add content as a new line at the end of mutable. replace: overwrite the whole mutable field (use only when user asks to clear or rewrite).",
				},
				"content": map[string]any{
					"type":        "string",
					"minLength":   1,
					"maxLength":   writeMemoryMaxContent,
					"description": fmt.Sprintf("The fact to record. 1-%d chars. Be specific and concise — write it as a statement you'd want to read at the start of a future turn.", writeMemoryMaxContent),
				},
			},
			"required": []string{"mode", "content"},
		},
	}
}

func (t *WriteMemoryTool) Invoke(ctx context.Context, args map[string]any) domainconv.Result {
	userID, ok := domainconv.UserIDFromContext(ctx)
	if !ok {
		return domainconv.Result{Error: "write_memory: user_id missing from context"}
	}
	mode, _ := args["mode"].(string)
	switch mode {
	case "append", "replace":
	default:
		return domainconv.Result{Error: fmt.Sprintf("mode must be 'append' or 'replace', got %q", mode)}
	}
	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	if content == "" {
		return domainconv.Result{Error: "content must be non-empty"}
	}
	if len(content) > writeMemoryMaxContent {
		return domainconv.Result{Error: fmt.Sprintf("content exceeds %d chars", writeMemoryMaxContent)}
	}

	var next string
	if mode == "replace" {
		next = content
	} else {
		_, current, err := t.users.GetPortrait(ctx, userID)
		if err != nil {
			return domainconv.Result{Error: fmt.Sprintf("read current memory failed: %v", err)}
		}
		if current == "" {
			next = content
		} else {
			next = current + "\n" + content
		}
	}

	if err := t.users.UpdateMutable(ctx, userID, next); err != nil {
		return domainconv.Result{Error: err.Error()}
	}
	return domainconv.Result{Content: "memory updated"}
}

func (t *WriteMemoryTool) Traits() domainconv.Traits {
	return domainconv.Traits{
		Concurrent: false,
		Timeout:    5 * time.Second,
		Message: func(args map[string]any) string {
			mode, _ := args["mode"].(string)
			switch mode {
			case "replace":
				return "Replacing user memory"
			default:
				return "Saving to user memory"
			}
		},
	}
}

var _ domainconv.Tool = (*WriteMemoryTool)(nil)

func RegisterWriteMemoryTool(users *userapp.UserService) {
	Default.MustRegister(NewWriteMemoryTool(users))
}
