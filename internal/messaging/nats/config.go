// Package nats provides NATS messaging integration for Agent OS.
package nats

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/agentos/aos/internal/config"
)

// ClientConfig wraps configuration needed to establish a NATS connection.
type ClientConfig struct {
	URL              string
	Token            string
	CredentialsFile  string
	TLS              *TLSConfig
	ReconnectWait    time.Duration
	MaxReconnects    int
	JetStreamEnabled bool
	JetStreamDomain  string
}

// TLSConfig holds optional TLS settings.
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// NewClientConfig creates a ClientConfig from the global configuration.
func NewClientConfig(cfg *config.Config) *ClientConfig {
	nc := &ClientConfig{
		URL:              cfg.Messaging.NATS.URL,
		Token:            cfg.Messaging.NATS.Token,
		CredentialsFile:  cfg.Messaging.NATS.CredentialsFile,
		ReconnectWait:    cfg.Messaging.NATS.ReconnectWait,
		MaxReconnects:    cfg.Messaging.NATS.MaxReconnects,
		JetStreamEnabled: cfg.Messaging.NATS.JetStreamEnabled,
		JetStreamDomain:  cfg.Messaging.NATS.JetStreamDomain,
	}

	// Configure TLS if any TLS file is specified
	if cfg.Messaging.NATS.TLSCert != "" || cfg.Messaging.NATS.TLSKey != "" || cfg.Messaging.NATS.TLSCA != "" {
		nc.TLS = &TLSConfig{
			CertFile: cfg.Messaging.NATS.TLSCert,
			KeyFile:  cfg.Messaging.NATS.TLSKey,
			CAFile:   cfg.Messaging.NATS.TLSCA,
		}
	}

	return nc
}

// BuildTLSConfig builds a Go TLS config from TLSConfig.
func (t *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate if provided
	if t.CertFile != "" && t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if t.CAFile != "" {
		caCert, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// Validate checks if the configuration is valid.
func (c *ClientConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("NATS URL is required")
	}

	if c.TLS != nil {
		if _, err := c.TLS.BuildTLSConfig(); err != nil {
			return fmt.Errorf("invalid TLS configuration: %w", err)
		}
	}

	return nil
}
