package governance

import (
	"context"
	"fmt"
)

// GDPRRequestType enumerates data subject operations.
type GDPRRequestType string

const (
	ExportData GDPRRequestType = "export"
	DeleteData GDPRRequestType = "erase"
)

// GDPRTicket tracks a compliance workflow.
type GDPRTicket struct {
	TenantID string
	Type     GDPRRequestType
	Subject  string
}

// Process routes GDPR workflows to storage adapters (stub).
func Process(ctx context.Context, t GDPRTicket) error {
	switch t.Type {
	case ExportData, DeleteData:
		return nil
	default:
		return fmt.Errorf("gdpr: unknown request type")
	}
}
