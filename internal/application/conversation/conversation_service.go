package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"learn/internal/domain"
	domainconv "learn/internal/domain/conversation"
	"learn/internal/domain/document"
	redisinfra "learn/internal/infra/redis"
)

type RenderSkillPrompt func(skills []domainconv.Skill, portrait string) string

type PortraitLoader func(ctx context.Context, userID int64) string

// turnHandle holds ctx, cancels, owner.
type turnHandle struct {
	convID      string
	agentCtx    context.Context
	cancelAgent context.CancelFunc
	cancelTurn  context.CancelFunc
	owner       string
	prevTurnID  int64
}

// ConversationService serializes turns; Redis lock.
type ConversationService struct {
	convoRepo      domainconv.ConvoRepo
	msgRepo        domainconv.MsgRepo
	msgSeqRepo     domainconv.MsgSeqRepo
	stream         domainconv.EventStreamRepo
	cache          domainconv.MessageCacheRepo
	sessionLock    *redisinfra.SessionLock
	cancelBus      *redisinfra.CancelBus
	agent          *Agent
	skillRepo      domainconv.SkillRepo
	render         RenderSkillPrompt
	docRepo        document.DocumentRepo
	portraitLoader PortraitLoader

	// localTurns: cancel listener read-only map.
	localTurns sync.Map
}

func NewConversationService(
	convoRepo domainconv.ConvoRepo,
	msgRepo domainconv.MsgRepo,
	msgSeqRepo domainconv.MsgSeqRepo,
	stream domainconv.EventStreamRepo,
	cache domainconv.MessageCacheRepo,
	sessionLock *redisinfra.SessionLock,
	cancelBus *redisinfra.CancelBus,
	agent *Agent,
	skillRepo domainconv.SkillRepo,
	render RenderSkillPrompt,
	docRepo document.DocumentRepo,
	portraitLoader PortraitLoader,
) *ConversationService {
	return &ConversationService{
		convoRepo:      convoRepo,
		msgRepo:        msgRepo,
		msgSeqRepo:     msgSeqRepo,
		stream:         stream,
		cache:          cache,
		sessionLock:    sessionLock,
		cancelBus:      cancelBus,
		agent:          agent,
		skillRepo:      skillRepo,
		render:         render,
		docRepo:        docRepo,
		portraitLoader: portraitLoader,
	}
}

func (s *ConversationService) Create(ctx context.Context, userID int64, title string) (*domainconv.Conversation, error) {
	c := &domainconv.Conversation{UserID: userID, Title: title}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := s.convoRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationService) List(ctx context.Context, userID int64, limit, offset int) ([]*domainconv.Conversation, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.convoRepo.List(ctx, userID, limit, offset)
}

func (s *ConversationService) Archive(ctx context.Context, userID int64, conversationID string) error {
	return s.convoRepo.Archive(ctx, conversationID, userID)
}

func (s *ConversationService) Get(ctx context.Context, userID int64, conversationID string) (*domainconv.Conversation, error) {
	return s.convoRepo.FindByIDAndUserID(ctx, conversationID, userID)
}

func (s *ConversationService) GetMessages(ctx context.Context, userID int64, conversationID string) ([]*domainconv.Message, int64, int64, bool, error) {
	if _, err := s.convoRepo.FindByIDAndUserID(ctx, conversationID, userID); err != nil {
		return nil, 0, 0, false, err
	}
	msgs, err := s.msgRepo.ListByConversationID(ctx, conversationID)
	if err != nil {
		return nil, 0, 0, false, err
	}
	var lastTurn, lastSeq int64
	if len(msgs) > 0 {
		lastTurn = msgs[len(msgs)-1].TurnID
		lastSeq = msgs[len(msgs)-1].SeqID
	}
	return msgs, lastTurn, lastSeq, s.IsTurnActive(ctx, conversationID), nil
}

// IsTurnActive checks SessionLock for inFlight turn.
func (s *ConversationService) IsTurnActive(ctx context.Context, conversationID string) bool {
	if s.sessionLock == nil {
		return false
	}
	held, err := s.sessionLock.IsHeld(ctx, conversationID)
	if err != nil {
		slog.WarnContext(ctx, "is turn active query failed", "err", err, "conv", conversationID)
		return false
	}
	return held
}

