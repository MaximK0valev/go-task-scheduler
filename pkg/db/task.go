package db

import (
	"database/sql"
	"errors"
	"fmt"
)

// Task represents a single scheduled task.
//
// Date uses DateFormat (YYYYMMDD).
// Repeat stores a repeat rule string (see API documentation).
type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// AddTask inserts a new task and returns its auto-generated database ID.
func AddTask(task *Task) (int64, error) {
	var id int64
	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat)
	if err == nil {
		id, err = res.LastInsertId()
	}
	return id, err
}

// Tasks returns latest tasks ordered by date (ascending) limited by `limit`.
func Tasks(limit int) ([]*Task, error) {
	rows, err := DB.Query("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?", limit)
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

// GetTask returns a single task by id.
// If the record does not exist, a "task not found" error is returned.
func GetTask(id string) (*Task, error) {
	task := &Task{}
	err := DB.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).
		Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, err
	}

	return task, nil
}

// UpdateTask updates an existing task by id.
// If no rows are affected, the task is considered missing.
func UpdateTask(task *Task) error {
	res, err := DB.Exec(
		"UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}

	return nil
}

// DeleteTask removes a task by id.
// If no rows are affected, the task is considered missing.
func DeleteTask(id string) error {
	res, err := DB.Exec("DELETE FROM scheduler WHERE id=?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}

// UpdateDate updates only the date field for a task.
// Used when marking repeating tasks as done.
func UpdateDate(next string, id string) error {
	res, err := DB.Exec("UPDATE scheduler SET date=? WHERE id=?", next, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача не найдена")
	}

	return nil
}
