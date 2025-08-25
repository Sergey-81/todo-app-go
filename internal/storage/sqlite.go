package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
	"todo-app/internal/manager"
	

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath) // "sqlite" вместо "sqlite3"
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %v", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	// Создаем таблицы
	if err := createTables(db); err != nil {
		return nil, err
	}

	log.Printf("SQLite база данных инициализирована: %s", dbPath)
	return &SQLiteStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
    // Таблица пользователей (ДОБАВЛЯЕМ ПЕРВОЙ)
    createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        device_id TEXT UNIQUE NOT NULL,
        telegram_id INTEGER UNIQUE,
        fcm_token TEXT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL
    )`

    // Таблица задач (уже существует)
    createTasksTable := `
    CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER REFERENCES users(id),
        description TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        priority TEXT NOT NULL DEFAULT 'medium',
        due_date DATETIME,
        tags TEXT
    )`

    // Таблица подзадач (уже существует)
    createSubTasksTable := `
    CREATE TABLE IF NOT EXISTS subtasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER REFERENCES users(id),
        task_id INTEGER NOT NULL,
        description TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        completed BOOLEAN NOT NULL DEFAULT FALSE,
        FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
    )`

    // Создаем таблицы в правильном порядке
    _, err := db.Exec(createUsersTable)
    if err != nil {
        return fmt.Errorf("ошибка создания таблицы users: %v", err)
    }

    _, err = db.Exec(createTasksTable)
    if err != nil {
        return fmt.Errorf("ошибка создания таблицы tasks: %v", err)
    }

    _, err = db.Exec(createSubTasksTable)
    if err != nil {
        return fmt.Errorf("ошибка создания таблицы subtasks: %v", err)
    }

    return nil
}

// Закрытие соединения
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Методы для работы с задачами
func (s *SQLiteStorage) AddTask(description string, tags []string) (int, error) {
    // Для обратной совместимости - используем user_id = 1
    return s.AddTaskForUser(1, description, tags)
}

// AddTaskForUser - новый метод для добавления задач с указанием пользователя
func (s *SQLiteStorage) AddTaskForUser(userID int, description string, tags []string) (int, error) {
    query := `
    INSERT INTO tasks (description, created_at, updated_at, completed, priority, due_date, tags, user_id)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

    now := time.Now()
    tagsStr := ""
    if len(tags) > 0 {
        tagsStr = strings.Join(tags, ",")
    }

    result, err := s.db.Exec(query, 
        description, now, now, false, "medium", nil, tagsStr, userID)
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    return int(id), err
}

