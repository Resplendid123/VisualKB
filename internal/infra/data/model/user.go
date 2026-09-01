package model

import "time"

type User struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name         string    `gorm:"column:name;size:64;not null"`
	Email        string    `gorm:"column:email;size:128;uniqueIndex;not null"`
	PasswordHash string    `gorm:"column:password_hash;size:128;not null;default:''"`
	Immutable    string    `gorm:"column:immutable;not null;default:''"`
	Mutable      string    `gorm:"column:mutable;not null;default:''"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string { return "users" }
