package database

import (
	"context"
	"fmt"
	"learn/internal/infra/config"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenDB(ctx context.Context, cfg config.DatabaseConfig) (*gorm.DB, func() error, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("ping database failed: %w", err)
	}

	return db, func() error {
		return sqlDB.Close()
	}, nil
}
