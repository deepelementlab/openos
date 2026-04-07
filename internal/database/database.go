package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"
)

// Database represents the database connection and operations
type Database struct {
	DB     *sqlx.DB
	Logger *zap.Logger
}

// Config holds database configuration
type Config struct {
	Host              string        `json:"host" mapstructure:"host"`
	Port              int           `json:"port" mapstructure:"port"`
	User              string        `json:"user" mapstructure:"user"`
	Password          string        `json:"password" mapstructure:"password"`
	Database          string        `json:"database" mapstructure:"database"`
	SSLMode           string        `json:"ssl_mode" mapstructure:"ssl_mode"`
	MaxOpenConns      int           `json:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns      int           `json:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `json:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	ConnectionTimeout time.Duration `json:"connection_timeout" mapstructure:"connection_timeout"`
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:              "localhost",
		Port:              5432,
		User:              "postgres",
		Password:          "postgres",
		Database:          "agent_os",
		SSLMode:           "disable",
		MaxOpenConns:      25,
		MaxIdleConns:      5,
		ConnMaxLifetime:   5 * time.Minute,
		ConnectionTimeout: 30 * time.Second,
	}
}

// NewDatabase creates a new Database instance
func NewDatabase(cfg *Config, logger *zap.Logger) (*Database, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
	)

	return &Database{
		DB:     db,
		Logger: logger,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// HealthCheck performs a database health check
func (db *Database) HealthCheck(ctx context.Context) error {
	if db.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Try to ping the database
	if err := db.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Try a simple query
	_, err := db.DB.ExecContext(ctx, "SELECT 1")
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	return nil
}

// BeginTransaction starts a new transaction
func (db *Database) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	return db.DB.BeginTxx(ctx, nil)
}

// ExecInTransaction executes a function within a transaction
func (db *Database) ExecInTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetStats returns database statistics
func (db *Database) GetStats() sql.DBStats {
	return db.DB.Stats()
}

// CreateTables creates all required tables (for development/testing)
func (db *Database) CreateTables(ctx context.Context) error {
	queries := []string{
		// Nodes table
		`CREATE TABLE IF NOT EXISTS nodes (
			id SERIAL PRIMARY KEY,
			node_id VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			address VARCHAR(255) NOT NULL,
			cluster VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			cpu_cores INTEGER NOT NULL,
			memory_bytes BIGINT NOT NULL,
			disk_bytes BIGINT NOT NULL,
			network_speed INTEGER NOT NULL,
			labels JSONB,
			annotations JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE,
			CHECK (status IN ('active', 'inactive', 'draining', 'maintenance'))
		)`,

		// Agents table
		`CREATE TABLE IF NOT EXISTS agents (
			id SERIAL PRIMARY KEY,
			agent_id VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			node_id VARCHAR(255) REFERENCES nodes(node_id),
			image VARCHAR(512) NOT NULL,
			command TEXT[],
			args TEXT[],
			environment JSONB,
			resources JSONB NOT NULL,
			labels JSONB,
			annotations JSONB,
			priority INTEGER DEFAULT 5,
			scheduling_constraints JSONB,
			health_status VARCHAR(50),
			health_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE,
			started_at TIMESTAMP WITH TIME ZONE,
			finished_at TIMESTAMP WITH TIME ZONE,
			CHECK (status IN ('pending', 'creating', 'running', 'stopping', 'stopped', 'error')),
			CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
		)`,

		// Resource pools table
		`CREATE TABLE IF NOT EXISTS resource_pools (
			id SERIAL PRIMARY KEY,
			pool_id VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			labels JSONB,
			total_cpu_cores INTEGER NOT NULL,
			total_memory_bytes BIGINT NOT NULL,
			total_disk_bytes BIGINT NOT NULL,
			allocated_cpu_cores INTEGER DEFAULT 0,
			allocated_memory_bytes BIGINT DEFAULT 0,
			allocated_disk_bytes BIGINT DEFAULT 0,
			max_agents_per_node INTEGER,
			scheduling_policy VARCHAR(100),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE
		)`,

		// Node resources tracking
		`CREATE TABLE IF NOT EXISTS node_resources (
			id SERIAL PRIMARY KEY,
			node_id VARCHAR(255) UNIQUE NOT NULL REFERENCES nodes(node_id),
			allocated_cpu_cores INTEGER DEFAULT 0,
			allocated_memory_bytes BIGINT DEFAULT 0,
			allocated_disk_bytes BIGINT DEFAULT 0,
			available_cpu_cores INTEGER NOT NULL,
			available_memory_bytes BIGINT NOT NULL,
			available_disk_bytes BIGINT NOT NULL,
			agent_count INTEGER DEFAULT 0,
			last_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Scheduler events for auditing
		`CREATE TABLE IF NOT EXISTS scheduler_events (
			id SERIAL PRIMARY KEY,
			event_id VARCHAR(255) UNIQUE NOT NULL,
			event_type VARCHAR(100) NOT NULL,
			agent_id VARCHAR(255),
			node_id VARCHAR(255),
			resource_pool_id VARCHAR(255),
			message TEXT NOT NULL,
			details JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Outbox events for reliable publishing
		`CREATE TABLE IF NOT EXISTS outbox_events (
			id SERIAL PRIMARY KEY,
			event_id VARCHAR(255) UNIQUE NOT NULL,
			event_type VARCHAR(100) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			retry_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			CHECK (status IN ('pending', 'sent', 'dead_letter'))
		)`,

		// Security audit logs
		`CREATE TABLE IF NOT EXISTS security_audit_logs (
			id SERIAL PRIMARY KEY,
			log_id VARCHAR(255) UNIQUE NOT NULL,
			user_id VARCHAR(255),
			username VARCHAR(255),
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100),
			resource_id VARCHAR(255),
			ip_address INET,
			user_agent TEXT,
			success BOOLEAN NOT NULL,
			error_message TEXT,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			metadata JSONB
		)`,
	}

	for _, query := range queries {
		if _, err := db.DB.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	db.Logger.Info("Database tables created successfully")
	return nil
}

// CreateIndexes creates indexes for performance optimization
func (db *Database) CreateIndexes(ctx context.Context) error {
	indexes := []string{
		// Nodes indexes
		"CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_cluster ON nodes(cluster)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_labels ON nodes USING GIN(labels)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_created_at ON nodes(created_at)",
		
		// Agents indexes
		"CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status)",
		"CREATE INDEX IF NOT EXISTS idx_agents_node_id ON agents(node_id)",
		"CREATE INDEX IF NOT EXISTS idx_agents_priority ON agents(priority)",
		"CREATE INDEX IF NOT EXISTS idx_agents_created_at ON agents(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_agents_labels ON agents USING GIN(labels)",
		"CREATE INDEX IF NOT EXISTS idx_agents_health_status ON agents(health_status)",
		
		// Resource pools indexes
		"CREATE INDEX IF NOT EXISTS idx_resource_pools_pool_id ON resource_pools(pool_id)",
		
		// Node resources indexes
		"CREATE INDEX IF NOT EXISTS idx_node_resources_node_id ON node_resources(node_id)",
		
		// Scheduler events indexes
		"CREATE INDEX IF NOT EXISTS idx_scheduler_events_event_type ON scheduler_events(event_type)",
		"CREATE INDEX IF NOT EXISTS idx_scheduler_events_agent_id ON scheduler_events(agent_id)",
		"CREATE INDEX IF NOT EXISTS idx_scheduler_events_node_id ON scheduler_events(node_id)",
		"CREATE INDEX IF NOT EXISTS idx_scheduler_events_created_at ON scheduler_events(created_at)",

		// Outbox indexes
		"CREATE INDEX IF NOT EXISTS idx_outbox_events_status ON outbox_events(status)",
		"CREATE INDEX IF NOT EXISTS idx_outbox_events_created_at ON outbox_events(created_at)",
		
		// Security audit logs indexes
		"CREATE INDEX IF NOT EXISTS idx_security_audit_logs_user_id ON security_audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_security_audit_logs_action ON security_audit_logs(action)",
		"CREATE INDEX IF NOT EXISTS idx_security_audit_logs_resource_type ON security_audit_logs(resource_type)",
		"CREATE INDEX IF NOT EXISTS idx_security_audit_logs_timestamp ON security_audit_logs(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_security_audit_logs_success ON security_audit_logs(success)",
	}

	for _, index := range indexes {
		if _, err := db.DB.ExecContext(ctx, index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	db.Logger.Info("Database indexes created successfully")
	return nil
}

// Migration represents a database migration
type Migration struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	AppliedAt time.Time `db:"applied_at"`
}

// CreateMigrationsTable creates the migrations tracking table
func (db *Database) CreateMigrationsTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS migrations (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.DB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// ApplyMigration applies a single migration
func (db *Database) ApplyMigration(ctx context.Context, name string, query string) error {
	// Check if migration already applied
	var count int
	err := db.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM migrations WHERE name = $1", name)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		db.Logger.Debug("Migration already applied", zap.String("name", name))
		return nil
	}

	// Apply migration
	tx, err := db.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO migrations (name) VALUES ($1)", name); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	db.Logger.Info("Migration applied successfully", zap.String("name", name))
	return nil
}