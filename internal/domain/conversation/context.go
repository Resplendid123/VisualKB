package conversation

import "context"

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeyConversationID
	ctxKeyMessageID
	ctxKeyProjectID
	ctxKeyEventSink
	ctxKeyToolCallID
)

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(int64)
	return v, ok
}

func WithConversationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyConversationID, id)
}

func ConversationIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyConversationID).(string)
	return v, ok
}

func WithMessageID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyMessageID, id)
}

func MessageIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyMessageID).(string)
	return v, ok
}

func WithProjectID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyProjectID, id)
}

func ProjectIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyProjectID).(string)
	return v, ok
}

func WithEventSink(ctx context.Context, sink EventSink) context.Context {
	return context.WithValue(ctx, ctxKeyEventSink, sink)
}

func EventSinkFromContext(ctx context.Context) (EventSink, bool) {
	v, ok := ctx.Value(ctxKeyEventSink).(EventSink)
	return v, ok
}

func WithToolCallID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyToolCallID, id)
}

func ToolCallIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyToolCallID).(string)
	return v, ok
}
