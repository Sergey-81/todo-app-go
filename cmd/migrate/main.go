package main

import (
	"database/sql"
	//"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	log.Println("🔄 Создание новой базы данных...")

	// Убедимся что папка существует
	os.MkdirAll("data", 0755)

	// Простое создание БД
	db, err := sql.Open("sqlite", "./data/todoapp.db")
	if err != nil {
		log.Fatal("❌ Ошибка открытия БД:", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatal("❌ Ошибка подключения:", err)
	}

	log.Println("✅ База данных создана!")

	// Создаем таблицы
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT UNIQUE NOT NULL,
			telegram_id INTEGER UNIQUE,
			fcm_token TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id),
			description TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			priority TEXT NOT NULL DEFAULT 'medium',
			due_date DATETIME,
			tags TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS subtasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id),
			task_id INTEGER NOT NULL,
			description TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
		)`,
	}

	for i, table := range tables {
		_, err = db.Exec(table)
		if err != nil {
			log.Fatal("❌ Ошибка создания таблицы:", err)
		}
		log.Printf("✅ Таблица %d создана", i+1)
	}

	// Создаем default пользователя
	_, err = db.Exec(`INSERT INTO users (device_id, created_at, updated_at) 
		VALUES ('default_legacy_user', datetime('now'), datetime('now'))`)
	if err != nil {
		log.Println("⚠️ Пользователь уже существует или ошибка:", err)
	} else {
		log.Println("✅ Default пользователь создан")
	}
_, err = db.Exec(`UPDATE users SET telegram_id = MY_ID NUMBER WHERE device_id = 'default_legacy_user'`)
if err != nil {
    log.Println("⚠️ Ошибка привязки Telegram ID:", err)
} else {
    log.Println("✅ Ваш Telegram ID привязан к default пользователю")
}
	log.Println("🎉 Миграция завершена успешно!")
	log.Println("📁 База данных: data/todoapp.db")
}