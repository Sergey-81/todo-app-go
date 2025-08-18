CLI Версия

Сборка
go build -o todo ./cmd/todo-cli

Использование
# Добавить задачу
./todo add --desc="Купить молоко" --tags="покупки,важно"

# Список задач
./todo list
./todo list --filter=completed
./todo list --filter=pending

# Завершить задачу
./todo complete --id=1

# Удалить задачу
./todo delete --id=1

# Экспорт задач
./todo export --format=json --out=tasks_backup.json
./todo export --format=csv --out=tasks_backup.csv

# Импорт задач
./todo load --file=tasks_backup.json