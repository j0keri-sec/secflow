// Package db provides a SQLite-backed local store for the secflow client.
// It caches task states, crawl results and local logs so the client can
// resume work after a restart and avoid duplicate uploads.
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps *sql.DB and exposes domain-specific helper methods.
type DB struct {
	conn *sql.DB
}

// Open initialises the SQLite database at the given path and runs
// all schema migrations.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	conn.SetMaxOpenConns(1) // SQLite is single-writer
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

// Close releases the underlying database connection.
func (d *DB) Close() error { return d.conn.Close() }

// ── Schema ────────────────────────────────────────────────────────────────

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
	CREATE TABLE IF NOT EXISTS tasks (
		id          TEXT PRIMARY KEY,
		task_id     TEXT NOT NULL UNIQUE,
		type        TEXT NOT NULL,
		status      TEXT NOT NULL DEFAULT 'pending',
		payload     TEXT,                     -- JSON
		progress    INTEGER NOT NULL DEFAULT 0,
		error       TEXT,
		received_at TEXT NOT NULL,
		updated_at  TEXT NOT NULL,
		finished_at TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

	CREATE TABLE IF NOT EXISTS vuln_cache (
		key         TEXT PRIMARY KEY,
		title       TEXT NOT NULL,
		severity    TEXT NOT NULL,
		cve         TEXT,
		source      TEXT,
		task_id     TEXT NOT NULL,
		uploaded    INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS logs (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		level      TEXT NOT NULL,
		message    TEXT NOT NULL,
		fields     TEXT,                      -- JSON key-value
		created_at TEXT NOT NULL
	);
	`)
	return err
}

// ── Task helpers ──────────────────────────────────────────────────────────

// TaskRecord mirrors the tasks table.
type TaskRecord struct {
	ID         string
	TaskID     string
	Type       string
	Status     string
	Payload    string // raw JSON
	Progress   int
	Error      string
	ReceivedAt time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
}

// UpsertTask creates or updates a task record.
func (d *DB) UpsertTask(t *TaskRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.Exec(`
	INSERT INTO tasks (id, task_id, type, status, payload, progress, error, received_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(task_id) DO UPDATE SET
		status = excluded.status,
		progress = excluded.progress,
		error = excluded.error,
		updated_at = excluded.updated_at,
		finished_at = excluded.finished_at
	`, t.ID, t.TaskID, t.Type, t.Status, t.Payload, t.Progress, t.Error, now, now)
	return err
}

// UpdateTaskStatus updates task status and optionally marks it finished.
func (d *DB) UpdateTaskStatus(taskID, status string, progress int, errMsg string) error {
	var finishedAt *string
	if status == "done" || status == "failed" {
		s := time.Now().UTC().Format(time.RFC3339)
		finishedAt = &s
	}
	_, err := d.conn.Exec(`
	UPDATE tasks SET status=?, progress=?, error=?, updated_at=?, finished_at=? WHERE task_id=?`,
		status, progress, errMsg, time.Now().UTC().Format(time.RFC3339), finishedAt, taskID)
	return err
}

// GetPendingTasks returns all non-finished tasks, ordered by received_at.
func (d *DB) GetPendingTasks() ([]*TaskRecord, error) {
	rows, err := d.conn.Query(`
	SELECT id, task_id, type, status, payload, progress, error, received_at, updated_at
	FROM tasks WHERE status NOT IN ('done','failed') ORDER BY received_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		t := &TaskRecord{}
		var recv, upd string
		if err := rows.Scan(&t.ID, &t.TaskID, &t.Type, &t.Status,
			&t.Payload, &t.Progress, &t.Error, &recv, &upd); err != nil {
			return nil, err
		}
		t.ReceivedAt, _ = time.Parse(time.RFC3339, recv)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, upd)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// ── Vuln cache helpers ────────────────────────────────────────────────────

// MarkVulnUploaded sets the uploaded flag for a cached vuln entry.
func (d *DB) MarkVulnUploaded(key string) error {
	_, err := d.conn.Exec(`UPDATE vuln_cache SET uploaded=1 WHERE key=?`, key)
	return err
}

// InsertVulnCache stores a vuln result locally.
func (d *DB) InsertVulnCache(key, title, severity, cve, source, taskID string) error {
	_, err := d.conn.Exec(`
	INSERT OR IGNORE INTO vuln_cache (key, title, severity, cve, source, task_id, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)`,
		key, title, severity, cve, source, taskID, time.Now().UTC().Format(time.RFC3339))
	return err
}

// CountPendingUploads returns how many vuln records have not yet been uploaded.
func (d *DB) CountPendingUploads() (int, error) {
	var n int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM vuln_cache WHERE uploaded=0`).Scan(&n)
	return n, err
}

// ── Log helpers ───────────────────────────────────────────────────────────

// LogRecord mirrors the logs table.
type LogRecord struct {
	ID        int64
	Level     string
	Message   string
	Fields    string // JSON
	CreatedAt time.Time
}

// InsertLog writes a log entry to the local SQLite database.
func (d *DB) InsertLog(level, message, fields string) error {
	_, err := d.conn.Exec(`
		INSERT INTO logs (level, message, fields, created_at)
		VALUES (?, ?, ?, ?)`,
		level, message, fields, time.Now().UTC().Format(time.RFC3339))
	return err
}

// GetRecentLogs returns the most recent log entries, limited by count.
func (d *DB) GetRecentLogs(limit int) ([]*LogRecord, error) {
	rows, err := d.conn.Query(`
		SELECT id, level, message, fields, created_at
		FROM logs
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*LogRecord
	for rows.Next() {
		l := &LogRecord{}
		var created string
		if err := rows.Scan(&l.ID, &l.Level, &l.Message, &l.Fields, &created); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, created)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// CleanOldLogs removes log entries older than the specified duration.
func (d *DB) CleanOldLogs(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	result, err := d.conn.Exec(`DELETE FROM logs WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
