package api

import (
	"net/http"

	"github.com/MaximK0valev/go-task-scheduler/pkg/db"
)

// TasksResp is a response wrapper for GET /api/tasks.
type TasksResp struct {
	Tasks []*db.Task `json:"tasks"`
}

// tasksHandler returns a list of tasks.
//
// Method: GET /api/tasks
// Query:
//   - search (optional): if set, tasks are filtered by substring or by date.
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJson(w, http.StatusMethodNotAllowed, map[string]string{"error": "Метод не поддерживается"})
		return
	}

	limit := 50
	search := r.URL.Query().Get("search")
	var tasks []*db.Task
	var err error

	if search == "" {
		tasks, err = db.Tasks(limit)
	} else {
		tasks, err = db.SearchTasks(search, limit)
	}

	if err != nil {
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJson(w, http.StatusOK, TasksResp{
		Tasks: tasks,
	})
}
