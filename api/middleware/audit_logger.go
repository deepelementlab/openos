package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AuditLevel defines the level of audit logging
type AuditLevel int

const (
	// AuditLevelNone disables audit logging
	AuditLevelNone AuditLevel = iota
	// AuditLevelMinimal logs only authentication and authorization failures
	AuditLevelMinimal
	// AuditLevelNormal logs authentication, authorization, and key operations
	AuditLevelNormal
	// AuditLevelVerbose logs all security-relevant operations
	AuditLevelVerbose
)

// AuditEventType defines the type of audit event
type AuditEventType string

const (
	// EventTypeAuthentication represents authentication events
	EventTypeAuthentication AuditEventType = "authentication"
	// EventTypeAuthorization represents authorization events
	EventTypeAuthorization AuditEventType = "authorization"
	// EventTypeUserOperation represents user operations
	EventTypeUserOperation AuditEventType = "user_operation"
	// EventTypeSystemOperation represents system operations
	EventTypeSystemOperation AuditEventType = "system_operation"
	// EventTypeSecurityViolation represents security violations
	EventTypeSecurityViolation AuditEventType = "security_violation"
	// EventTypeConfigurationChange represents configuration changes
	EventTypeConfigurationChange AuditEventType = "configuration_change"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID           string          `json:"id"`
	Type         AuditEventType  `json:"type"`
	Timestamp    time.Time       `json:"timestamp"`
	UserID       string          `json:"user_id,omitempty"`
	Username     string          `json:"username,omitempty"`
	TenantID     string          `json:"tenant_id,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	Resource     string          `json:"resource,omitempty"`
	Action       string          `json:"action,omitempty"`
	Method       string          `json:"method,omitempty"`
	Path         string          `json:"path,omitempty"`
	Status       int             `json:"status,omitempty"`
	Success      bool            `json:"success"`
	Reason       string          `json:"reason,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
}

// AuditLogger provides structured audit logging
type AuditLogger interface {
	LogEvent(ctx context.Context, event AuditEvent) error
	LogAuthentication(ctx context.Context, success bool, userID, username, reason string) error
	LogAuthorization(ctx context.Context, success bool, userID, resource, action, reason string) error
	LogHTTPRequest(ctx context.Context, r *http.Request, status int, userID string) error
	SetLevel(level AuditLevel)
	GetLevel() AuditLevel
}

// ZapAuditLogger implements AuditLogger using zap logger
type ZapAuditLogger struct {
	logger *zap.Logger
	level  AuditLevel
	mu     sync.RWMutex
}

// NewZapAuditLogger creates a new audit logger
func NewZapAuditLogger(logger *zap.Logger, level AuditLevel) *ZapAuditLogger {
	return &ZapAuditLogger{
		logger: logger.With(zap.String("component", "audit")),
		level:  level,
	}
}

// LogEvent logs a structured audit event
func (l *ZapAuditLogger) LogEvent(ctx context.Context, event AuditEvent) error {
	if l.level == AuditLevelNone {
		return nil
	}

	// Filter events based on audit level
	if !l.shouldLogEvent(event.Type, event.Success) {
		return nil
	}

	// Add context fields
	fields := []zap.Field{
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)),
		zap.Time("timestamp", event.Timestamp),
		zap.Bool("success", event.Success),
	}

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}
	if event.Username != "" {
		fields = append(fields, zap.String("username", event.Username))
	}
	if event.TenantID != "" {
		fields = append(fields, zap.String("tenant_id", event.TenantID))
	}
	if event.IPAddress != "" {
		fields = append(fields, zap.String("ip_address", event.IPAddress))
	}
	if event.Resource != "" {
		fields = append(fields, zap.String("resource", event.Resource))
	}
	if event.Action != "" {
		fields = append(fields, zap.String("action", event.Action))
	}
	if event.Reason != "" {
		fields = append(fields, zap.String("reason", event.Reason))
	}
	if event.CorrelationID != "" {
		fields = append(fields, zap.String("correlation_id", event.CorrelationID))
	}
	if len(event.Details) > 0 {
		fields = append(fields, zap.ByteString("details", event.Details))
	}

	// Determine log level based on event type and success
	var logFunc func(string, ...zap.Field)
	if !event.Success && l.level >= AuditLevelMinimal {
		logFunc = l.logger.Error
	} else if l.level >= AuditLevelNormal {
		logFunc = l.logger.Info
	} else if l.level >= AuditLevelVerbose {
		logFunc = l.logger.Debug
	} else {
		return nil
	}

	logFunc(fmt.Sprintf("audit_event_%s", string(event.Type)), fields...)
	return nil
}

// LogAuthentication logs authentication events
func (l *ZapAuditLogger) LogAuthentication(ctx context.Context, success bool, userID, username, reason string) error {
	if l.level < AuditLevelMinimal {
		return nil
	}

	event := AuditEvent{
		ID:        generateEventID(),
		Type:      EventTypeAuthentication,
		Timestamp: time.Now(),
		UserID:    userID,
		Username:  username,
		Success:   success,
		Reason:    reason,
	}

	// Extract IP address and user agent from context if available
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		event.IPAddress = getIPAddress(req)
		event.UserAgent = req.UserAgent()
		event.Path = req.URL.Path
		event.Method = req.Method
	}

	// Extract session ID from context
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		event.SessionID = sessionID
	}

	return l.LogEvent(ctx, event)
}

