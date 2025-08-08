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

func main() {
	// Инициализация логгера
	logger.SetLevel(logger.LevelDebug)
	ctx := context.Background()
	logger.Info(ctx, "Starting todo-app server...")

	tm := manager.NewTaskManager()
	r := chi.NewRouter()

	// Маршруты API
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

	// Метрики Prometheus
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":2112",
		Handler: r,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info(ctx, "Server started on :2112")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, err, "Server error")
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info(ctx, "Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(ctx, err, "Server shutdown error")
	}
	logger.Info(ctx, "Server stopped")
}
