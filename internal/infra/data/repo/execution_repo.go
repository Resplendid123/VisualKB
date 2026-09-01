package repo

import (
	"context"
	"time"

	"learn/internal/domain/project"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type executionRepo struct {
	db *gorm.DB
}

func NewExecutionRepo(db *gorm.DB) project.ExecutionRepo {
	return &executionRepo{db: db}
}

func (r *executionRepo) Create(ctx context.Context, e *project.SandboxExecution) error {
	m := executionToModel(e)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}

	e.ID = m.ID
	return nil
}

func (r *executionRepo) FindByID(ctx context.Context, id string) (*project.SandboxExecution, error) {
	var m model.SandboxExecution
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		return nil, err
	}
	return executionToDomain(&m), nil
}

func (r *executionRepo) MarkFinished(
	ctx context.Context, id, status string,
	exitCode *int, durationMs int64,
	outputTail *string,
) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.SandboxExecution{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      status,
			"exit_code":   exitCode,
			"duration_ms": durationMs,
			"output_tail": outputTail,
			"finished_at": now,
		})
	return res.Error
}

func executionToDomain(m *model.SandboxExecution) *project.SandboxExecution {
	return &project.SandboxExecution{
		ID:             m.ID,
		SandboxPodID:   m.SandboxPodID,
		UserID:         m.UserID,
		ProjectID:      m.ProjectID,
		ConversationID: m.ConversationID,
		MessageID:      m.MessageID,
		Command:        m.Command,
		ExitCode:       m.ExitCode,
		DurationMs:     m.DurationMs,
		Status:         m.Status,
		OutputTail:     m.OutputTail,
		CreatedAt:      m.CreatedAt,
		FinishedAt:     m.FinishedAt,
	}
}

func executionToModel(e *project.SandboxExecution) *model.SandboxExecution {
	return &model.SandboxExecution{
		ID:             e.ID,
		SandboxPodID:   e.SandboxPodID,
		UserID:         e.UserID,
		ProjectID:      e.ProjectID,
		ConversationID: e.ConversationID,
		MessageID:      e.MessageID,
		Command:        e.Command,
		ExitCode:       e.ExitCode,
		DurationMs:     e.DurationMs,
		Status:         e.Status,
		OutputTail:     e.OutputTail,
		CreatedAt:      e.CreatedAt,
		FinishedAt:     e.FinishedAt,
	}
}

var _ project.ExecutionRepo = (*executionRepo)(nil)
