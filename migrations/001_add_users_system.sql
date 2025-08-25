-- Добавляем таблицу пользователей
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT UNIQUE,          -- Уникальный ID устройства
    telegram_id INTEGER UNIQUE,     -- ID Telegram аккаунта
    fcm_token TEXT,                 -- Токен для push-уведомлений
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем user_id к существующей таблице задач
ALTER TABLE tasks ADD COLUMN user_id INTEGER REFERENCES users(id);

-- Добавляем user_id к подзадачам
ALTER TABLE subtasks ADD COLUMN user_id INTEGER REFERENCES users(id);

-- Создаем индекс для быстрого поиска задач пользователя
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_subtasks_user_id ON subtasks(user_id);

-- Создаем таблицу для хранения настроек напоминаний
CREATE TABLE IF NOT EXISTS reminder_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER UNIQUE REFERENCES users(id),
    enabled BOOLEAN DEFAULT TRUE,
    remind_before_days INTEGER DEFAULT 1,
    telegram_notifications BOOLEAN DEFAULT TRUE,
    push_notifications BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);