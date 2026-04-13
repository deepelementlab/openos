// Package logging provides unified zap configuration for all AOS components.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ProductionConfig returns JSON logging suitable for centralized ingestion (ELK/Loki).
func ProductionConfig() zap.Config {
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

func encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// NewProduction builds a production logger with optional component field.
func NewProduction(component string) (*zap.Logger, error) {
	cfg := ProductionConfig()
	log, err := cfg.Build(zap.AddCaller())
	if err != nil {
		return nil, err
	}
	if component != "" {
		log = log.With(zap.String("component", component))
	}
	return log, nil
}

// NewDevelopment builds a console logger for local work.
func NewDevelopment(component string) (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	log, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	if component != "" {
		log = log.With(zap.String("component", component))
	}
	return log, nil
}

// FromEnv picks production vs development based on AOS_LOG_MODE.
func FromEnv(component string) (*zap.Logger, error) {
	if os.Getenv("AOS_LOG_MODE") == "dev" {
		return NewDevelopment(component)
	}
	return NewProduction(component)
}
