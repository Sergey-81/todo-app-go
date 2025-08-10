package main

import (
	"context"
	"html/template"
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

type TemplateData struct {
	Tasks []manager.Task
}

func printWelcomeMessage() {
	println(`
🚀 Todo-App Server
-----------------------------
Available endpoints:
  POST   /tasks          - Add new task
  POST   /tasks/toggle/{id} - Toggle task completion
  POST   /tasks/update/{id} - Update task
  POST   /tasks/delete/{id} - Delete task
  GET    /               - Web Interface (:8080)
  GET    /metrics        - Prometheus metrics
-----------------------------
Storage type: In-Memory
Start time: ` + time.Now().Format("2006-01-02 15:04:05") + `
-----------------------------
`)
}

func main() {
	ctx := context.Background()
	logger.SetLevel(logger.LevelInfo)
	printWelcomeMessage()
	logger.Info(ctx, "Starting todo-app server...")

	tm := manager.NewTaskManager()
	
	// Тестовые задачи
	tm.AddTask("Первая задача")
	tm.AddTask("Вторая задача")

	r := chi.NewRouter()
	setupRoutes(r, tm)

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info(ctx, "Server started on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, err, "Server error")
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	logger.Info(ctx, "Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error(ctx, err, "Server shutdown error")
	}
	logger.Info(ctx, "Server stopped")
}

func setupRoutes(r *chi.Mux, tm *manager.TaskManager) {
	// Метрики Prometheus
	r.Handle("/metrics", promhttp.Handler())

	// Главная страница
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("static/index.html"))
		data := TemplateData{
			Tasks: tm.GetAllTasks(),
		}
		tmpl.Execute(w, data)
	})

	// Добавление задачи
	r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		description := r.FormValue("description")
		
		if description == "" {
			manager.AddTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Описание задачи обязательно", http.StatusBadRequest)
			return
		}

		_, err := tm.AddTask(description)
		if err != nil {
			manager.AddTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		manager.AddTaskCount.WithLabelValues("success").Inc()
		manager.AddTaskDuration.Observe(time.Since(startTime).Seconds())
		manager.TaskDescLength.Observe(float64(len(description)))
		
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Переключение статуса задачи
	r.Post("/tasks/toggle/{id}", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		_, err = tm.ToggleComplete(id)
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		manager.UpdateTaskCount.WithLabelValues("success").Inc()
		manager.UpdateTaskDuration.Observe(time.Since(startTime).Seconds())
		
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Обновление задачи
	r.Post("/tasks/update/{id}", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		description := r.FormValue("description")
		if description == "" {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Описание задачи обязательно", http.StatusBadRequest)
			return
		}

		_, err = tm.UpdateTask(id, manager.UpdateTaskRequest{
			Description: &description,
		})
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		manager.UpdateTaskCount.WithLabelValues("success").Inc()
		manager.UpdateTaskDuration.Observe(time.Since(startTime).Seconds())
		
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Удаление задачи
	r.Post("/tasks/delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		
		if err != nil {
			manager.DeleteTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		if err := tm.DeleteTask(id); err != nil {
			manager.DeleteTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		manager.DeleteTaskCount.WithLabelValues("success").Inc()
		manager.DeleteTaskDuration.Observe(time.Since(startTime).Seconds())
		
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}