// StartCancelListener subscribes cross-instance cancel bus.
func (s *ConversationService) StartCancelListener(ctx context.Context) func() error {
	if s.cancelBus == nil {
		return func() error { return nil }
	}
	msgCh, cleanup := s.cancelBus.Subscribe(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cm, ok := <-msgCh:
				if !ok {
					return
				}
				s.handleRemoteCancel(ctx, cm)
			}
		}
	}()
	return cleanup
}

func (s *ConversationService) handleRemoteCancel(ctx context.Context, cm redisinfra.CancelMessage) {
	v, ok := s.localTurns.Load(cm.ConvID)
	if !ok {
		// Skip: stale msg or conv not local.
		slog.DebugContext(ctx, "remote cancel for unknown local turn", "conv", cm.ConvID)
		return
	}
	h, ok := v.(*turnHandle)
	if !ok {
		return
	}
	// Cancel agentCtx alongside Lua Release.
	_ = s.sessionLock.Release(context.Background(), cm.ConvID, h.owner)
	h.cancelAgent()
	slog.InfoContext(ctx, "remote cancel applied",
		"conv", cm.ConvID, "reason", cm.Reason, "owner", h.owner)
}

func (s *ConversationService) Chat(
	ctx context.Context,
	conversationID string,
	content string,
	documentIDs []int64,
	userID int64,
	edit bool,
) (int64, int64, error) {
	if strings.TrimSpace(content) == "" {
		return 0, 0, domain.ErrInvalidConvoContent
	}
	convo, err := s.convoRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return 0, 0, err
	}

	h, err := s.acquireTurn(ctx, conversationID, edit)
	if err != nil {
		return 0, 0, err
	}

	if edit && h.prevTurnID > 0 {
		if err := s.msgRepo.MarkModifiedFromTurn(ctx, conversationID, h.prevTurnID); err != nil {
			s.releaseHandle(h)
			return 0, 0, err
		}
		if err := s.cache.Invalidate(ctx, conversationID); err != nil {
			slog.WarnContext(ctx, "msg cache invalidate failed", "err", err, "conv", conversationID)
		}
	}

	turnID, err := s.msgSeqRepo.NextTurn(ctx, conversationID)
	if err != nil {
		s.releaseHandle(h)
		return 0, 0, err
	}

	history, err := s.loadHistory(ctx, conversationID)
	if err != nil {
		s.releaseHandle(h)
		return 0, 0, err
	}

	userSeq, err := s.msgSeqRepo.Next(ctx, conversationID)
	if err != nil {
		s.releaseHandle(h)
		return 0, 0, err
	}
	userMsg := &domainconv.Message{
		ConversationID: conversationID,
		Role:           "user",
		Content:        &content,
		Seq:            userSeq,
		TurnID:         turnID,
		SeqID:          1,
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		s.releaseHandle(h)
		return 0, 0, err
	}
	if err := s.cache.Push(ctx, userMsg); err != nil {
		slog.WarnContext(ctx, "msg cache push (user) failed", "err", err)
	}

	// Prepend doc refs to agent content.
	agentContent := content
	if len(documentIDs) > 0 {
		refs := make([]string, 0, len(documentIDs))
		for _, id := range documentIDs {
			ref := fmt.Sprintf("#%d", id)
			if s.docRepo != nil {
				if d, err := s.docRepo.FindByID(ctx, id); err == nil && d != nil {
					ref = fmt.Sprintf("#%d titled %q", d.ID, d.Title)
				}
			}
			refs = append(refs, ref)
		}
		agentContent = fmt.Sprintf(
			"[User attached %d documents: %s. Call read_document for any as needed.]\n\n%s",
			len(refs), strings.Join(refs, ", "), content,
		)
	}

	es := newFanoutSink(s.stream, conversationID, turnID)

	// Inject per-turn ctx values for tools.
	agentCtx := h.agentCtx
	agentCtx = domainconv.WithUserID(agentCtx, userID)
	agentCtx = domainconv.WithConversationID(agentCtx, conversationID)
	agentCtx = domainconv.WithMessageID(agentCtx, userMsg.ID)
	if convo != nil && convo.ActiveProjectID != nil && *convo.ActiveProjectID != "" {
		agentCtx = domainconv.WithProjectID(agentCtx, *convo.ActiveProjectID)
	}

	var skills []domainconv.Skill
	if s.skillRepo != nil {
		skills = s.skillRepo.List()
	}
	var portrait string
	if s.portraitLoader != nil {
		portrait = s.portraitLoader(ctx, userID)
	}
	var systemPrompt string
	if s.render != nil {
		systemPrompt = s.render(skills, portrait)
	}

	ms := newPersistSink(conversationID, userMsg.ID, turnID, s.msgRepo, s.msgSeqRepo, s.cache)
	// Pass sessionLock to persist for heartbeat.
	ms.sessionLock = s.sessionLock
	ms.lockConvID = conversationID
	ms.lockOwner = h.owner

	go func() {
		defer s.releaseHandle(h)
		defer func() {
			if agentCtx.Err() == nil {
				return
			}
			if err := ms.AbortSysMsg(context.Background(), turnID); err != nil {
				slog.WarnContext(ctx, "abort sys msg failed", "err", err, "conv", conversationID)
			}
		}()
		if err := s.agent.Run(agentCtx, systemPrompt, history, agentContent, es, ms, conversationID, turnID); err != nil {
			slog.WarnContext(ctx, "agent run failed", "err", err, "conv", conversationID)
		}
	}()

	return turnID, userSeq, nil
}

