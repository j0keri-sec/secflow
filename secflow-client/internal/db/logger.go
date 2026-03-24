// Package db provides logging utilities that write to SQLite.
package db

import (
	"encoding/json"

	"go.uber.org/zap/zapcore"
)

// SQLiteLogSink implements zapcore.WriteSyncer to write logs to SQLite.
type SQLiteLogSink struct {
	db *DB
}

// NewSQLiteLogSink creates a new log sink that writes to the provided database.
func NewSQLiteLogSink(db *DB) *SQLiteLogSink {
	return &SQLiteLogSink{db: db}
}

// Write implements zapcore.WriteSyncer.
func (s *SQLiteLogSink) Write(p []byte) (n int, err error) {
	// Parse the JSON log entry
	var entry map[string]interface{}
	if err := json.Unmarshal(p, &entry); err != nil {
		// If we can't parse it, store it as a raw message
		_ = s.db.InsertLog("unknown", string(p), "")
		return len(p), nil
	}

	// Extract level
	level := "info"
	if l, ok := entry["level"].(string); ok {
		level = l
	}

	// Extract message
	message := ""
	if m, ok := entry["msg"].(string); ok {
		message = m
	}

	// Remove standard fields to create the fields JSON
	fields := make(map[string]interface{})
	for k, v := range entry {
		if k != "level" && k != "msg" && k != "ts" {
			fields[k] = v
		}
	}

	var fieldsJSON string
	if len(fields) > 0 {
		b, _ := json.Marshal(fields)
		fieldsJSON = string(b)
	}

	// Insert into database (ignore errors to avoid disrupting logging)
	_ = s.db.InsertLog(level, message, fieldsJSON)

	return len(p), nil
}

// Sync implements zapcore.WriteSyncer.
func (s *SQLiteLogSink) Sync() error {
	return nil
}

// LogEntry represents a structured log entry from the database.
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// ZapCore creates a zapcore.Core that writes to SQLite.
func (d *DB) ZapCore(enab zapcore.LevelEnabler) zapcore.Core {
	sink := NewSQLiteLogSink(d)
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	return zapcore.NewCore(encoder, sink, enab)
}
