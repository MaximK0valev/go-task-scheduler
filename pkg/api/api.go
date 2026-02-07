package api

import (
	"encoding/json"
	"net/http"
)

// Init registers all HTTP routes for the application.
//
// Public endpoints:
//   - POST /api/signin
//   - GET  /api/nextdate
//
// Protected endpoints (require AuthMiddleware):
//   - /api/task (CRUD)
//   - GET /api/tasks
//   - POST /api/task/done
func Init() {
	http.HandleFunc("/api/signin", SigninHandler)
	http.HandleFunc("/api/nextdate", nextDayHandler)
	http.HandleFunc("/api/task", AuthMiddleware(taskHandler))
	http.HandleFunc("/api/tasks", AuthMiddleware(tasksHandler))
	http.HandleFunc("/api/task/done", AuthMiddleware(taskDoneHandler))
}

// taskHandler is a multiplexer for CRUD operations on a single task.
func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		addTaskHandler(w, r)
	case http.MethodGet:
		getTaskHandler(w, r)
	case http.MethodPut:
		updateTaskHandler(w, r)
	case http.MethodDelete:
		deleteTaskHandler(w, r)
	}

}

// taskDoneHandler marks a task as done.
// For repeating tasks, it moves the date to the next occurrence.
func taskDoneHandler(w http.ResponseWriter, r *http.Request) {
	taskDone(w, r)
}

// writeJson is a small helper to marshal JSON responses.
func writeJson(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
