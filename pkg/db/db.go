package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// DateFormat matches the canonical date format used in DB and API.
// It corresponds to YYYYMMDD.
const DateFormat = "20060102"

// DB is the shared database connection used by data access functions.
var DB *sql.DB

// schema is installed on first run (when the database file does not exist).
const schema = `
CREATE TABLE scheduler (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date CHAR(8) NOT NULL DEFAULT '',
    title VARCHAR(256) NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    repeat VARCHAR(128) NOT NULL DEFAULT ''
);
CREATE INDEX idx_scheduler_date ON scheduler(date);
`

// Init opens SQLite database and installs schema on first run.
func Init(dbFile string) error {
	_, err := os.Stat(dbFile)
	install := os.IsNotExist(err)
	DB, err = sql.Open("sqlite", dbFile)
	if err != nil {
		return fmt.Errorf("ошибка при открытии базы данных: %w", err)
	}

	if install {
		_, err := DB.Exec(schema)
		if err != nil {
			return fmt.Errorf("ошибка при открытии базы данных: %w", err)
		}
	}
	return nil
}

// SearchTasks searches tasks by either:
//   - a date in DD.MM.YYYY format, or
//   - a substring match in title/comment.
//
// The `limit` parameter controls maximum number of returned items.
func SearchTasks(search string, limit int) ([]*Task, error) {

	date, err := time.Parse("02.01.2006", search)
	if err == nil {
		formatted := date.Format(DateFormat)
		return TasksByDate(formatted, limit)
	}

	pattern := "%" + search + "%"
	return TasksByPattern(pattern, limit)
}

// TasksByDate returns tasks scheduled on a specific date (YYYYMMDD).
func TasksByDate(formatted string, limit int) ([]*Task, error) {
	rows, err := DB.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT ?", formatted, limit)
	if err != nil {
		return []*Task{}, err
	}

	defer rows.Close()
	tasks := []*Task{}

	for rows.Next() {
		task := &Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return []*Task{}, err
	}
	if tasks == nil {
		tasks = []*Task{}
	}

	return tasks, nil
}

// TasksByPattern returns tasks where title or comment matches the given SQL LIKE pattern.
func TasksByPattern(pattern string, limit int) ([]*Task, error) {
	rows, err := DB.Query("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ? ", pattern, pattern, limit)
	if err != nil {
		return []*Task{}, err
	}

	defer rows.Close()
	tasks := []*Task{}

	for rows.Next() {
		task := &Task{}
		err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return []*Task{}, err
	}
	if tasks == nil {
		tasks = []*Task{}
	}

	return tasks, nil
}
