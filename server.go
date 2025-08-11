package server

import (
	"encoding/json"
	"net/http"
	"todo-app/internal/manager"
	"todo-app/internal/models"

	"github.com/go-chi/chi/v5"
)

func NewRouter(tm *manager.TaskManager) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/tasks", addTaskHandler(tm))
	return r
}

func addTaskHandler(tm *manager.TaskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateTaskRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Добавляем пустой слайс тегов, если они не переданы
		tags := []string{}
		if req.Tags != nil {
			tags = req.Tags
		}

		if _, err := tm.AddTask(req.Description, tags); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}