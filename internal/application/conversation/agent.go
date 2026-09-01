package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	domainconv "learn/internal/domain/conversation"
)

type Agent struct {
	llm       domainconv.LLMClient
	tools     domainconv.ToolRegistry
	maxRound  int
	convoRepo domainconv.ConvoRepo
}

// SetConvoRepo wires convo repo for project-context refresh.
func (a *Agent) SetConvoRepo(r domainconv.ConvoRepo) { a.convoRepo = r }

func NewAgent(llm domainconv.LLMClient, tools domainconv.ToolRegistry) *Agent {
	return &Agent{llm: llm, tools: tools, maxRound: domainconv.DefaultMaxRound}
}

// createProjectToolName matches tools.ProjectToolName without import cycle.
const createProjectToolName = "create_project"

// hasCreateProject returns true if calls includes a create_project tool call.
func hasCreateProject(calls []domainconv.ToolCall) bool {
	for _, c := range calls {
		if c.Name == createProjectToolName {
			return true
		}
	}
	return false
}

// refreshProjectCtx re-reads convo to pick up create_project binding.
func (a *Agent) refreshProjectCtx(ctx context.Context, convoID string) context.Context {
	if a.convoRepo == nil {
		return ctx
	}
	userID, ok := domainconv.UserIDFromContext(ctx)
	if !ok {
		return ctx
	}
	convo, err := a.convoRepo.FindByIDAndUserID(ctx, convoID, userID)
	if err != nil || convo == nil || convo.ActiveProjectID == nil {
		return ctx
	}
	return domainconv.WithProjectID(ctx, *convo.ActiveProjectID)
}

func (a *Agent) Run(
	ctx context.Context,
	systemPrompt string,
	history []*domainconv.Message,
	content string,
	es domainconv.EventSink,
	ms domainconv.MessageSink,
	convoID string,
	turnID int64,
) error {
	msgs := buildLLMMessages(history, content, systemPrompt)
	initialLen := len(msgs)

	defer func() {
		if ctx.Err() == nil || ms == nil {
			return
		}
		fulfilled := map[string]struct{}{}
		for _, m := range msgs[initialLen:] {
			if m.Role == "tool" && m.ToolCallID != nil {
				fulfilled[*m.ToolCallID] = struct{}{}
			}
		}
		seen := map[string]struct{}{}
		for _, m := range msgs[initialLen:] {
			if m.Role != "assistant" {
				continue
			}
			for _, c := range m.ToolCalls {
				if _, ok := fulfilled[c.ID]; ok {
					continue
				}
				if _, dup := seen[c.ID]; dup {
					continue
				}
				seen[c.ID] = struct{}{}

				id := c.ID
				name := c.Name
				const txt = "interrupted: user canceled before result was captured"
				tm := newToolMessage(txt, id, name)
				if turnSeq, perr := ms.AllocTurnSeq(context.Background(), convoID, turnID); perr != nil {
					slog.WarnContext(context.Background(), "alloc turn seq for interrupted tool msg failed",
						"err", perr, "tool_call_id", id)
				} else if _, perr := ms.Persist(context.Background(), tm, turnSeq); perr != nil {
					slog.WarnContext(context.Background(), "persist interrupted tool msg failed",
						"err", perr, "tool_call_id", id)
				}
				payload := domainconv.ToolResultPayload{
					ToolCallID: id,
					Name:       name,
					Result:     txt,
					Error:      "interrupted",
				}
				_ = es.Emit(context.Background(), domainconv.Event{
					Type:    domainconv.EventTypeToolRes,
					Payload: mustJSON(payload),
				})
			}
		}
	}()

	for round := 0; round < a.maxRound; round++ {

		var roundTurnSeq int64
		if ms != nil {
			seq, err := ms.AllocTurnSeq(ctx, convoID, turnID)
			if err != nil {
				return es.Emit(ctx, errorEvent("seq_alloc", err))
			}
			roundTurnSeq = seq
		}

		text, calls, usage, err := a.llm.Stream(
			ctx, msgs, a.tools.GetAll(),
			func(delta string) {
				_ = es.Emit(ctx, domainconv.Event{
					Type:    domainconv.EventTypeText,
					Payload: mustJSON(domainconv.TextPayload{Delta: delta}),
					SeqID:   roundTurnSeq,
				})
			},
		)
		if err != nil {
			return es.Emit(ctx, errorEvent("llm_error", err))
		}

		if len(calls) == 0 {
			// Terminal assistant: persist before done.
			m := newAssistantMessage(text, nil, usage)
			if ms != nil {
				if _, perr := ms.Persist(ctx, m, roundTurnSeq); perr != nil {
					slog.WarnContext(ctx, "persist assistant msg failed", "err", perr)
				}
			}
			return es.Emit(ctx, domainconv.Event{
				Type:    domainconv.EventTypeDone,
				Payload: mustJSON(domainconv.DonePayload{MessageID: m.ID}),
				SeqID:   roundTurnSeq,
			})
		}

		m := newAssistantMessage(text, toToolCallData(calls), usage)
		if ms != nil {
			if _, perr := ms.Persist(ctx, m, roundTurnSeq); perr != nil {
				slog.WarnContext(ctx, "persist assistant msg failed", "err", perr)
			}
		}
		msgs = append(msgs, domainconv.AssistantLLMMessage(text, calls))

		if err := a.dispatchCalls(ctx, es, ms, calls, &msgs, roundTurnSeq, convoID, turnID); err != nil {
			return err
		}

		// Pick up new active project after create_project succeeded.
		if hasCreateProject(calls) {
			ctx = a.refreshProjectCtx(ctx, convoID)
		}

		if callsPauseTurn(a.tools, calls) {
			return es.Emit(ctx, domainconv.Event{
				Type:    domainconv.EventTypeDone,
				Payload: mustJSON(domainconv.DonePayload{}),
			})
		}
	}
	return es.Emit(ctx, errorEvent("max_round", fmt.Errorf("agent exceeded %d rounds", a.maxRound)))
}

