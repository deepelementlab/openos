package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewMigrationManager(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	assert.NotNil(t, mm)
	assert.Empty(t, mm.Status())
}

func TestMigrationManager_Register(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())

	err := mm.Register(MigrationRecord{
		ID:    "001",
		Name:  "create_users",
		UpSQL: "CREATE TABLE users (id SERIAL PRIMARY KEY)",
	})
	require.NoError(t, err)

	status := mm.Status()
	assert.Len(t, status, 1)
	assert.Equal(t, "001", status[0].ID)
	assert.Equal(t, "create_users", status[0].Name)
	assert.Nil(t, status[0].AppliedAt)
}

func TestMigrationManager_RegisterDuplicate(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())

	mm.Register(MigrationRecord{ID: "001", Name: "first", UpSQL: "SELECT 1"})
	err := mm.Register(MigrationRecord{ID: "001", Name: "duplicate", UpSQL: "SELECT 1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestMigrationManager_RegisterEmptyID(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	err := mm.Register(MigrationRecord{Name: "test", UpSQL: "SELECT 1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestMigrationManager_RegisterEmptyName(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	err := mm.Register(MigrationRecord{ID: "001", UpSQL: "SELECT 1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestMigrationManager_RegisterEmptyUpSQL(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	err := mm.Register(MigrationRecord{ID: "001", Name: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "up SQL is required")
}

func TestMigrationManager_Up_NilDB(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	mm.Register(MigrationRecord{ID: "001", Name: "test", UpSQL: "SELECT 1"})
	err := mm.Up(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database is nil")
}

func TestMigrationManager_Down_NilDB(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	err := mm.Down(nil, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database is nil")
}

func TestMigrationManager_PendingCount(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	assert.Equal(t, 0, mm.PendingCount())

	mm.Register(MigrationRecord{ID: "001", Name: "first", UpSQL: "SELECT 1"})
	mm.Register(MigrationRecord{ID: "002", Name: "second", UpSQL: "SELECT 1"})
	assert.Equal(t, 2, mm.PendingCount())

	now := time.Now()
	mm.migrations[0].AppliedAt = &now
	assert.Equal(t, 1, mm.PendingCount())
}

func TestMigrationRecord_Fields(t *testing.T) {
	now := time.Now()
	r := MigrationRecord{
		ID:        "001",
		Name:      "create_table",
		UpSQL:     "CREATE TABLE t (id INT)",
		DownSQL:   "DROP TABLE t",
		AppliedAt: &now,
	}
	assert.Equal(t, "001", r.ID)
	assert.Equal(t, "create_table", r.Name)
	assert.Equal(t, "CREATE TABLE t (id INT)", r.UpSQL)
	assert.Equal(t, "DROP TABLE t", r.DownSQL)
	assert.Equal(t, &now, r.AppliedAt)
}

func TestMigrationManager_Status_ReturnsCopies(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())
	mm.Register(MigrationRecord{ID: "001", Name: "test", UpSQL: "SELECT 1"})

	status := mm.Status()
	status[0].Name = "tampered"

	status2 := mm.Status()
	assert.Equal(t, "test", status2[0].Name)
}

func TestMigrationManager_MultipleRegistrations(t *testing.T) {
	mm := NewMigrationManager(nil, zap.NewNop())

	for i := 0; i < 10; i++ {
		err := mm.Register(MigrationRecord{
			ID:    fmt.Sprintf("%03d", i),
			Name:  fmt.Sprintf("migration_%d", i),
			UpSQL: fmt.Sprintf("SELECT %d", i),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 10, mm.PendingCount())
	assert.Len(t, mm.Status(), 10)
}
