package main

import (
	"context"
	"encoding/json"
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
	"todo-app/internal/storage"
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

	if err := os.MkdirAll("data", 0755); err != nil {
		logger.Error(ctx, err, "Ошибка создания директории data")
		return
	}

	dbStorage, err := storage.NewSQLiteStorage("./data/todoapp.db")
	if err != nil {
		logger.Error(ctx, err, "Ошибка инициализации SQLite хранилища")
		return
	}
	defer dbStorage.Close()

	logger.Info(ctx, "SQLite хранилище успешно инициализировано")

	taskManager := manager.NewTaskManagerWithStorage(dbStorage)
	userManager := manager.NewUserManager(dbStorage)
	subTaskManager := manager.NewSubTaskManager()

	r := chi.NewRouter()
	
	// Middleware аутентификации ПЕРВЫМ
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := userManager.GetUserByDeviceID("default_legacy_user")
			if err != nil {
				user, err = userManager.CreateUser("default_legacy_user", 0)
				if err != nil {
					logger.Error(r.Context(), err, "Ошибка создания пользователя")
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
			}
			
			ctx := context.WithValue(r.Context(), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Затем роуты
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		data := TemplateData{Tasks: tasks}
		tmpl.Execute(w, data)
	})

	r.Get("/tasks/filter/{status}", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

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
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		var filteredTasks []manager.Task
		for _, task := range tasks {
			if completed == nil || task.Completed == *completed {
				filteredTasks = append(filteredTasks, task)
			}
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

	r.Get("/tasks/priority/{priority}", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		priority := manager.Priority(chi.URLParam(r, "priority"))
		if priority != manager.PriorityLow && priority != manager.PriorityMedium && priority != manager.PriorityHigh {
			http.Error(w, "Недопустимый приоритет", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		var filteredTasks []manager.Task
		for _, task := range tasks {
			if task.Priority == priority {
				filteredTasks = append(filteredTasks, task)
			}
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

	r.Get("/tasks/tag/{tag}", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		tag := chi.URLParam(r, "tag")
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		var filteredTasks []manager.Task
		for _, task := range tasks {
			for _, taskTag := range task.Tags {
				if strings.EqualFold(taskTag, tag) {
					filteredTasks = append(filteredTasks, task)
					break
				}
			}
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

	r.Get("/tasks/upcoming/{days}", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		daysStr := chi.URLParam(r, "days")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			http.Error(w, "Недопустимое количество дней", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		now := time.Now()
		var filteredTasks []manager.Task
		for _, task := range tasks {
			if !task.DueDate.IsZero() && !task.Completed {
				daysUntilDue := int(task.DueDate.Sub(now).Hours() / 24)
				if daysUntilDue >= 0 && daysUntilDue <= days {
					filteredTasks = append(filteredTasks, task)
				}
			}
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

	r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
    // ДОБАВИТЬ - получить пользователя из контекста
    user, ok := r.Context().Value("user").(*manager.User)
    if !ok {
        http.Error(w, "User not found", http.StatusInternalServerError)
        return
    }

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

    if priority != manager.PriorityLow && priority != manager.PriorityMedium && priority != manager.PriorityHigh {
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

    // ИЗМЕНИТЬ эту строку:
    taskID, err := taskManager.AddTaskForUser(user.ID, description, tags)
    if err != nil {
        manager.AddTaskCount.WithLabelValues("error").Inc()
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    _, err = taskManager.UpdateTask(taskID, manager.UpdateTaskRequest{
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
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		taskFound := false
		for _, task := range tasks {
			if task.ID == id {
				taskFound = true
				break
			}
		}
		if !taskFound {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return
		}
		
		_, err = taskManager.ToggleComplete(id)
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
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			manager.UpdateTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		taskFound := false
		for _, task := range tasks {
			if task.ID == id {
				taskFound = true
				break
			}
		}
		if !taskFound {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
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
		_, err = taskManager.UpdateTask(id, manager.UpdateTaskRequest{
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
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		startTime := time.Now()
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			manager.DeleteTaskCount.WithLabelValues("error").Inc()
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		taskFound := false
		for _, task := range tasks {
			if task.ID == id {
				taskFound = true
				break
			}
		}
		if !taskFound {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return
		}
		
		if err := taskManager.DeleteTask(id); err != nil {
			manager.DeleteTaskCount.WithLabelValues("error").Inc()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		manager.DeleteTaskCount.WithLabelValues("success").Inc()
		manager.DeleteTaskDuration.Observe(time.Since(startTime).Seconds())
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	r.Get("/tasks/filter/date", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		startStr := r.URL.Query().Get("start")
		endStr := r.URL.Query().Get("end")
		
		parseRussianDate := func(dateStr string) (time.Time, error) {
			return time.Parse("02.01.2006", dateStr)
		}
		
		start, err := parseRussianDate(startStr)
		if err != nil {
			http.Error(w, "Неверный формат начальной даты", http.StatusBadRequest)
			return
		}
		
		end, err := parseRussianDate(endStr)
		if err != nil {
			http.Error(w, "Неверный формат конечной даты", http.StatusBadRequest)
			return
		}

		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		var filteredTasks []manager.Task
		for _, task := range tasks {
			if !task.DueDate.IsZero() && !task.DueDate.Before(start) && !task.DueDate.After(end) {
				filteredTasks = append(filteredTasks, task)
			}
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

	// Подзадачи
	r.Get("/tasks/{taskID}/subtasks", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		taskIDStr := chi.URLParam(r, "taskID")
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		taskFound := false
		for _, task := range tasks {
			if task.ID == taskID {
				taskFound = true
				break
			}
		}
		if !taskFound {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return
		}
		
		subtasks := subTaskManager.GetSubTasks(taskID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subtasks)
	})
	
	r.Post("/tasks/{taskID}/subtasks", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		taskIDStr := chi.URLParam(r, "taskID")
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		taskFound := false
		for _, task := range tasks {
			if task.ID == taskID {
				taskFound = true
				break
			}
		}
		if !taskFound {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
			return
		}
		
		description := r.FormValue("description")
		if description == "" {
			http.Error(w, "Описание подзадачи обязательно", http.StatusBadRequest)
			return
		}
		
		id, err := subTaskManager.AddSubTask(taskID, description)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"id": id})
	})
	
	r.Post("/subtasks/{id}/toggle", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Неверный ID подзадачи", http.StatusBadRequest)
			return
		}
		
		if err := subTaskManager.ToggleSubTask(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	})
	
	r.Delete("/subtasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Неверный ID подзадачи", http.StatusBadRequest)
			return
		}
		
		if err := subTaskManager.DeleteSubTask(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/tasks/filter/advanced", func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(*manager.User)
		if !ok {
			http.Error(w, "User not found", http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		options := manager.FilterOptions{}
		
		if completedStr := query.Get("completed"); completedStr != "" {
			completed := completedStr == "true"
			options.Completed = &completed
		}
		
		if priorityStr := query.Get("priority"); priorityStr != "" {
			priority := manager.Priority(priorityStr)
			if priority == manager.PriorityLow || priority == manager.PriorityMedium || priority == manager.PriorityHigh {
				options.Priority = &priority
			}
		}
		
		if tagsStr := query.Get("tags"); tagsStr != "" {
			rawTags := strings.Split(tagsStr, ",")
			options.Tags = make([]string, 0)
			for _, tag := range rawTags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					options.Tags = append(options.Tags, tag)
				}
			}
		}
		
		if startStr := query.Get("start_date"); startStr != "" {
			if start, err := time.Parse("02.01.2006", startStr); err == nil {
				options.StartDate = &start
			}
		}
		
		if endStr := query.Get("end_date"); endStr != "" {
			if end, err := time.Parse("02.01.2006", endStr); err == nil {
				options.EndDate = &end
			}
		}
		
		if hasDueDateStr := query.Get("has_due_date"); hasDueDateStr != "" {
			hasDueDate := hasDueDateStr == "true"
			options.HasDueDate = &hasDueDate
		}
		
		tasks, err := taskManager.GetAllTasksForUser(user.ID)
		if err != nil {
			http.Error(w, "Ошибка загрузки задач", http.StatusInternalServerError)
			return
		}
		var filteredTasks []manager.Task
		
		for _, task := range tasks {
			if options.Completed != nil && task.Completed != *options.Completed {
				continue
			}
			
			if options.Priority != nil && task.Priority != *options.Priority {
				continue
			}
			
			if len(options.Tags) > 0 {
				hasMatchingTag := false
				for _, filterTag := range options.Tags {
					filterTag = strings.TrimSpace(strings.ToLower(filterTag))
					for _, taskTag := range task.Tags {
						if strings.ToLower(taskTag) == filterTag {
							hasMatchingTag = true
							break
						}
					}
					if hasMatchingTag {
						break
					}
				}
				if !hasMatchingTag {
					continue
				}
			}
			
			if options.HasDueDate != nil {
				hasDueDate := !task.DueDate.IsZero()
				if hasDueDate != *options.HasDueDate {
					continue
				}
			}
			
			if options.StartDate != nil || options.EndDate != nil {
				if task.DueDate.IsZero() {
					continue
				}
				
				if options.StartDate != nil && task.DueDate.Before(*options.StartDate) {
					continue
				}
				if options.EndDate != nil && task.DueDate.After(*options.EndDate) {
					continue
				}
			}
			
			filteredTasks = append(filteredTasks, task)
		}
		
		tmpl := template.Must(template.New("index.html").Funcs(templateFuncs).ParseFiles("static/index.html"))
		tmpl.Execute(w, TemplateData{Tasks: filteredTasks})
	})

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
