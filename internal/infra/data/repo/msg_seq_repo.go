package repo

import (
	"context"
	"errors"
	"fmt"

	"learn/internal/domain/conversation"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type msgSeqRepo struct {
	rdb *redis.Client
	db  *gorm.DB
}

func NewMsgSeqRepo(rdb *redis.Client, db *gorm.DB) conversation.MsgSeqRepo {
	return &msgSeqRepo{rdb: rdb, db: db}
}

func msgSeqKey(conversationID string) string    { return "msgs:seq:" + conversationID }
func turnCountKey(conversationID string) string { return "turns:count:" + conversationID }
func turnSeqKey(conversationID string, turnID int64) string {
	return fmt.Sprintf("seq:%s:%d", conversationID, turnID)
}

func (r *msgSeqRepo) Next(ctx context.Context, conversationID string) (int64, error) {
	key := msgSeqKey(conversationID)
	return r.coldStartIncr(ctx, key, conversationID, r.watermarkSeq, "max seq")
}

func (r *msgSeqRepo) NextTurn(ctx context.Context, conversationID string) (int64, error) {
	key := turnCountKey(conversationID)
	return r.coldStartIncr(ctx, key, conversationID, r.watermarkTurn, "max turn_id")
}

func (r *msgSeqRepo) NextTurnSeq(ctx context.Context, conversationID string, turnID int64) (int64, error) {
	key := turnSeqKey(conversationID, turnID)
	wmFn := func(ctx context.Context, cid string) (int64, error) {
		return r.watermarkTurnSeq(ctx, cid, turnID)
	}
	return r.coldStartIncr(ctx, key, conversationID, wmFn, "max seq_id")
}

func (r *msgSeqRepo) coldStartIncr(
	ctx context.Context, key, conversationID string,
	watermark func(context.Context, string) (int64, error), wmLabel string,
) (int64, error) {
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("exists %s: %w", key, err)
	}
	if exists == 0 {
		wm, werr := watermark(ctx, conversationID)
		if werr != nil {
			return 0, werr
		}
		val, serr := initSeqScript.Run(ctx, r.rdb, []string{key}, wm).Result()
		if serr != nil {
			return 0, fmt.Errorf("init seq script (%s): %w", wmLabel, serr)
		}
		n, ok := val.(int64)
		if !ok {
			return 0, fmt.Errorf("init seq script returned %T", val)
		}
		return n, nil
	}
	n, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incr %s: %w", key, err)
	}
	return n, nil
}

func (r *msgSeqRepo) watermarkSeq(ctx context.Context, conversationID string) (int64, error) {
	if r.db == nil {
		return 0, errors.New("msg_seq_repo: db is nil")
	}
	var wm int64
	err := r.db.WithContext(ctx).
		Table("messages").
		Select("COALESCE(MAX(seq), 0)").
		Where("conversation_id = ?::uuid", conversationID).
		Scan(&wm).Error
	if err != nil {
		return 0, fmt.Errorf("max seq: %w", err)
	}
	return wm, nil
}

func (r *msgSeqRepo) watermarkTurn(ctx context.Context, conversationID string) (int64, error) {
	if r.db == nil {
		return 0, errors.New("msg_seq_repo: db is nil")
	}
	var wm int64
	err := r.db.WithContext(ctx).
		Table("messages").
		Select("COALESCE(MAX(turn_id), 0)").
		Where("conversation_id = ?::uuid", conversationID).
		Scan(&wm).Error
	if err != nil {
		return 0, fmt.Errorf("max turn_id: %w", err)
	}
	return wm, nil
}

func (r *msgSeqRepo) watermarkTurnSeq(ctx context.Context, conversationID string, turnID int64) (int64, error) {
	if r.db == nil {
		return 0, errors.New("msg_seq_repo: db is nil")
	}
	var wm int64
	err := r.db.WithContext(ctx).
		Table("messages").
		Select("COALESCE(MAX(seq_id), 0)").
		Where("conversation_id = ?::uuid AND turn_id = ?", conversationID, turnID).
		Scan(&wm).Error
	if err != nil {
		return 0, fmt.Errorf("max seq_id: %w", err)
	}
	return wm, nil
}

var initSeqScript = redis.NewScript(`
local key = KEYS[1]
local wm = tonumber(ARGV[1])
if redis.call('EXISTS', key) == 0 then
  if wm > 0 then
    redis.call('SET', key, wm)
  else
    redis.call('SET', key, 0)
  end
end
return redis.call('INCR', key)
`)

var _ conversation.MsgSeqRepo = (*msgSeqRepo)(nil)
