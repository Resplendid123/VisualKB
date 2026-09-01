package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"learn/internal/domain/conversation"
	"learn/internal/infra/config"
)

type OpenAI struct {
	openai.Client
	APIKey  string
	BaseURL string
	Model   string
	m       *metrics
}

func NewOpenAI(cfg config.LLMConfig) *OpenAI {
	opts := []option.RequestOption{option.WithAPIKey(cfg.ApiKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	m, _ := newMetrics()
	return &OpenAI{
		Client:  openai.NewClient(opts...),
		APIKey:  cfg.ApiKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		m:       m,
	}
}

func (o *OpenAI) recordResult(operation, status, finishReason string, toolCount int, start time.Time, ttft *time.Duration, promptTokens, completionTokens *int64) {
	if o.m == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("model", o.Model),
		attribute.String("operation", operation),
		attribute.String("status", status),
		attribute.String("finish_reason", finishReason),
		attribute.Int("tool_count", toolCount),
	}
	elapsed := time.Since(start).Seconds()
	o.m.requestDuration.Record(context.Background(), elapsed, metric.WithAttributes(attrs...))
	if ttft != nil {
		o.m.ttft.Record(context.Background(), ttft.Seconds(), metric.WithAttributes(attrs...))
	}
	if promptTokens != nil {
		o.m.inputTokens.Record(context.Background(), *promptTokens, metric.WithAttributes(attrs...))
	}
	if completionTokens != nil {
		o.m.outputTokens.Record(context.Background(), *completionTokens, metric.WithAttributes(attrs...))
		if ttft != nil && *completionTokens > 1 {
			remaining := elapsed - ttft.Seconds()
			if remaining < 0 {
				remaining = 0
			}
			o.m.interToken.Record(context.Background(), remaining/float64(*completionTokens-1), metric.WithAttributes(attrs...))
		}
	}
	o.m.requestCount.Add(context.Background(), 1, metric.WithAttributes(attrs...))
}

func (o *OpenAI) Stream(
	ctx context.Context,
	msgs []conversation.LLMMessage,
	tools []conversation.Tool,
	onText func(string),
) (string, []conversation.ToolCall, *conversation.TokenUsage, error) {
	oaMsgs, err := toOpenAIMessages(msgs)
	if err != nil {
		return "", nil, nil, fmt.Errorf("translate llm messages: %w", err)
	}
	if payload, mErr := json.Marshal(oaMsgs); mErr == nil {
		preview := string(payload)
		if len(preview) > 4096 {
			preview = preview[:4096] + "...(truncated, total " + fmt.Sprintf("%d", len(payload)) + " bytes)"
		}
		slog.InfoContext(ctx, "openai chat request", "model", o.Model, "messages", preview)
	}

	stream := o.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    o.Model,
		Messages: oaMsgs,
		Tools:    toOpenAITools(tools),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	})

	start := time.Now()
	var (
		full         string
		accByIdx     = make(map[int64]*toolCallAcc)
		finished     bool
		firstDelta   bool
		ttft         time.Duration
		promptTok    int64
		outputTok    int64
		finishReason string
	)
	for stream.Next() {
		chunk := stream.Current()

		if chunk.Usage.PromptTokens > 0 {
			promptTok = chunk.Usage.PromptTokens
		}
		if chunk.Usage.CompletionTokens > 0 {
			outputTok = chunk.Usage.CompletionTokens
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		if !firstDelta && (choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0) {
			ttft = time.Since(start)
			firstDelta = true
		}
		if delta := choice.Delta.Content; delta != "" {
			full += delta
			if onText != nil {
				onText(delta)
			}
		}
		for _, dtc := range choice.Delta.ToolCalls {
			acc, ok := accByIdx[dtc.Index]
			if !ok {
				acc = &toolCallAcc{}
				accByIdx[dtc.Index] = acc
			}
			if dtc.ID != "" {
				acc.id = dtc.ID
			}
			if dtc.Function.Name != "" {
				acc.name += dtc.Function.Name
			}
			if dtc.Function.Arguments != "" {
				acc.args += dtc.Function.Arguments
			}
		}
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
		if choice.FinishReason == "tool_calls" {
			finished = true
		}
	}
	if err := stream.Err(); err != nil {
		o.recordStreamMetrics(start, ttft, promptTok, outputTok, "error", finishReason)
		return full, nil, buildUsage(promptTok, outputTok), errors.Join(errors.New("openai stream"), err)
	}
	if !finished {
		o.recordStreamMetrics(start, ttft, promptTok, outputTok, "success", finishReason)
		return full, nil, buildUsage(promptTok, outputTok), nil
	}
	toolCalls := make([]conversation.ToolCall, 0, len(accByIdx))
	for _, acc := range accByIdx {
		toolCalls = append(toolCalls, conversation.ToolCall{
			ID:        acc.id,
			Name:      acc.name,
			Arguments: acc.args,
		})
	}
	o.recordStreamMetrics(start, ttft, promptTok, outputTok, "success_tool", finishReason, len(toolCalls))
	return full, toolCalls, buildUsage(promptTok, outputTok), nil
}