// LogAuthorization logs authorization events
func (l *ZapAuditLogger) LogAuthorization(ctx context.Context, success bool, userID, resource, action, reason string) error {
	if l.level < AuditLevelNormal {
		return nil
	}

	event := AuditEvent{
		ID:        generateEventID(),
		Type:      EventTypeAuthorization,
		Timestamp: time.Now(),
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		Success:   success,
		Reason:    reason,
	}

	// Extract additional context
	if username, ok := ctx.Value("username").(string); ok {
		event.Username = username
	}
	if tenantID, ok := ctx.Value("tenant_id").(string); ok {
		event.TenantID = tenantID
	}
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		event.IPAddress = getIPAddress(req)
		event.UserAgent = req.UserAgent()
		event.Path = req.URL.Path
		event.Method = req.Method
	}

	return l.LogEvent(ctx, event)
}

// LogHTTPRequest logs HTTP request events
func (l *ZapAuditLogger) LogHTTPRequest(ctx context.Context, r *http.Request, status int, userID string) error {
	if l.level < AuditLevelVerbose {
		return nil
	}

	event := AuditEvent{
		ID:        generateEventID(),
		Type:      EventTypeUserOperation,
		Timestamp: time.Now(),
		UserID:    userID,
		IPAddress: getIPAddress(r),
		UserAgent: r.UserAgent(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Status:    status,
		Success:   status < 400,
	}

	// Extract query parameters for auditing (excluding sensitive data)
	query := r.URL.Query()
	safeQuery := make(map[string][]string)
	for key, values := range query {
		// Filter out sensitive query parameters
		if !isSensitiveQueryParam(key) {
			safeQuery[key] = values
		}
	}

	if len(safeQuery) > 0 {
		queryJSON, _ := json.Marshal(safeQuery)
		event.Details = queryJSON
	}

	return l.LogEvent(ctx, event)
}

// SetLevel sets the audit logging level
func (l *ZapAuditLogger) SetLevel(level AuditLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current audit logging level
func (l *ZapAuditLogger) GetLevel() AuditLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// shouldLogEvent determines if an event should be logged based on audit level
func (l *ZapAuditLogger) shouldLogEvent(eventType AuditEventType, success bool) bool {
	switch l.level {
	case AuditLevelNone:
		return false
	case AuditLevelMinimal:
		return eventType == EventTypeAuthentication && !success
	case AuditLevelNormal:
		return eventType == EventTypeAuthentication ||
			eventType == EventTypeAuthorization ||
			(!success && eventType == EventTypeUserOperation)
	case AuditLevelVerbose:
		return true
	default:
		return false
	}
}

// getIPAddress extracts client IP address from request
func getIPAddress(r *http.Request) string {
	// Check for X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Use the first IP in the chain
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// isSensitiveQueryParam checks if a query parameter contains sensitive data
func isSensitiveQueryParam(key string) bool {
	sensitiveKeys := []string{
		"password",
		"token",
		"secret",
		"key",
		"auth",
		"credential",
		"jwt",
		"api_key",
	}

	lowerKey := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lowerKey, sensitive) {
			return true
		}
	}
	return false
}

// NewAuditMiddleware creates middleware for HTTP request auditing
func NewAuditMiddleware(auditLogger AuditLogger, level AuditLevel) func(http.Handler) http.Handler {
	auditLogger.SetLevel(level)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create response wrapper to capture status code
			rw := &responseWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			// Add request to context
			ctx := context.WithValue(r.Context(), "http_request", r)
			
			// Process request
			next.ServeHTTP(rw, r.WithContext(ctx))
			
			// Log request if audit level is high enough
			if level >= AuditLevelVerbose {
				// Extract user ID from context if available
				userID := ""
				if ctxUserID, ok := ctx.Value("user_id").(string); ok {
					userID = ctxUserID
				}
				
				// Log HTTP request
				_ = auditLogger.LogHTTPRequest(ctx, r, rw.statusCode, userID)
			}
		})
	}
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// NullAuditLogger implements AuditLogger but does nothing
type NullAuditLogger struct{}

func (n *NullAuditLogger) LogEvent(ctx context.Context, event AuditEvent) error { return nil }
func (n *NullAuditLogger) LogAuthentication(ctx context.Context, success bool, userID, username, reason string) error { return nil }
func (n *NullAuditLogger) LogAuthorization(ctx context.Context, success bool, userID, resource, action, reason string) error { return nil }
func (n *NullAuditLogger) LogHTTPRequest(ctx context.Context, r *http.Request, status int, userID string) error { return nil }
func (n *NullAuditLogger) SetLevel(level AuditLevel) {}
func (n *NullAuditLogger) GetLevel() AuditLevel { return AuditLevelNone }