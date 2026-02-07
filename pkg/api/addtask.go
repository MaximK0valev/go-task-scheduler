package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MaximK0valev/go-task-scheduler/pkg/db"
)

// addTaskHandler creates a new task.
//
// Method: POST /api/task
// Body:   JSON (db.Task)
// Result: {"id": "..."}
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, map[string]string{"error": "Метод не поддерживается"})
		return
	}
	var task db.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Ошибка десериализации JSON: " + err.Error()})
		return
	}

	if task.Title == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан заголовок задачи"})
		return
	}

	// Normalize the date: set default date, prevent dates in the past,
	// and for repeating tasks calculate the next occurrence.
	if err := checkDate(&task); err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	id, err := db.AddTask(&task)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка сохранения задачи: " + err.Error()})
		return
	}

	writeJson(w, http.StatusOK, map[string]string{"id": strconv.FormatInt(id, 10)})
}

// checkDate validates and normalizes task.Date.
//
// Rules:
//   - If date is empty, it defaults to "today".
//   - Non-repeating tasks cannot be scheduled in the past: past dates are replaced with "today".
//   - Repeating tasks scheduled for today or earlier are moved to the next occurrence.
func checkDate(task *db.Task) error {
	now := time.Now()

	if task.Date == "" {
		task.Date = now.Format(DateFormat)
	}

	t, err := time.Parse(DateFormat, task.Date)
	if err != nil {
		return fmt.Errorf("некорректная дата: %v", err)
	}

	// startOfDay returns midnight for the given date.
	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	}

	if task.Repeat != "" {
		next, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return fmt.Errorf("некорректное правило повторения: %v", err)
		}

		// If the initial date is not after today, move it forward.
		if !t.After(startOfDay(now)) {
			task.Date = next
		}
	} else {
		// Non-repeating task: do not allow dates strictly before today.
		if t.Before(startOfDay(now)) {
			task.Date = now.Format(DateFormat)
		}
	}

	return nil
}

// getTaskHandler returns a single task by ID.
//
// Method: GET /api/task?id=<id>
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}
	task, err := db.GetTask(id)
	if err != nil {
		writeJson(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		return
	}
	writeJson(w, http.StatusOK, task)
}

// updateTaskHandler updates an existing task.
//
// Method: PUT /api/task
// Body:   JSON (db.Task with non-empty ID)
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var t db.Task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "некорректный JSON"})
		return
	}
	if t.ID == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}
	if t.Title == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан заголовок задачи"})
		return
	}

	// Validate repeat rule format.
	err = checkRepeat(t.Repeat)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Normalize/validate date for the updated task.
	err = checkDate(&t)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	err = db.UpdateTask(&t)
	if err != nil {
		if err.Error() == "задача не найдена" {
			writeJson(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		} else {
			writeJson(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка обновления задачи: " + err.Error()})
		}
		return
	}
	writeJson(w, http.StatusOK, struct{}{})
}

// deleteTaskHandler deletes a task by ID.
//
// Method: DELETE /api/task?id=<id>
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}
	err := db.DeleteTask(id)
	if err != nil {
		if err.Error() == "задача не найдена" {
			writeJson(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		} else {
			writeJson(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка удаления задачи: " + err.Error()})
		}
		return
	}
	writeJson(w, http.StatusOK, struct{}{})
}

// taskDone marks a task as completed.
//
// Behavior:
//   - For non-repeating tasks: delete from DB.
//   - For repeating tasks: compute next date and update the task.
//
// Method: POST /api/task/done?id=<id>
func taskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, map[string]string{"error": "Метод не поддерживается"})
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}
	task, err := db.GetTask(id)
	if err != nil {
		writeJson(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		return
	}
	if task.Repeat == "" {
		err = db.DeleteTask(id)
		if err != nil {
			writeJson(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка удаления: " + err.Error()})
			return
		}
		writeJson(w, http.StatusOK, struct{}{})
		return
	}

	nextdata, err := NextDate(time.Now(), task.Date, task.Repeat)
	if err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "Не удалось раcчитать следующую дату: " + err.Error()})
		return
	}
	err = db.UpdateDate(nextdata, id)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": "Не удалось обновить дату: " + err.Error()})
		return
	}
	writeJson(w, http.StatusOK, struct{}{})
}

// checkRepeat validates repeat rule format.
//
// Supported formats:
//   - d <N>           every N days (1..400)
//   - w <list>        weekly on weekdays (1..7), e.g. "w 1,3,5"
//   - m <days> [mons] monthly on day numbers, e.g. "m 1,15" or "m -1" (last day)
//     optional months list: "m 1,15 1,6,12"
//   - y               yearly
func checkRepeat(repeat string) error {
	if repeat == "" {
		return nil
	}

	parts := strings.Fields(repeat)
	if len(parts) < 1 {
		return errors.New("некорректный repeat")
	}

	unit := parts[0]

	switch unit {
	case "d":
		if len(parts) != 2 {
			return errors.New("некорректный формат для d")
		}
		num, err := strconv.Atoi(parts[1])
		if err != nil || num <= 0 {
			return errors.New("некорректное число repeat")
		}

	case "w":
		if len(parts) < 2 {
			return errors.New("отсутствуют дни недели для w")
		}
		dayStrs := strings.Split(parts[1], ",")
		for _, dayStr := range dayStrs {
			dayNum, err := strconv.Atoi(dayStr)
			if err != nil || dayNum < 1 || dayNum > 7 {
				return errors.New("некорректный день недели")
			}
		}

	case "m":
		if len(parts) < 2 {
			return errors.New("отсутствуют дни месяца для m")
		}
		dayStrs := strings.Split(parts[1], ",")
		for _, dayStr := range dayStrs {
			dayNum, err := strconv.Atoi(dayStr)
			if err != nil || (dayNum < -2 || dayNum > 31 || dayNum == 0) {
				return errors.New("некорректный день месяца")
			}
		}

	case "y":
		if len(parts) != 1 {
			return errors.New("некорректный формат для d/y")
		}
	default:
		return errors.New("некорректная единица repeat")
	}

	return nil
}
