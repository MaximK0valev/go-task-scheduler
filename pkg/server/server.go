package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaximK0valev/go-task-scheduler/pkg/api"
)

// Run starts the HTTP server, registers API routes and serves static web files.
//
// The server supports graceful shutdown on SIGINT/SIGTERM.
func Run() {
	config := api.GetConfig()
	port := config.TodoPort

	// Register HTTP handlers under /api/*.
	api.Init()

	// Serve static UI from ./web (login page, index, assets).
	webDir := "./web"
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	srv := &http.Server{
		Addr: ":" + port,
	}

	log.Printf("Сервер запускается на порту %s", port)
	log.Printf("Статические файлы обслуживаются из директории: %s", webDir)

	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		log.Printf("ВНИМАНИЕ: Директория '%s' не найдена", webDir)
	}

	go func() {
		log.Printf("Сервер запущен на http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Ошибка сервера: %v", err)
		}
	}()

	// Wait for termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Завершение работы сервера...")

	// Gracefully shut down with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Принудительное завершение сервера: %v", err)
	}

	log.Println("Сервер остановлен")
}