func (s *SQLiteStorage) GetAllTasks() ([]manager.Task, error) {
	query := `
	SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id
	FROM tasks ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []manager.Task
	for rows.Next() {
		var task manager.Task
		var dueDate sql.NullTime
		var tagsStr sql.NullString
		var priority string
		var userID int

		err := rows.Scan(
			&task.ID, &task.Description, &task.CreatedAt, &task.UpdatedAt,
			&task.Completed, &priority, &dueDate, &tagsStr, &userID,
		)
		if err != nil {
			return nil, err
		}

		task.UserID = userID
		task.Priority = manager.Priority(priority)

		if dueDate.Valid {
			task.DueDate = dueDate.Time
		}

		if tagsStr.Valid && tagsStr.String != "" {
			task.Tags = strings.Split(tagsStr.String, ",")
		} else {
			task.Tags = []string{}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *SQLiteStorage) GetTask(id int) (*manager.Task, error) {
	query := `
	SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id
	FROM tasks WHERE id = ?`

	var task manager.Task
	var dueDate sql.NullTime
	var tagsStr sql.NullString
	var priority string
	var userID int

	err := s.db.QueryRow(query, id).Scan(
		&task.ID, &task.Description, &task.CreatedAt, &task.UpdatedAt,
		&task.Completed, &priority, &dueDate, &tagsStr, &userID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("задача с ID %d не найдена", id)
		}
		return nil, err
	}

	task.UserID = userID
	task.Priority = manager.Priority(priority)

	if dueDate.Valid {
		task.DueDate = dueDate.Time
	}

	if tagsStr.Valid && tagsStr.String != "" {
		task.Tags = strings.Split(tagsStr.String, ",")
	}

	return &task, nil
}

func (s *SQLiteStorage) UpdateTask(id int, req manager.UpdateTaskRequest) (*manager.Task, error) {
	// Сначала получаем текущую задачу
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	// Обновляем поля
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Completed != nil {
		task.Completed = *req.Completed
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = *req.DueDate
	}
	if req.Tags != nil {
		task.Tags = *req.Tags
	}

	task.UpdatedAt = time.Now()

	// Обновляем в базе
	query := `
	UPDATE tasks 
	SET description = ?, updated_at = ?, completed = ?, priority = ?, due_date = ?, tags = ?
	WHERE id = ?`

	tagsStr := ""
	if len(task.Tags) > 0 {
		tagsStr = strings.Join(task.Tags, ",")
	}

	var dueDate interface{}
	if task.DueDate.IsZero() {
		dueDate = nil
	} else {
		dueDate = task.DueDate
	}

	_, err = s.db.Exec(query,
		task.Description, task.UpdatedAt, task.Completed,
		string(task.Priority), dueDate, tagsStr, id,
	)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *SQLiteStorage) DeleteTask(id int) error {
	query := "DELETE FROM tasks WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("задача с ID %d не найдена", id)
	}

	return nil
}

func (s *SQLiteStorage) ToggleComplete(id int) (*manager.Task, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	task.Completed = !task.Completed
	task.UpdatedAt = time.Now()

	query := "UPDATE tasks SET completed = ?, updated_at = ? WHERE id = ?"
	_, err = s.db.Exec(query, task.Completed, task.UpdatedAt, id)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Методы для подзадач
func (s *SQLiteStorage) AddSubTask(taskID int, description string) (int, error) {
	query := `
	INSERT INTO subtasks (task_id, description, created_at, updated_at, completed)
	VALUES (?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := s.db.Exec(query, taskID, description, now, now, false)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (s *SQLiteStorage) GetSubTasks(taskID int) ([]manager.SubTask, error) {
	query := `
	SELECT id, task_id, description, created_at, updated_at, completed
	FROM subtasks WHERE task_id = ? ORDER BY created_at`

	rows, err := s.db.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subtasks []manager.SubTask
	for rows.Next() {
		var subtask manager.SubTask
		err := rows.Scan(
			&subtask.ID, &subtask.TaskID, &subtask.Description,
			&subtask.CreatedAt, &subtask.UpdatedAt, &subtask.Completed,
		)
		if err != nil {
			return nil, err
		}
		subtasks = append(subtasks, subtask)
	}

	return subtasks, nil
}

func (s *SQLiteStorage) ToggleSubTask(id int) error {
	// Получаем текущий статус
	var completed bool
	err := s.db.QueryRow("SELECT completed FROM subtasks WHERE id = ?", id).Scan(&completed)
	if err != nil {
		return err
	}

	// Инвертируем статус
	query := "UPDATE subtasks SET completed = ?, updated_at = ? WHERE id = ?"
	_, err = s.db.Exec(query, !completed, time.Now(), id)
	return err
}

func (s *SQLiteStorage) DeleteSubTask(id int) error {
	query := "DELETE FROM subtasks WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("подзадача с ID %d не найдена", id)
	}

	return nil
}

// Методы фильтрации (упрощенные версии)
func (s *SQLiteStorage) FilterTasks(completed *bool) ([]manager.Task, error) {
    query := "SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id FROM tasks"
    if completed != nil {
        query += " WHERE completed = ?"
    }
    query += " ORDER BY created_at DESC"

    var rows *sql.Rows
    var err error

    if completed != nil {
        rows, err = s.db.Query(query, *completed)
    } else {
        rows, err = s.db.Query(query)
    }

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanTasks(rows)
}

// Вспомогательная функция для сканирования задач
func scanTasks(rows *sql.Rows) ([]manager.Task, error) {
	var tasks []manager.Task
	for rows.Next() {
		var task manager.Task
		var dueDate sql.NullTime
		var tagsStr sql.NullString
		var priority string
		var userID int

		err := rows.Scan(
			&task.ID, &task.Description, &task.CreatedAt, &task.UpdatedAt,
			&task.Completed, &priority, &dueDate, &tagsStr, &userID,
		)
		if err != nil {
			return nil, err
		}

		task.UserID = userID
		task.Priority = manager.Priority(priority)

		if dueDate.Valid {
			task.DueDate = dueDate.Time
		}

		if tagsStr.Valid && tagsStr.String != "" {
			task.Tags = strings.Split(tagsStr.String, ",")
		} else {
			task.Tags = []string{}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Фильтрация по приоритету
func (s *SQLiteStorage) FilterByPriority(priority manager.Priority) ([]manager.Task, error) {
	query := "SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id FROM tasks WHERE priority = ? ORDER BY created_at DESC"
	
	rows, err := s.db.Query(query, string(priority))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

// Фильтрация по тегу
func (s *SQLiteStorage) FilterByTag(tag string) ([]manager.Task, error) {
    query := `
        SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id 
        FROM tasks 
        WHERE tags LIKE ? 
        ORDER BY created_at DESC`
    
    rows, err := s.db.Query(query, "%"+strings.TrimSpace(tag)+"%")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanTasks(rows)
}

// Предстоящие задачи
func (s *SQLiteStorage) GetUpcomingTasks(days int) ([]manager.Task, error) {
	query := `
	SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id 
	FROM tasks 
	WHERE due_date BETWEEN date('now') AND date('now', ? || ' days') 
	AND completed = false 
	ORDER BY due_date`
	
	rows, err := s.db.Query(query, fmt.Sprintf("+%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (s *SQLiteStorage) FilterByDateRange(start, end time.Time) ([]manager.Task, error) {
    query := `
        SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id 
        FROM tasks 
        WHERE due_date BETWEEN ? AND ?
        ORDER BY due_date`
    
    rows, err := s.db.Query(query, start.Format("2006-01-02"), end.Format("2006-01-02"))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanTasks(rows)
}

// FilterTasksAdvanced - расширенная фильтрация
func (s *SQLiteStorage) FilterTasksAdvanced(options manager.FilterOptions) ([]manager.Task, error) {
    query := "SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id FROM tasks WHERE 1=1"
    var args []interface{}
    
    // Фильтр по статусу
    if options.Completed != nil {
        query += " AND completed = ?"
        args = append(args, *options.Completed)
    }
    
    // Фильтр по приоритету
    if options.Priority != nil {
        query += " AND priority = ?"
        args = append(args, string(*options.Priority))
    }
    
    // Фильтр по тегам (простая реализация)
    if len(options.Tags) > 0 {
        for _, tag := range options.Tags {
            query += " AND tags LIKE ?"
            args = append(args, "%"+tag+"%")
        }
    }
    
    // Фильтр по диапазону дат
    if options.StartDate != nil && options.EndDate != nil {
        query += " AND due_date BETWEEN ? AND ?"
        args = append(args, *options.StartDate, *options.EndDate)
    } else if options.StartDate != nil {
        query += " AND due_date >= ?"
        args = append(args, *options.StartDate)
    } else if options.EndDate != nil {
        query += " AND due_date <= ?"
        args = append(args, *options.EndDate)
    }
    
    // Фильтр по наличию дата
    if options.HasDueDate != nil {
        if *options.HasDueDate {
            query += " AND due_date IS NOT NULL"
        } else {
            query += " AND due_date IS NULL"
        }
    }
    
    query += " ORDER BY created_at DESC"
    
    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanTasks(rows)
}

// 🆕 Методы для работы с пользователями
func (s *SQLiteStorage) CreateUser(user *manager.User) (int, error) {
    query := `
    INSERT INTO users (device_id, telegram_id, fcm_token, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?)`

    result, err := s.db.Exec(query,
        user.DeviceID,
        user.TelegramID,
        user.FCMToken,
        user.CreatedAt,
        user.UpdatedAt,
    )
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    return int(id), err
}

func (s *SQLiteStorage) GetUserByTelegramID(telegramID int64) (*manager.User, error) {
    query := `SELECT id, device_id, telegram_id, fcm_token, created_at, updated_at 
              FROM users WHERE telegram_id = ?`

    var user manager.User
    err := s.db.QueryRow(query, telegramID).Scan(
        &user.ID,
        &user.DeviceID,
        &user.TelegramID,
        &user.FCMToken,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func (s *SQLiteStorage) GetAllTasksForUser(userID int) ([]manager.Task, error) {
    query := `
    SELECT id, description, created_at, updated_at, completed, priority, due_date, tags, user_id
    FROM tasks WHERE user_id = ? ORDER BY created_at DESC`

    rows, err := s.db.Query(query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanTasks(rows)
}

func (s *SQLiteStorage) GetUserByDeviceID(deviceID string) (*manager.User, error) {
    query := `SELECT id, device_id, telegram_id, fcm_token, created_at, updated_at 
              FROM users WHERE device_id = ?`

    var user manager.User
    err := s.db.QueryRow(query, deviceID).Scan(
        &user.ID,
        &user.DeviceID,
        &user.TelegramID,
        &user.FCMToken,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func (s *SQLiteStorage) UpdateUser(user *manager.User) error {
    query := `
    UPDATE users 
    SET device_id = ?, telegram_id = ?, fcm_token = ?, updated_at = ?
    WHERE id = ?`

    _, err := s.db.Exec(query,
        user.DeviceID,
        user.TelegramID,
        user.FCMToken,
        time.Now(),
        user.ID,
    )
    return err
}

func (s *SQLiteStorage) GetUserByID(userID int) (*manager.User, error) {
    query := `SELECT id, device_id, telegram_id, fcm_token, created_at, updated_at 
              FROM users WHERE id = ?`

    var user manager.User
    err := s.db.QueryRow(query, userID).Scan(
        &user.ID,
        &user.DeviceID,
        &user.TelegramID,
        &user.FCMToken,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func (s *SQLiteStorage) MigrateExistingTasksToUser(userID int, deviceID string) error {
	// Привязываем все существующие задачи к пользователю
	query := `UPDATE tasks SET user_id = ? WHERE user_id IS NULL OR user_id = 1`
	_, err := s.db.Exec(query, userID)
	return err
}