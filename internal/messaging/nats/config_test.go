package nats

import (
	"testing"
	"time"

	"github.com/agentos/aos/internal/config"
)

func TestClientConfig_Validate_EmptyURL(t *testing.T) {
	cfg := &ClientConfig{URL: ""}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestClientConfig_Validate_ValidURL(t *testing.T) {
	cfg := &ClientConfig{URL: "nats://localhost:4222"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClientConfig_Validate_InvalidTLS(t *testing.T) {
	cfg := &ClientConfig{
		URL: "nats://localhost:4222",
		TLS: &TLSConfig{
			CertFile: "/nonexistent/cert.pem",
			KeyFile:  "/nonexistent/key.pem",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for nonexistent TLS files")
	}
}

func TestClientConfig_Validate_NilTLS(t *testing.T) {
	cfg := &ClientConfig{
		URL: "nats://localhost:4222",
		TLS: nil,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTLSConfig_BuildTLSConfig_Nil(t *testing.T) {
	var tls *TLSConfig
	result, err := tls.BuildTLSConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil TLSConfig")
	}
}

func TestTLSConfig_BuildTLSConfig_Empty(t *testing.T) {
	tls := &TLSConfig{}
	result, err := tls.BuildTLSConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil TLS config for empty TLSConfig")
	}
}

func TestTLSConfig_BuildTLSConfig_MissingKey(t *testing.T) {
	tls := &TLSConfig{
		CertFile: "/nonexistent/cert.pem",
	}
	result, err := tls.BuildTLSConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil config (cert without key is ignored)")
	}
}

func TestTLSConfig_BuildTLSConfig_InvalidCA(t *testing.T) {
	tls := &TLSConfig{
		CAFile: "/nonexistent/ca.pem",
	}
	_, err := tls.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for nonexistent CA file")
	}
}

func TestNewClientConfig(t *testing.T) {
	cfg := &config.Config{
		Messaging: config.MessagingConfig{
			NATS: config.NATSConfig{
				URL:              "nats://example.com:4222",
				Token:            "test-token",
				ReconnectWait:    5 * time.Second,
				MaxReconnects:    20,
				JetStreamEnabled: true,
				JetStreamDomain:  "test-domain",
			},
		},
	}

	clientCfg := NewClientConfig(cfg)
	if clientCfg.URL != "nats://example.com:4222" {
		t.Errorf("expected nats://example.com:4222, got %s", clientCfg.URL)
	}
	if clientCfg.Token != "test-token" {
		t.Errorf("expected test-token, got %s", clientCfg.Token)
	}
	if clientCfg.ReconnectWait != 5*time.Second {
		t.Errorf("expected 5s, got %v", clientCfg.ReconnectWait)
	}
	if clientCfg.MaxReconnects != 20 {
		t.Errorf("expected 20, got %d", clientCfg.MaxReconnects)
	}
	if !clientCfg.JetStreamEnabled {
		t.Error("expected JetStream enabled")
	}
	if clientCfg.JetStreamDomain != "test-domain" {
		t.Errorf("expected test-domain, got %s", clientCfg.JetStreamDomain)
	}
}

func TestNewClientConfig_WithTLS(t *testing.T) {
	cfg := &config.Config{
		Messaging: config.MessagingConfig{
			NATS: config.NATSConfig{
				URL:     "nats://localhost:4222",
				TLSCert: "/path/to/cert.pem",
				TLSKey:  "/path/to/key.pem",
				TLSCA:   "/path/to/ca.pem",
			},
		},
	}

	clientCfg := NewClientConfig(cfg)
	if clientCfg.TLS == nil {
		t.Error("expected TLS config to be set")
	}
	if clientCfg.TLS.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected cert path, got %s", clientCfg.TLS.CertFile)
	}
	if clientCfg.TLS.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected key path, got %s", clientCfg.TLS.KeyFile)
	}
	if clientCfg.TLS.CAFile != "/path/to/ca.pem" {
		t.Errorf("expected CA path, got %s", clientCfg.TLS.CAFile)
	}
}

func TestNewClientConfig_NoTLS(t *testing.T) {
	cfg := &config.Config{
		Messaging: config.MessagingConfig{
			NATS: config.NATSConfig{
				URL: "nats://localhost:4222",
			},
		},
	}

	clientCfg := NewClientConfig(cfg)
	if clientCfg.TLS != nil {
		t.Error("expected nil TLS config when no TLS files specified")
	}
}
