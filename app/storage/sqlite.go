package storage

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
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

		log.Printf("DB_PATH not set, using default: %s", defaultPath)
		return defaultPath
	}
	return dbPath
}

func NewSQLiteStorage() *SQLiteStorage {
	dbPath := getDBPath()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("❌ Error opening SQLite DB at %s: %v", dbPath, err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS task_history (
            task_id TEXT NOT NULL,
            iteration INTEGER NOT NULL,
            action TEXT NOT NULL,
            filename TEXT,
            response TEXT NOT NULL,
            PRIMARY KEY (task_id, iteration)
        );
    `)
	if err != nil {
		log.Fatalf("❌ Error creating table: %v", err)
	}

	return &SQLiteStorage{db: db}
}

func (ts *SQLiteStorage) SaveRecord(record Record) error {
	_, err := ts.db.Exec(`INSERT INTO task_history (task_id, iteration, action, filename, response) 
		VALUES (?, ?, ?, ?, ?)`,

		record.TaskID, record.Iteration, record.Action, record.Filename, record.Response,
	)

	if err != nil {
		log.Printf("⚠️ Error saving iteration %d for task %s: %v", record.Iteration, record.TaskID, err)
		return err
	}

	log.Printf("✅ Successfully saved iteration %d for task %s", record.Iteration, record.TaskID)
	return nil
}

func (ts *SQLiteStorage) GetRecords(taskID string) ([]Record, error) {
	rows, err := ts.db.Query(`SELECT iteration, action, filename, response 
		FROM task_history WHERE task_id = ? ORDER BY iteration ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []Record
	for rows.Next() {
		var record Record

		if err = rows.Scan(&record.Iteration, &record.Action, &record.Filename, &record.Response); err != nil {
			log.Printf("⚠️ Error scanning row: %v", err)
			continue
		}

		record.TaskID = taskID
		history = append(history, record)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}