// Replay streams events, filters by turn/seq.
func (s *ConversationService) Replay(
	ctx context.Context,
	conversationID string,
	userID int64,
	fromID string,
	lastTurn int64,
	lastSeq int64,
) ([]domainconv.EventRecord, string, error) {
	if _, err := s.convoRepo.FindByIDAndUserID(ctx, conversationID, userID); err != nil {
		return nil, fromID, err
	}
	records, err := s.stream.Read(ctx, conversationID, fromID, domainconv.StreamBatchSize, domainconv.StreamBlockTimeout)
	if err != nil {
		return nil, fromID, err
	}
	cursor := fromID
	if n := len(records); n > 0 {
		cursor = records[n-1].ID
	}
	out := make([]domainconv.EventRecord, 0, len(records))
	for _, r := range records {
		if r.Event.TurnID > lastTurn ||
			(r.Event.TurnID == lastTurn && r.Event.SeqID > lastSeq) {
			out = append(out, r)
		}
	}
	return out, cursor, nil
}

// loadHistory: cache first, DB fallback.
func (s *ConversationService) loadHistory(ctx context.Context, conversationID string) ([]*domainconv.Message, error) {
	cached, err := s.cache.List(ctx, conversationID)
	if err == nil && len(cached) > 0 {
		return cached, nil
	}
	if err != nil {
		slog.WarnContext(ctx, "msg cache list failed, falling back to db", "err", err)
	}
	msgs, err := s.msgRepo.ListByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if len(msgs) > 0 {
		if err := s.cache.PushAll(ctx, msgs); err != nil {
			slog.WarnContext(ctx, "msg cache filled failed", "err", err)
		}
	}
	return msgs, nil
}

