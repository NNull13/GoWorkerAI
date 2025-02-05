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
        CREATE TABLE IF NOT EXISTS iterations (
            id INTEGER NOT NULL,
            task_id TEXT NOT NULL,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            tool TEXT NULL,
            parameters TEXT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (task_id, id) 
        );
        CREATE INDEX IF NOT EXISTS idx_task_id ON iterations (task_id); 
    `)
	if err != nil {
		log.Fatalf("❌ Error creating table: %v", err)
	}

	return &SQLiteContextStorage{db: db}
}

func (s *SQLiteContextStorage) SaveIteration(ctx context.Context, iteration Iteration) error {
	var lastID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(id), 0) FROM iterations WHERE task_id = ?`, iteration.TaskID,
	).Scan(&lastID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("⚠️ Error retrieving last ID for task %s: %v", iteration.TaskID, err)
		return err
	}

	iteration.ID = lastID + 1

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO iterations (id, task_id, role, content, tool, parameters, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime(?))`,
		iteration.ID, iteration.TaskID, iteration.Role, iteration.Content, iteration.Tool, iteration.Parameters, iteration.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		log.Printf("⚠️ Error saving iteration for task %s: %v", iteration.TaskID, err)
		return err
	}
	log.Printf("✅ Iteration saved: %+v", iteration)
	return nil
}

func (s *SQLiteContextStorage) GetHistoryByTaskID(ctx context.Context, taskID string) ([]Iteration, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, role, content, tool, parameters, created_at
		 FROM iterations
		 WHERE task_id = ?
		 ORDER BY id ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []Iteration
	for rows.Next() {
		var it Iteration
		var createdAt string
		if err = rows.Scan(&it.ID, &it.TaskID, &it.Role, &it.Content, &it.Tool, &it.Parameters, &createdAt); err != nil {
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
