package ml

import (
	"context"
	"fmt"
	"time"
)

// CheckpointState tracks training checkpoints for resume.
type CheckpointState struct {
	JobID       string
	Step        int64
	Path        string
	CreatedAt   time.Time
	Valid       bool
}

// Store persists checkpoint metadata (object storage integration point).
func Store(ctx context.Context, c CheckpointState) error {
	if c.JobID == "" || c.Path == "" {
		return fmt.Errorf("ml: invalid checkpoint")
	}
	return nil
}
