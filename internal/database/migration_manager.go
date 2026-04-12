package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MigrationManager struct {
	db         *Database
	logger     *zap.Logger
	mu         sync.Mutex
	migrations []MigrationRecord
}

type MigrationRecord struct {
	ID        string
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

func NewMigrationManager(db *Database, logger *zap.Logger) *MigrationManager {
	return &MigrationManager{
		db:         db,
		logger:     logger,
		migrations: make([]MigrationRecord, 0),
	}
}

func (mm *MigrationManager) Register(m MigrationRecord) error {
	if m.ID == "" {
		return fmt.Errorf("migration ID is required")
	}
	if m.Name == "" {
		return fmt.Errorf("migration name is required")
	}
	if m.UpSQL == "" {
		return fmt.Errorf("migration up SQL is required")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, existing := range mm.migrations {
		if existing.ID == m.ID {
			return fmt.Errorf("migration %s already registered", m.ID)
		}
	}

	mm.migrations = append(mm.migrations, m)
	return nil
}

func (mm *MigrationManager) Up(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.db == nil || mm.db.DB == nil {
		return fmt.Errorf("database is nil")
	}

	if err := mm.db.CreateMigrationsTable(ctx); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for i, m := range mm.migrations {
		if m.AppliedAt != nil {
			continue
		}

		mm.logger.Info("applying migration", zap.String("id", m.ID), zap.String("name", m.Name))

		if err := mm.db.ApplyMigration(ctx, m.Name, m.UpSQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.ID, err)
		}

		now := time.Now()
		mm.migrations[i].AppliedAt = &now
		mm.logger.Info("migration applied", zap.String("id", m.ID), zap.String("name", m.Name))
	}

	return nil
}

func (mm *MigrationManager) Down(ctx context.Context, count int) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.db == nil || mm.db.DB == nil {
		return fmt.Errorf("database is nil")
	}

	applied := mm.appliedMigrations()
	if count > len(applied) {
		count = len(applied)
	}

	for i := 0; i < count; i++ {
		m := applied[len(applied)-1-i]
		if m.DownSQL == "" {
			return fmt.Errorf("migration %s has no down SQL", m.ID)
		}

		mm.logger.Info("rolling back migration", zap.String("id", m.ID), zap.String("name", m.Name))

		if _, err := mm.db.DB.ExecContext(ctx, m.DownSQL); err != nil {
			return fmt.Errorf("rollback migration %s: %w", m.ID, err)
		}

		if _, err := mm.db.DB.ExecContext(ctx, "DELETE FROM migrations WHERE name = $1", m.Name); err != nil {
			return fmt.Errorf("remove migration record %s: %w", m.ID, err)
		}

		for j := range mm.migrations {
			if mm.migrations[j].ID == m.ID {
				mm.migrations[j].AppliedAt = nil
				break
			}
		}
	}

	return nil
}

func (mm *MigrationManager) Status() []MigrationRecord {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	out := make([]MigrationRecord, len(mm.migrations))
	copy(out, mm.migrations)
	return out
}

func (mm *MigrationManager) appliedMigrations() []MigrationRecord {
	var applied []MigrationRecord
	for _, m := range mm.migrations {
		if m.AppliedAt != nil {
			applied = append(applied, m)
		}
	}
	return applied
}

func (mm *MigrationManager) PendingCount() int {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	count := 0
	for _, m := range mm.migrations {
		if m.AppliedAt == nil {
			count++
		}
	}
	return count
}