// acquireTurn grabs lock, handles edit supersede.
func (s *ConversationService) acquireTurn(ctx context.Context, conversationID string, edit bool) (*turnHandle, error) {
	owner := buildOwner(conversationID)

	// cancelTurn clears SSE writer loop on release.
	_, cancelTurn := context.WithCancel(ctx)

	for range 2 {
		lock, err := s.sessionLock.Acquire(ctx, conversationID, owner)
		if err == nil {
			// Got lock; build agentCtx, register local handle.
			agentCtx, cancelAgent := context.WithCancel(context.Background())
			h := &turnHandle{
				convID:      conversationID,
				agentCtx:    agentCtx,
				cancelAgent: cancelAgent,
				cancelTurn:  cancelTurn,
				owner:       lock.Owner(),
			}
			s.localTurns.Store(conversationID, h)
			return h, nil
		}

		if !errors.Is(err, redisinfra.ErrLockNotAcquired) {
			cancelTurn()
			return nil, fmt.Errorf("acquire session lock: %w", err)
		}

		// Lock held by another turn.
		if !edit {
			cancelTurn()
			return nil, domain.ErrConcurrentTurn
		}

		// Edit: notify prev owner, wait release.
		prevOwner, err := s.sessionLock.CurrentOwner(ctx, conversationID)
		if err != nil {
			cancelTurn()
			return nil, fmt.Errorf("read current lock owner: %w", err)
		}
		if prevOwner == "" {
			// Lock released in race window; retry.
			continue
		}
		if prevOwner == owner {
			// Reflexive reentry; refuse to avoid loop.
			cancelTurn()
			return nil, domain.ErrConcurrentTurn
		}
		// Cross-instance supersede cannot read prevTurnID; skip mark-modified.
		var prevTurnID int64
		if v, ok := s.localTurns.Load(conversationID); ok {
			if ph, ok2 := v.(*turnHandle); ok2 && ph.owner == prevOwner {
				prevTurnID = ph.prevTurnID
			}
		}

		if err := s.cancelBus.Publish(ctx, prevOwner, redisinfra.CancelMessage{
			ConvID: conversationID,
			Reason: "edit supersede",
		}); err != nil {
			cancelTurn()
			return nil, fmt.Errorf("publish cancel to prev owner: %w", err)
		}

		// Wait for lock release with timeout.
		if err := s.waitLockReleased(ctx, conversationID, prevOwner, 5*time.Second); err != nil {
			cancelTurn()
			return nil, err
		}

		h := &turnHandle{
			cancelTurn: cancelTurn,
			prevTurnID: prevTurnID,
			owner:      owner,
		}
		return s.tryAcquireAfterCancel(ctx, h, conversationID, owner, cancelTurn)
	}

	cancelTurn()
	return nil, domain.ErrConcurrentTurn
}

// tryAcquireAfterCancel retries after cancel publish.
func (s *ConversationService) tryAcquireAfterCancel(
	ctx context.Context,
	prev *turnHandle,
	conversationID, owner string,
	cancelTurn context.CancelFunc,
) (*turnHandle, error) {
	lock, err := s.sessionLock.Acquire(ctx, conversationID, owner)
	if err != nil {
		cancelTurn()
		if errors.Is(err, redisinfra.ErrLockNotAcquired) {
			return nil, domain.ErrConcurrentTurn
		}
		return nil, err
	}
	agentCtx, cancelAgent := context.WithCancel(context.Background())
	h := &turnHandle{
		convID:      conversationID,
		agentCtx:    agentCtx,
		cancelAgent: cancelAgent,
		cancelTurn:  cancelTurn,
		owner:       lock.Owner(),
		prevTurnID:  prev.prevTurnID,
	}
	s.localTurns.Store(conversationID, h)
	return h, nil
}

// waitLockReleased waits lock release; Exists sufficient.
func (s *ConversationService) waitLockReleased(ctx context.Context, conversationID, _ string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 50 * time.Millisecond
	for {
		held, err := s.sessionLock.IsHeld(ctx, conversationID)
		if err != nil {
			return err
		}
		if !held {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for lock release on conv %s", conversationID)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		if interval < 500*time.Millisecond {
			interval *= 2
		}
	}
}

// releaseHandle clears map, lock, ctx on exit.
func (s *ConversationService) releaseHandle(h *turnHandle) {
	if h == nil {
		return
	}
	s.localTurns.Delete(h.convID)
	ctx := context.Background()
	_ = s.sessionLock.Release(ctx, h.convID, h.owner)
	if h.cancelTurn != nil {
		h.cancelTurn()
	}
}

// buildOwner: <instance>:<pid>:<gid>:0; "0"=placeholder.
func buildOwner(_ string) string {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "local"
	}
	gid := strconv.FormatInt(int64(os.Getpid()), 10) + ":" + strconv.FormatInt(int64(runtime.NumGoroutine()), 10)
	return instanceID + ":" + gid + ":0"
}

