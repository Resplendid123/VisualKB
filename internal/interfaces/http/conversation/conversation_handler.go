package conversation

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	app "learn/internal/application/conversation"
	domainconv "learn/internal/domain/conversation"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	convoSvc *app.ConversationService
}

func NewConversationHandler(convoSvc *app.ConversationService) *ConversationHandler {
	return &ConversationHandler{convoSvc: convoSvc}
}

func (h *ConversationHandler) Create(c *gin.Context) {
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	convo, err := h.convoSvc.Create(c.Request.Context(), userID, req.Title)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toConversationResponse(convo))
}

func (h *ConversationHandler) List(c *gin.Context) {
	var req listConversationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	convos, total, err := h.convoSvc.List(c.Request.Context(), userID, req.Limit, req.Offset)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toListConversationsResponse(convos, total, req.Limit, req.Offset))
}

func (h *ConversationHandler) Get(c *gin.Context) {
	var path getConversationPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	res, err := h.convoSvc.Get(c.Request.Context(), userID, path.ConversationID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toConversationResponse(res))
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	var path getConversationPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	messages, lastTurn, lastSeqID, inFlight, err := h.convoSvc.GetMessages(c.Request.Context(), userID, path.ConversationID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, listMessagesResponse{
		Items:           toListMessagesResponse(messages),
		LastTurnAtLoad:  lastTurn,
		LastSeqIDAtLoad: lastSeqID,
		InFlight:        inFlight,
	})
}

// Archive soft-deletes a conversation.
func (h *ConversationHandler) Archive(c *gin.Context) {
	var path getConversationPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.convoSvc.Archive(c.Request.Context(), userID, path.ConversationID); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ConversationHandler) Events(c *gin.Context) {
	var path getConversationPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	ctx := c.Request.Context()

	var fromTurn, fromSeq int64
	var fromID string
	if content := strings.TrimSpace(c.Query("content")); content != "" {
		// New turn: turnID is the cursor.
		docIDs := parseDocIDs(c.Query("document_ids"))
		edit := c.Query("edit") == "true"
		turnID, _, err := h.convoSvc.Chat(ctx, path.ConversationID, content, docIDs, userID, edit)
		if err != nil {
			slog.WarnContext(ctx, "chat turn failed", "err", err, "conv", path.ConversationID)
			response.Fail(c, err)
			return
		}
		fromTurn = turnID
	} else {

		fromTurn = parseInt64(c.GetHeader("Last-Turn"), c.Query("last_turn"))
		fromSeq = parseInt64(c.GetHeader("Last-Seq-Id"), c.Query("last_seq_id"))
		fromID = c.GetHeader("Last-Event-ID")
		if q := c.Query("last_event_id"); q != "" {
			fromID = q
		}
	}

	response.WriteSSEHeaders(c)
	sse := response.NewSSEWriter(c.Writer)
	defer sse.Close()
	sse.StartHeartbeat(ctx, domainconv.StreamBlockTimeout)

	h.streamSSE(ctx, sse, path.ConversationID, userID, fromID, fromTurn, fromSeq)
}

func (h *ConversationHandler) streamSSE(
	ctx context.Context,
	sse *response.SSEWriter,
	conversationID string,
	userID int64,
	fromID string,
	fromTurn int64,
	fromSeq int64,
) {
	for {
		if ctx.Err() != nil {
			return
		}
		recs, cursor, err := h.convoSvc.Replay(ctx, conversationID, userID, fromID, fromTurn, fromSeq)
		if err != nil {
			slog.WarnContext(ctx, "replay failed", "err", err, "conv", conversationID)
			payload, _ := json.Marshal(map[string]string{"message": err.Error()})
			_ = sse.WriteEvent(domainconv.EventTypeError, payload, "")
			return
		}
		fromID = cursor
		if len(recs) == 0 {
			sse.Ping()
			continue
		}
		for _, rec := range recs {
			if err := sse.WriteEvent(rec.Event.Type, rec.Event.Payload, rec.ID); err != nil {
				return
			}
		}
	}
}

func parseDocIDs(s string) []int64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func parseInt64(headerVal, queryVal string) int64 {
	if v := strings.TrimSpace(headerVal); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	if v := strings.TrimSpace(queryVal); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return 0
}
