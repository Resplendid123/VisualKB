package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	domainconv "learn/internal/domain/conversation"
)

const (
	AskUserToolName = "ask_user_tool"

	minAskOptions = 1
	maxAskOptions = 8
)

type AskUserTool struct{}

func NewAskUserTool() *AskUserTool { return &AskUserTool{} }

func (t *AskUserTool) Spec() domainconv.Spec {
	return domainconv.Spec{
		Name: AskUserToolName,
		Description: "Ask the user a multiple-choice question and pause until they pick an option. " +
			"Use this when you need clarification, a preference, or a single decisive answer before continuing. " +
			"Do NOT use it for open-ended questions — pick 2-4 short, mutually exclusive options instead. " +
			"The user sees the options as clickable buttons above the chat input; the chosen option's label is sent verbatim as the next user message.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"minLength":   1,
					"maxLength":   280,
					"description": "The question shown to the user; keep it short (one sentence) and self-contained.",
				},
				"options": map[string]any{
					"type":        "array",
					"minItems":    minAskOptions,
					"maxItems":    maxAskOptions,
					"description": fmt.Sprintf("Choice list (%d-%d options). Each option has a 'label' (button text + verbatim next user message) and a required 'desc' (helper text under the label that explains the option).", minAskOptions, maxAskOptions),
					"items": map[string]any{
						"type":     "object",
						"required": []string{"label", "desc"},
						"properties": map[string]any{
							"label": map[string]any{
								"type":        "string",
								"minLength":   1,
								"maxLength":   80,
								"description": "Short button label (1-80 chars); also sent verbatim as the next user message when this option is picked.",
							},
							"desc": map[string]any{
								"type":        "string",
								"minLength":   1,
								"maxLength":   200,
								"description": "Helper text shown under the label — explains what this option means or what choosing it does.",
							},
						},
					},
				},
			},
			"required": []string{"question", "options"},
		},
	}
}

type askOption struct {
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

func (t *AskUserTool) Invoke(ctx context.Context, args map[string]any) domainconv.Result {
	question, _ := args["question"].(string)
	question = strings.TrimSpace(question)
	if question == "" {
		return domainconv.Result{Error: "question is required"}
	}
	rawOpts, _ := args["options"].([]any)
	if len(rawOpts) < minAskOptions || len(rawOpts) > maxAskOptions {
		return domainconv.Result{Error: fmt.Sprintf("options must have %d-%d entries", minAskOptions, maxAskOptions)}
	}
	options := make([]domainconv.AskOption, 0, len(rawOpts))
	seen := map[string]struct{}{}
	for i, raw := range rawOpts {
		blob, _ := json.Marshal(raw)
		var one askOption
		if err := json.Unmarshal(blob, &one); err != nil {
			return domainconv.Result{Error: fmt.Sprintf("options[%d] invalid shape: %v", i, err)}
		}
		one.Label = strings.TrimSpace(one.Label)
		one.Desc = strings.TrimSpace(one.Desc)
		if one.Label == "" {
			return domainconv.Result{Error: fmt.Sprintf("options[%d] requires non-empty label", i)}
		}
		if one.Desc == "" {
			return domainconv.Result{Error: fmt.Sprintf("options[%d] requires non-empty desc", i)}
		}
		if _, dup := seen[one.Label]; dup {
			return domainconv.Result{Error: fmt.Sprintf("options[%d] duplicate label %q", i, one.Label)}
		}
		seen[one.Label] = struct{}{}
		options = append(options, domainconv.AskOption{
			Label: one.Label,
			Desc:  one.Desc,
		})
	}

	toolCallID, _ := domainconv.ToolCallIDFromContext(ctx)
	sink, ok := domainconv.EventSinkFromContext(ctx)
	if !ok {
		return domainconv.Result{Error: "ask_user_tool: event sink unavailable in context"}
	}
	payload, _ := json.Marshal(domainconv.QuestionPayload{
		ToolCallID: toolCallID,
		Question:   question,
		Options:    options,
	})
	_ = sink.Emit(ctx, domainconv.Event{Type: domainconv.EventTypeQuestion, Payload: payload})

	preview := make([]string, 0, len(options))
	for _, o := range options {
		preview = append(preview, fmt.Sprintf("%q (desc=%q)", o.Label, o.Desc))
	}
	return domainconv.Result{
		Content: fmt.Sprintf(
			"STOP: awaiting user selection for question=%q; options=[%s]. Do NOT pick on the user's behalf — the agent loop has paused and will resume only when the next user-role message arrives, which will be one of the labels above verbatim.",
			question, strings.Join(preview, ", "),
		),
	}
}

func (t *AskUserTool) Traits() domainconv.Traits {
	return domainconv.Traits{
		Concurrent: false,
		Timeout:    5 * 1_000_000_000,
		Message: func(args map[string]any) string {
			q, _ := args["question"].(string)
			if q == "" {
				return "Asking user"
			}
			return "Asking: " + q
		},
		PausesTurn: true,
	}
}

var _ domainconv.Tool = (*AskUserTool)(nil)

func RegisterAskUserTool() {
	Default.MustRegister(NewAskUserTool())
}