func (o *OpenAI) recordStreamMetrics(start time.Time, ttft time.Duration, promptTok, outputTok int64, status, finishReason string, toolCount ...int) {
	count := 0
	if len(toolCount) > 0 {
		count = toolCount[0]
	}
	var ttftPtr *time.Duration
	if ttft > 0 {
		ttftPtr = &ttft
	}
	var pt, ot *int64
	if promptTok > 0 {
		pt = &promptTok
	}
	if outputTok > 0 {
		ot = &outputTok
	}
	o.recordResult("chat", status, finishReason, count, start, ttftPtr, pt, ot)
}

func (o *OpenAI) Complete(ctx context.Context, prompt string) (string, error) {
	start := time.Now()
	resp, err := o.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		o.recordResult("complete", "error", "", 0, start, nil, nil, nil)
		return "", fmt.Errorf("openai complete: %w", err)
	}
	if len(resp.Choices) == 0 {
		o.recordResult("complete", "success_empty", "", 0, start, nil, nil, nil)
		return "", nil
	}
	var pt, ot *int64
	if resp.Usage.PromptTokens > 0 {
		v := resp.Usage.PromptTokens
		pt = &v
	}
	if resp.Usage.CompletionTokens > 0 {
		v := resp.Usage.CompletionTokens
		ot = &v
	}
	o.recordResult("complete", "success", resp.Choices[0].FinishReason, 0, start, nil, pt, ot)
	return resp.Choices[0].Message.Content, nil
}

type toolCallAcc struct {
	id, name, args string
}

func buildUsage(promptTok, outputTok int64) *conversation.TokenUsage {
	if promptTok == 0 && outputTok == 0 {
		return nil
	}
	return &conversation.TokenUsage{PromptTokens: promptTok, CompletionTokens: outputTok}
}

func toOpenAITools(tools []conversation.Tool) []openai.ChatCompletionToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		spec := t.Spec()
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        spec.Name,
			Description: openai.String(spec.Description),
			Parameters:  spec.Parameters,
		}))
	}
	return out
}

func toOpenAIMessages(msgs []conversation.LLMMessage) ([]openai.ChatCompletionMessageParamUnion, error) {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			out = append(out, openai.SystemMessage(m.Content))
		case "user":
			out = append(out, openai.UserMessage(m.Content))
		case "assistant":
			am := openai.ChatCompletionAssistantMessageParam{}

			if m.Content != "" {
				am.Content.OfString = openai.String(m.Content)
			} else if len(m.ToolCalls) > 0 {
				am.Content.OfString = openai.String("")
			}
			for _, c := range m.ToolCalls {
				args := c.Arguments
				if args == "" {
					args = "{}"
				}
				am.ToolCalls = append(am.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: c.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      c.Name,
							Arguments: args,
						},
					},
				})
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &am})
		case "tool":
			if m.ToolCallID == nil {
				continue
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					Content: openai.ChatCompletionToolMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
					ToolCallID: *m.ToolCallID,
				},
			})
		default:
		}
	}
	return out, nil
}
