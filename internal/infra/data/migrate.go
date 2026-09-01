package data

import (
	"context"

	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

func AutoMigrate(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS pg_search").Error; err != nil {
		return err
	}
	// Dev: drop sandbox tables, recreate on project_id.
	if err := db.WithContext(ctx).Exec(`DROP TABLE IF EXISTS sandbox_pods CASCADE`).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DROP TABLE IF EXISTS sandbox_executions CASCADE`).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.Message{},
		&model.SandboxPod{},
		&model.SandboxExecution{},
		&model.Project{},
		&model.Document{},
		&model.DocumentVersion{},
		&model.Chunk{},
		&model.KnowledgeTree{},
	); err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS document_chunks_embedding_hnsw_idx
		ON document_chunks USING hnsw (embedding vector_cosine_ops)
	`).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS document_chunks_bm25_idx
		ON document_chunks USING bm25 (id, content)
		WITH (key_field='id')
	`).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_msg_conv_turn_seq
		ON messages (conversation_id, turn_id, seq_id)
	`).Error; err != nil {
		return err
	}
	return nil
}
