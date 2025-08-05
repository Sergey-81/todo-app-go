# Журнал разработки ToDo-приложения

## 05-08-2025 05:25
### Создание базовой структуры
- Инициализирован Go-модуль: `go mod init todo-app`.
- Созданы директории:
  - `cmd/todo-app` — точка входа.
  - `internal/models` — структуры данных.
  - `docs` — документация.
- Добавлен этот файл `PROJECT_LOG.md`.

Изначальная структура проекта:
C:\Users\Сергей Нижегородов\hello\todo-app\
├── go.mod          # файл модуля Go
├── cmd\            # точка входа
├── internal\       # основной код
└── docs\           # документация

Теперь структура проекта выглядит так:
todo-app/
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   └── todo-app/
│       └── main.go
├── internal/
│   ├── manager/
│   │   ├── task_manager.go
│   │   └── task_manager_test.go
│   ├── models/
│   │   └── task.go
│   └── logger/
│       └── logger.go
└── go.mod