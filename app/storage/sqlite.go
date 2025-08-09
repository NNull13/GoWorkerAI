package storage

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var _ Interface = &SQLiteContextStorage{}

type SQLiteContextStorage struct {
	db *sql.DB
}

func getDBPath() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		projectDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("❌ Error getting project directory: %v", err)
		}
		defaultPath := filepath.Join(projectDir, "data", "database.db")
		if err := os.MkdirAll(filepath.Dir(defaultPath), os.ModePerm); err != nil {
			log.Fatalf("❌ Error creating data directory: %v", err)
		}
		log.Printf("📂 DB_PATH not set, using default: %s", defaultPath)
		return defaultPath
	}
	return dbPath
}

func NewSQLiteStorage() *SQLiteContextStorage {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("❌ Error opening SQLite DB at %s: %v", dbPath, err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS records (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_id TEXT NOT NULL,
            step_id INTEGER,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            tool TEXT NULL,
            parameters TEXT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        CREATE INDEX IF NOT EXISTS idx_task_id ON records (task_id);
    `)
	if err != nil {
		log.Fatalf("❌ Error creating table: %v", err)
	}

	return &SQLiteContextStorage{db: db}
}

func (s *SQLiteContextStorage) SaveHistory(ctx context.Context, record Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO records (task_id, step_id, role, content, tool, parameters, created_at)
                 VALUES (?, ?, ?, ?, ?, ?, datetime(?))`,
		record.TaskID, record.StepID, record.Role, record.Content, record.Tool, record.Parameters, record.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		log.Printf("⚠️ Error saving record for task %s: %v", record.TaskID, err)
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	record.ID = id

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("✅ Record saved: %+v", record)
	return nil
}

func (s *SQLiteContextStorage) GetHistoryByTaskID(ctx context.Context, taskID string, stepID ...int) ([]Record, error) {
	query := `
         SELECT id, task_id, step_id, role, content, tool, parameters, created_at
         FROM records
         WHERE task_id = ?`
	args := []any{taskID}
	if len(stepID) > 0 {
		query += " AND step_id = ?"
		args = append(args, stepID[0])
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []Record
	for rows.Next() {
		var it Record
		var createdAt string
		if err = rows.Scan(&it.ID, &it.TaskID, &it.StepID, &it.Role, &it.Content, &it.Tool, &it.Parameters, &createdAt); err != nil {
			log.Printf("⚠️ Error scanning row for task %s: %v", taskID, err)
			continue
		}
		it.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		history = append(history, it)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}
