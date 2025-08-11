package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"todo-app/internal/logger"
	"todo-app/internal/manager"
)

type TemplateData struct {
	Tasks []manager.Task
}

var templateFuncs = template.FuncMap{
	"now": time.Now,
	"daysLeft": func(dueDate time.Time) int {
		return int(time.Until(dueDate).Hours() / 24)
	},
	"getPopularTags": func(tasks []manager.Task) []string {
		tagCounts := make(map[string]int)
		for _, task := range tasks {
			for _, tag := range task.Tags {
				tagCounts[tag]++
			}
		}
		var popular []string
		for tag := range tagCounts {
			popular = append(popular, tag)
		}
		sort.Slice(popular, func(i, j int) bool {
			return tagCounts[popular[i]] > tagCounts[popular[j]]
		})
		if len(popular) > 5 {
			return popular[:5]
		}
		return popular
	},
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
  GET    /tasks/filter/{status} - Filter tasks (all/completed/active)
  GET    /tasks/priority/{priority} - Filter by priority (low/medium/high)
  GET    /tasks/tag/{tag} - Filter by tag
  GET    /tasks/upcoming/{days} - Upcoming tasks (within days)
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
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		data := TemplateData{
			Tasks: tm.GetAllTasks(),
		}
		tmpl.Execute(w, data)
	})

	r.Get("/tasks/filter/{status}", func(w http.ResponseWriter, r *http.Request) {
		status := chi.URLParam(r, "status")
		var completed *bool
		switch status {
		case "completed":
			val := true
			completed = &val
		case "active":
			val := false
			completed = &val
		case "all":
			completed = nil
		default:
			http.Error(w, "Недопустимый статус фильтра", http.StatusBadRequest)
			return
		}
		tasks := tm.FilterTasks(completed)
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: tasks})
	})

	r.Get("/tasks/priority/{priority}", func(w http.ResponseWriter, r *http.Request) {
		priority := manager.Priority(chi.URLParam(r, "priority"))
		if priority != manager.PriorityLow && 
		   priority != manager.PriorityMedium && 
		   priority != manager.PriorityHigh {
			http.Error(w, "Недопустимый приоритет", http.StatusBadRequest)
			return
		}
		tasks := tm.FilterByPriority(priority)
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: tasks})
	})

	r.Get("/tasks/tag/{tag}", func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "tag")
		tasks := tm.FilterByTag(tag)
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: tasks})
	})

	r.Get("/tasks/upcoming/{days}", func(w http.ResponseWriter, r *http.Request) {
		daysStr := chi.URLParam(r, "days")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			http.Error(w, "Недопустимое количество дней", http.StatusBadRequest)
			return
		}
		tasks := tm.GetUpcomingTasks(days)
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: tasks})
	})

	r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		description := r.FormValue("description")
		priority := manager.Priority(r.FormValue("priority"))
		dueDateStr := r.FormValue("due_date")
		tagsStr := r.FormValue("tags")
		
		if description == "" {
			manager.AddTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Описание задачи обязательно", http.StatusBadRequest)
			return
		}

		if priority != manager.PriorityLow && 
		   priority != manager.PriorityMedium && 
		   priority != manager.PriorityHigh {
			priority = manager.PriorityMedium
		}

		var dueDate time.Time
		if dueDateStr != "" {
			var err error
			dueDate, err = time.Parse("2006-01-02", dueDateStr)
			if err != nil {
				http.Error(w, "Некорректная дата выполнения", http.StatusBadRequest)
				return
			}
		}

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		taskID, err := tm.AddTask(description, tags)
		if err != nil {
			manager.AddTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = tm.UpdateTask(taskID, manager.UpdateTaskRequest{
			Priority: &priority,
			DueDate:  &dueDate,
		})
		if err != nil {
			manager.AddTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		manager.AddTaskCount.WithLabelValues("success").Inc()
		manager.AddTaskDuration.Observe(time.Since(startTime).Seconds())
		manager.TaskDescLength.Observe(float64(len(description)))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

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
		priority := manager.Priority(r.FormValue("priority"))
		dueDateStr := r.FormValue("due_date")
		tagsStr := r.FormValue("tags")
		var dueDate time.Time
		if dueDateStr != "" {
			dueDate, err = time.Parse("2006-01-02", dueDateStr)
			if err != nil {
				http.Error(w, "Некорректная дата выполнения", http.StatusBadRequest)
				return
			}
		}
		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}
		_, err = tm.UpdateTask(id, manager.UpdateTaskRequest{
			Description: &description,
			Priority:    &priority,
			DueDate:     &dueDate,
			Tags:        &tags,
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
