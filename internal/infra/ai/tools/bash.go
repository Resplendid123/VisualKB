package tools

import (
	"context"
	"fmt"
	"time"

	"learn/internal/domain/conversation"
	"learn/internal/domain/project"
)

const defaultBashTimeout = 5 * time.Minute

type BashTool struct {
	exec project.CommandExecutor
}

func NewBashTool(exec project.CommandExecutor) *BashTool {
	return &BashTool{exec: exec}
}

func (b *BashTool) Spec() conversation.Spec {
	return conversation.Spec{
		Name: conversation.BashName,
		Description: "Run a shell command inside a sandbox pod. " +
			"cwd is `/workspace/project` (do not `cd`). Runs via `bash -c`; non-zero exit is an error. " +
			"`/workspace/project` is fuse (noexec) — install + run binaries from `/tmp`, " +
			"copy results back. For previews: build output MUST land in `/workspace/dist/`.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to execute. Runs via `bash -c` inside the sandbox; cwd is /workspace/project.",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (b *BashTool) Invoke(ctx context.Context, args map[string]any) conversation.Result {
	command, _ := args["command"].(string)
	if command == "" {
		return conversation.Result{Error: "command is required"}
	}

	userID, ok := conversation.UserIDFromContext(ctx)
	if !ok {
		return conversation.Result{Error: "sandbox context missing user_id"}
	}
	projectID, _ := conversation.ProjectIDFromContext(ctx)
	convoID, _ := conversation.ConversationIDFromContext(ctx)
	msgID, _ := conversation.MessageIDFromContext(ctx)
	timeout := b.Traits().Timeout

	content, err := b.exec.ExecCommand(ctx, userID, projectID, convoID, msgID, command, timeout)
	if err != nil {
		return conversation.Result{Content: content, Error: err.Error()}
	}
	return conversation.Result{Content: content}
}

func (b *BashTool) Traits() conversation.Traits {
	return conversation.Traits{
		Concurrent: false,
		Timeout:    defaultBashTimeout,
		Message: func(args map[string]any) string {
			cmd, _ := args["command"].(string)
			if cmd == "" {
				return "Running bash command"
			}
			return fmt.Sprintf("Running: %s", cmd)
		},
	}
}

func RegisterBashTool(exec project.CommandExecutor) {
	Default.MustRegister(NewBashTool(exec))
}
