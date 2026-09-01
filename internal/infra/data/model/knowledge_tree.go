package model

import "time"

type KnowledgeTree struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;index:idx_user_parent,priority:1"`
	ParentID  *int64    `gorm:"column:parent_id;index:idx_user_parent,priority:2"`
	Name      string    `gorm:"column:name;size:64"`
	IsFolder  bool      `gorm:"column:is_folder;not null;default:false"`
	DocID     *int64    `gorm:"column:doc_id;index"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
}

func (KnowledgeTree) TableName() string { return "knowledge_tree" }
