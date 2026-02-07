package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MaximK0valev/go-task-scheduler/pkg/api"
	"github.com/MaximK0valev/go-task-scheduler/pkg/db"
	"github.com/MaximK0valev/go-task-scheduler/pkg/server"

	"github.com/joho/godotenv"
)

// Application entry point.
//
// Loads configuration from environment (.env is optional), initializes the database
// and starts the HTTP server.
func main() {
	// Load environment variables from .env if present.
	// If the file is missing, the application falls back to system environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем системные переменные")
	}

	config := api.GetConfig()

	// Debug-print effective configuration (useful during local development).
	// Note: printing secrets (password/token) is not recommended for production.
	fmt.Printf("Конфигурация приложения:\n")
	fmt.Printf("TODO_PASSWORD = %s\n", config.TodoPassword)
	fmt.Printf("TODO_PORT = %s\n", config.TodoPort)
	fmt.Printf("TODO_DBFILE = %s\n", config.TodoDBFile)

	// Initialize SQLite database and install schema on first run.
	if err := db.Init(config.TodoDBFile); err != nil {
		fmt.Printf("Ошибка инициализации базы данных: %v\n", err)
		os.Exit(1)
	}

	// Validate DB connection early.
	if err := db.DB.Ping(); err != nil {
		fmt.Printf("Нет соединения с БД: %v\n", err)
		os.Exit(1)
	}
	defer db.DB.Close()

	fmt.Println("База данных подключена успешно")
	server.Run()
}
