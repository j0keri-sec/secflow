// Package logger initialises a production-grade zap logger for the client.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DBWriter is the interface for writing logs to SQLite.
type DBWriter interface {
	ZapCore(enab zapcore.LevelEnabler) zapcore.Core
}

// New creates a zap.Logger that writes JSON to logPath (and always to stderr).
// If logPath is empty, only stderr output is configured.
// If dbWriter is provided, logs will also be written to SQLite.
func New(logPath string, debug bool, dbWriter DBWriter) (*zap.Logger, error) {
	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var cores []zapcore.Core

	// Always log to stderr.
	cores = append(cores, zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(os.Stderr),
		level,
	))

	// Optionally append a file sink.
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(f),
			level,
		))
	}

	// Optionally append SQLite sink.
	if dbWriter != nil {
		cores = append(cores, dbWriter.ZapCore(level))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller()), nil
}