func (a *Agent) dispatchCalls(
	ctx context.Context,
	es domainconv.EventSink,
	ms domainconv.MessageSink,
	calls []domainconv.ToolCall,
	msgs *[]domainconv.LLMMessage,
	assistantTurnSeq int64,
	convoID string,
	turnID int64,
) error {
	i := 0
	for i < len(calls) {
		if !a.callConcurrent(calls[i]) {
			if err := a.dispatchTool(ctx, es, ms, calls[i], msgs, assistantTurnSeq, convoID, turnID); err != nil {
				return err
			}
			i++
			continue
		}
		j := i + 1
		for j < len(calls) && a.callConcurrent(calls[j]) {
			j++
		}
		if j == i+1 {
			if err := a.dispatchTool(ctx, es, ms, calls[i], msgs, assistantTurnSeq, convoID, turnID); err != nil {
				return err
			}
		} else {
			if err := a.dispatchParallel(ctx, es, ms, calls[i:j], msgs, assistantTurnSeq, convoID, turnID); err != nil {
				return err
			}
		}
		i = j
	}
	return nil
}

func (a *Agent) callConcurrent(c domainconv.ToolCall) bool {
	tool, ok := a.tools.Get(c.Name)
	if !ok {
		return false
	}
	return tool.Traits().Concurrent
}

func callsPauseTurn(tools domainconv.ToolRegistry, calls []domainconv.ToolCall) bool {
	for _, c := range calls {
		if tool, ok := tools.Get(c.Name); ok && tool.Traits().PausesTurn {
			return true
		}
	}
	return false
}

func (a *Agent) dispatchParallel(
	ctx context.Context,
	es domainconv.EventSink,
	ms domainconv.MessageSink,
	calls []domainconv.ToolCall,
	msgs *[]domainconv.LLMMessage,
	assistantTurnSeq int64,
	convoID string,
	turnID int64,
) error {
	// Phase 1: emit tool_call in LLM order.
	for _, c := range calls {
		a.emitToolCall(ctx, es, c, assistantTurnSeq)
	}

	// Phase 2: parallel invoke.
	results := make([]domainconv.Result, len(calls))
	var wg sync.WaitGroup
	for i, c := range calls {
		wg.Add(1)
		go func(i int, c domainconv.ToolCall) {
			defer wg.Done()
			results[i] = a.invokeCall(ctx, es, c)
		}(i, c)
	}
	wg.Wait()

	// Phase 3: commit results in LLM order.
	for i, c := range calls {
		a.commitTool(ctx, es, ms, c, results[i], msgs, convoID, turnID)
	}
	return nil
}

func (a *Agent) emitToolCall(ctx context.Context, es domainconv.EventSink, c domainconv.ToolCall, assistantTurnSeq int64) {
	description := ""
	if tool, ok := a.tools.Get(c.Name); ok {
		if msgFn := tool.Traits().Message; msgFn != nil {
			args := map[string]any{}
			_ = json.Unmarshal([]byte(c.Arguments), &args)
			description = msgFn(args)
		}
	}
	_ = es.Emit(ctx, domainconv.Event{
		Type: domainconv.EventTypeToolCall,
		Payload: mustJSON(domainconv.ToolCallPayload{
			ToolCallID:  c.ID,
			Name:        c.Name,
			Args:        argsOrEmpty(c.Arguments),
			Description: description,
		}),
		SeqID: assistantTurnSeq,
	})
}