type fanoutSink struct {
	stream  domainconv.EventStreamRepo
	convoID string
	turnID  int64
}

func newFanoutSink(stream domainconv.EventStreamRepo, convoID string, turnID int64) *fanoutSink {
	return &fanoutSink{stream: stream, convoID: convoID, turnID: turnID}
}

func (s *fanoutSink) Emit(ctx context.Context, e domainconv.Event) error {
	e.TurnID = s.turnID
	if _, err := s.stream.Append(context.Background(), s.convoID, e); err != nil {
		return err
	}
	return nil
}

type persistSink struct {
	convoID        string
	ownerUserMsgID string
	ownerTurnID    int64
	msgRepo        domainconv.MsgRepo
	seqRepo        domainconv.MsgSeqRepo
	cache          domainconv.MessageCacheRepo
	lastTurnSeq    atomic.Int64

	// Set by Chat; Persist uses for heartbeat.
	sessionLock *redisinfra.SessionLock
	lockConvID  string
	lockOwner   string
}

func newPersistSink(
	convoID, ownerUserMsgID string,
	ownerTurnID int64,
	msgRepo domainconv.MsgRepo,
	seqRepo domainconv.MsgSeqRepo,
	cache domainconv.MessageCacheRepo,
) *persistSink {
	return &persistSink{
		convoID:        convoID,
		ownerUserMsgID: ownerUserMsgID,
		ownerTurnID:    ownerTurnID,
		msgRepo:        msgRepo,
		seqRepo:        seqRepo,
		cache:          cache,
	}
}

func (p *persistSink) AllocTurnSeq(ctx context.Context, convoID string, turnID int64) (int64, error) {
	// Per-round heartbeat renewal point.
	if p.sessionLock != nil {
		hbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := p.sessionLock.Heartbeat(hbCtx, p.lockConvID, p.lockOwner); err != nil {
			// Heartbeat failure is non-fatal; caller checks ctx.
			slog.WarnContext(ctx, "session lock heartbeat failed", "err", err,
				"conv", p.lockConvID, "owner", p.lockOwner)
		}
		cancel()
	}
	return p.seqRepo.NextTurnSeq(ctx, convoID, turnID)
}

func (p *persistSink) Persist(ctx context.Context, m *domainconv.Message, turnSeq int64) (int64, error) {
	owner, err := p.msgRepo.FindByID(ctx, p.ownerUserMsgID)
	if err != nil || owner == nil {
		return 0, nil
	}
	if owner.IsModified {
		return 0, nil
	}
	if turnSeq <= 0 {
		return 0, nil
	}
	globalSeq, err := p.seqRepo.Next(ctx, p.convoID)
	if err != nil {
		return 0, err
	}
	m.ConversationID = p.convoID
	m.TurnID = p.ownerTurnID
	m.SeqID = turnSeq
	m.Seq = globalSeq
	if err := p.msgRepo.Create(ctx, m); err != nil {
		return 0, err
	}
	p.lastTurnSeq.Store(turnSeq)
	if p.cache != nil {
		if err := p.cache.Push(ctx, m); err != nil {
			slog.WarnContext(ctx, "msg cache push failed", "err", err, "msg_id", m.ID)
		}
	}
	return globalSeq, nil
}

func (p *persistSink) AbortSysMsg(ctx context.Context, turnID int64) error {
	turnSeq, err := p.seqRepo.NextTurnSeq(ctx, p.convoID, turnID)
	if err != nil {
		return err
	}
	content := "用户中止"
	msg := &domainconv.Message{
		ConversationID: p.convoID,
		Role:           "system",
		Content:        &content,
		TurnID:         turnID,
		SeqID:          turnSeq,
	}
	if err := p.msgRepo.Create(ctx, msg); err != nil {
		return err
	}
	if p.cache != nil {
		if err := p.cache.Push(ctx, msg); err != nil {
			slog.WarnContext(ctx, "msg cache push (abort sys) failed", "err", err)
		}
	}
	return nil
}
