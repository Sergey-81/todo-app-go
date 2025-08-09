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

## 05-08-2025 07:55  
### Этап 1: Базовый каркас  
- Настроен CI/CD с тестами и линтерами  
- Реализован модуль логирования (100% coverage)  
- Определены структуры данных (модель Task)

## 05-08-2025 16:46  
### Этап 2: Добавление задач  
- Реализован метод AddTask() с проверками:
  - Непустое описание
  - Ограничение длины (1000 символов)
  - Автоинкремент ID
- Добавлены табличные тесты (92% coverage)
- Обновлена документация методов

## 05-08-2025 19:06
### Реализованы метрики:
todoapp_tasks_added_total (counter) - подсчет операций добавления
todoapp_task_desc_length_bytes (histogram) - распределение длин описаний
todoapp_add_task_duration_seconds (histogram) - время выполнения
Тестирование:
Написаны unit-тесты с 100% покрытием
Подтверждена работа в CI (зеленый маркер в GitHub)
## Мониторинг
Метрики доступны на порту 8080:
```bash
curl http://localhost:8080/metrics

## 06-08-2025 21:31
### Реализация HTTP API для AddTask

**Добавлено:**
- Роутер на базе `chi` (v5)
- Обработчик POST-запросов `/tasks`
- JSON-сериализация запросов/ответов:
  ```go
  type CreateTaskRequest struct {
      Description string `json:"description"`
  }
новая структура проекта:
  todo-app/
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   └── todo-app/
│       └── main.go          # Точка входа (инициализация сервера)
├── internal/
│   ├── manager/
│   │   ├── task_manager.go  # Логика добавления задач
│   │   └── task_manager_test.go
│   ├── models/
│   │   └── task.go          # Модель Task + CreateTaskRequest
│   ├── logger/
│   │   └── logger.go        # Логирование
│   └── server/
│       └── server.go        # Роутер и обработчик API (единственный новый файл!)
│── go.mod
│── go.sum
└── Dockerfile
## 07-08-2025 01:35
### Реализован Docker:
создан Dockerfile
успешно "упаковал" своё Go-приложение в Docker-контейнер
настроил CI/CD → GitHub Actions + Docker Hub

## 09-08-2025 06:46  
### Этап 3: Добавление задач 
- Реализован метод UpdateTask() с проверками:
  - Непустое описание
  - Ограничение длины (1000 символов)
  - Автоинкремент ID
- Добавлены табличные тесты (92% coverage)
- Обновлена документация методов

### Реализована базовая работа HTMX-интерфейса
Открыто в браузере:
http://localhost:8080

Страница отображается корректно:
Заголовок «Мои задачи»
Поле ввода с плейсхолдером «Новая задача...»
Кнопка «Добавить»

новая структура проекта:
  todo-app/
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   └── todo-app/
│       └── main.go          # Точка входа (инициализация сервера)
├── internal/
│   ├── manager/
│   │   ├── task_manager.go  # Логика добавления задач
│   │   └── task_manager_test.go
│   ├── models/
│   │   └── task.go          # Модель Task + CreateTaskRequest
│   ├── logger/
│   │   └── logger.go        # Логирование
│   └── server/
│       └── server.go        # Роутер и обработчик API (единственный новый файл!)
├── static
│       └── index.html
│── go.mod
│── go.sum
└── Dockerfile
