package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// Открываем базу данных
	db, err := sql.Open("sqlite", "./data/todoapp.db")
	if err != nil {
		log.Fatal("Ошибка открытия БД:", err)
	}
	defer db.Close()

	// Добавляем колонку user_id в таблицу tasks
	_, err = db.Exec("ALTER TABLE tasks ADD COLUMN user_id INTEGER REFERENCES users(id)")
	if err != nil {
		log.Fatal("Ошибка добавления колонки user_id:", err)
	}

	// Привязываем все существующие задачи к default пользователю (ID=1)
	_, err = db.Exec("UPDATE tasks SET user_id = 1 WHERE user_id IS NULL")
	if err != nil {
		log.Fatal("Ошибка привязки задач к пользователю:", err)
	}

	log.Println("✅ Колонка user_id добавлена в таблицу tasks!")
	log.Println("✅ Все существующие задачи привязаны к пользователю с ID=1!")
}