// invokeCall runs tool; unknown tools return Result.Error.
func (a *Agent) invokeCall(ctx context.Context, es domainconv.EventSink, c domainconv.ToolCall) domainconv.Result {
	tool, ok := a.tools.Get(c.Name)
	if !ok {
		return domainconv.Result{Error: fmt.Sprintf("unknown tool: %s", c.Name)}
	}

	toolCtx := domainconv.WithEventSink(ctx, es)
	toolCtx = domainconv.WithToolCallID(toolCtx, c.ID)
	return tool.Invoke(toolCtx, argsOrEmpty(c.Arguments))
}

func (a *Agent) commitTool(
	ctx context.Context,
	es domainconv.EventSink,
	ms domainconv.MessageSink,
	c domainconv.ToolCall,
	res domainconv.Result,
	msgs *[]domainconv.LLMMessage,
	convoID string,
	turnID int64,
) {
	payload := domainconv.ToolResultPayload{ToolCallID: c.ID, Name: c.Name, Result: res.Content}
	if res.Error != "" {
		payload.Error = res.Error
	}
	content := toolContent(res)
	*msgs = append(*msgs, domainconv.ToolLLMMessage(content, c.ID))
	tm := newToolMessage(content, c.ID, c.Name)
	if ms != nil {
		turnSeq, err := ms.AllocTurnSeq(ctx, convoID, turnID)
		if err != nil {
			slog.WarnContext(ctx, "alloc turn seq for tool msg failed", "err", err, "tool_call_id", c.ID)
		} else if _, perr := ms.Persist(ctx, tm, turnSeq); perr != nil {
			slog.WarnContext(ctx, "persist tool msg failed", "err", perr, "tool_call_id", c.ID)
		}
	}
	_ = es.Emit(ctx, domainconv.Event{
		Type:    domainconv.EventTypeToolRes,
		Payload: mustJSON(payload),
	})
}

func argsOrEmpty(s string) map[string]any {
	if s == "" {
		return map[string]any{}
	}
	args := map[string]any{}
	_ = json.Unmarshal([]byte(s), &args)
	return args
}

// dispatchTool runs a single tool serially.
func (a *Agent) dispatchTool(
	ctx context.Context,
	es domainconv.EventSink,
	ms domainconv.MessageSink,
	c domainconv.ToolCall,
	msgs *[]domainconv.LLMMessage,
	assistantTurnSeq int64,
	convoID string,
	turnID int64,
) error {
	a.emitToolCall(ctx, es, c, assistantTurnSeq)
	res := a.invokeCall(ctx, es, c)
	a.commitTool(ctx, es, ms, c, res, msgs, convoID, turnID)
	return nil
}

func toolContent(res domainconv.Result) string {
	if res.Error != "" {
		if res.Content == "" {
			return fmt.Sprintf("error: %s", res.Error)
		}
		return fmt.Sprintf("%s\nerror: %s", res.Content, res.Error)
	}
	return res.Content
}

// buildLLMMessages assembles history plus current user input.
func buildLLMMessages(history []*domainconv.Message, content, systemPrompt string) []domainconv.LLMMessage {
	out := make([]domainconv.LLMMessage, 0, len(history)+2)
	out = append(out, domainconv.SystemLLMMessage(systemPrompt))
	for _, m := range history {
		if msg := m.ToLLMMessage(); msg != nil {
			out = append(out, *msg)
		}
	}
	out = append(out, domainconv.UserLLMMessage(content))
	return out
}

func errorEvent(code string, err error) domainconv.Event {
	return domainconv.Event{Type: domainconv.EventTypeError, Payload: mustJSON(domainconv.ErrorPayload{Code: code, Message: err.Error()})}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Warn("event payload marshal failed", "type", fmt.Sprintf("%T", v), "err", err)
		return json.RawMessage("null")
	}
	return b
}

func newAssistantMessage(text string, calls []domainconv.ToolCallData, usage *domainconv.TokenUsage) *domainconv.Message {
	m := &domainconv.Message{Role: "assistant", ToolCalls: calls, TokenUsage: usage}
	if text != "" {
		m.Content = &text
	}
	return m
}

func newToolMessage(content, toolCallID, name string) *domainconv.Message {
	return &domainconv.Message{
		Role:       "tool",
		Content:    &content,
		ToolCallID: &toolCallID,
		ToolName:   &name,
	}
}

func toToolCallData(calls []domainconv.ToolCall) []domainconv.ToolCallData {
	if len(calls) == 0 {
		return nil
	}
	out := make([]domainconv.ToolCallData, len(calls))
	for i, c := range calls {
		args := c.Arguments
		if args == "" {
			args = "{}"
		}
		out[i] = domainconv.ToolCallData{
			ID:   c.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      c.Name,
				Arguments: args,
			},
		}
	}
	return out
}
