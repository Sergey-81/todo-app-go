package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"todo-app/internal/logger"
	"todo-app/internal/manager"
)

func printWelcomeMessage() {
	fmt.Println("")
	fmt.Println("🚀 Todo-App Server")
	fmt.Println("-----------------------------")
	fmt.Println("Available endpoints:")
	fmt.Println("  POST   /tasks      - Add new task")
	fmt.Println("  PATCH  /tasks/{id} - Update task")
	fmt.Println("  GET    /tasks      - Get tasks list (HTMX)") // NEW
	fmt.Println("  GET    /metrics    - Prometheus metrics (:2112)")
	fmt.Println("  GET    /           - HTMX Interface (:8080)")
	fmt.Println("")
	fmt.Println("Storage type: In-Memory")
	fmt.Printf("Start time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("-----------------------------")
	fmt.Println("")
}

func main() {
	ctx := context.Background()
	logger.SetLevel(logger.LevelInfo)
	
	printWelcomeMessage()
	logger.Info(ctx, "Starting todo-app server...")

	tm := manager.NewTaskManager()
	
	// Добавим тестовые задачи для демонстрации // NEW
	tm.AddTask("Первая задача")
	tm.AddTask("Вторая задача")
	
	// Основной роутер (API + метрики)
	apiRouter := chi.NewRouter()
	setupAPIRoutes(apiRouter, tm)

	// Роутер для HTMX-интерфейса
	htmxRouter := chi.NewRouter()
	setupHTMXRoutes(htmxRouter)

	// Сервер для API и метрик (оставляем на :2112)
	apiServer := &http.Server{
		Addr:    ":2112",
		Handler: apiRouter,
	}

	// Сервер для HTMX (новый порт :8080)
	htmxServer := &http.Server{
		Addr:    ":8080",
		Handler: htmxRouter,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера API
	go func() {
		logger.Info(ctx, fmt.Sprintf("API server started on http://localhost%s", apiServer.Addr))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, err, "API server error")
			quit <- syscall.SIGTERM
		}
	}()

	// Запуск сервера HTMX
	go func() {
		logger.Info(ctx, fmt.Sprintf("HTMX server started on http://localhost%s", htmxServer.Addr))
		if err := htmxServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, err, "HTMX server error")
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	logger.Info(ctx, "Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Остановка обоих серверов
	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Error(ctx, err, "API server shutdown error")
	}
	if err := htmxServer.Shutdown(ctx); err != nil {
		logger.Error(ctx, err, "HTMX server shutdown error")
	}
	logger.Info(ctx, "Servers stopped")
}

func setupAPIRoutes(r *chi.Mux, tm *manager.TaskManager) {
	// API endpoints
	r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}

		id, err := tm.AddTask(req.Description)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          id,
			"description": req.Description,
			"created_at":  time.Now(),
		})
	})

	r.Patch("/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, `{"error":"invalid task ID"}`, http.StatusBadRequest)
			return
		}

		var req manager.UpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}

		updatedTask, err := tm.UpdateTask(id, req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedTask)
	})

	// NEW: HTMX endpoint для получения списка задач
	r.Get("/tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks := tm.GetAllTasks()
		w.Header().Set("Content-Type", "text/html")

		for _, task := range tasks {
			completed := ""
			if task.Completed {
				completed = "completed"
			}
			
			fmt.Fprintf(w, `
			<div class="task %s" id="task-%d">
				<span>%s</span>
				<button hx-delete="/tasks/%d" hx-target="#task-%d" hx-swap="outerHTML">
					Удалить
				</button>
			</div>`,
			completed,
			task.ID,
			task.Description,
			task.ID,
			task.ID)
		}
	})

	// Метрики
	r.Handle("/metrics", promhttp.Handler())
}

func setupHTMXRoutes(r *chi.Mux) {
	// Статические файлы (HTMX)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	// Главная страница
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
}
