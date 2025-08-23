package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"todo-app/internal/logger"
	"todo-app/internal/manager"
	"todo-app/internal/storage" // 🆕 Добавляем этот импорт!
)

type Bot struct {
	api         *tgbotapi.BotAPI
	taskManager *manager.TaskManager
}

func NewBot(token string, tm *manager.TaskManager) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %v", err)
	}

	bot.Debug = true // Режим отладки
	log.Printf("Авторизован как %s", bot.Self.UserName)

	return &Bot{
		api:         bot,
		taskManager: tm,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Ошибка получения updates: %v", err)
	}

	log.Println("Бот запущен и слушает сообщения...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		go b.handleMessage(update.Message)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	ctx := context.Background()
	
	logger.Info(ctx, "Получено сообщение", 
		"user", msg.From.UserName, 
		"text", msg.Text,
	)

	// Обрабатываем команды
	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	// Обрабатываем обычные сообщения
	b.handleTextMessage(msg)
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.sendWelcomeMessage(msg.Chat.ID)
	case "add":
		b.addTask(msg)
	case "list":
		b.listTasks(msg.Chat.ID)
	case "done":
		b.completeTask(msg)
	case "delete":
		b.deleteTask(msg)
	case "help":
		b.sendHelp(msg.Chat.ID)
	default:
		b.sendMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

func (b *Bot) handleTextMessage(msg *tgbotapi.Message) {
	// Автоматически добавляем задачу из текста
	if strings.TrimSpace(msg.Text) != "" {
		b.addTaskFromText(msg.Chat.ID, msg.Text)
	}
}

func (b *Bot) sendWelcomeMessage(chatID int64) {
	text := `🎯 *Добро пожаловать в TodoBot!*

*Доступные команды:*
/add [задача] - Добавить задачу
/list - Показать все задачи  
/done [номер] - Отметить задачу выполненной
/delete [номер] - Удалить задачу
/help - Помощь

*Примеры:*
/add Купить молоко #покупки
/add Создать отчет до пятницы 🚀
/done 1`

	b.sendMessage(chatID, text)
}

func (b *Bot) addTask(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		b.sendMessage(msg.Chat.ID, "Укажите задачу после команды: /add Купить молоко")
		return
	}

	b.addTaskFromText(msg.Chat.ID, args)
}

func (b *Bot) addTaskFromText(chatID int64, text string) {
	// Парсим теги из текста
	var tags []string
	description := text
	
	if strings.Contains(description, "#") {
		words := strings.Fields(description)
		for _, word := range words {
			if strings.HasPrefix(word, "#") {
				tags = append(tags, strings.TrimPrefix(word, "#"))
			}
		}
		// Убираем теги из описания
		description = strings.TrimSpace(strings.ReplaceAll(description, "#", ""))
	}

	taskID, err := b.taskManager.AddTask(description, tags)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка: "+err.Error())
		return
	}

	response := fmt.Sprintf("✅ *Задача добавлена!*\n\nID: #%d\nЗадача: %s", taskID, description)
	if len(tags) > 0 {
		response += fmt.Sprintf("\nТеги: %s", strings.Join(tags, ", "))
	}

	b.sendMessage(chatID, response)
}

func (b *Bot) listTasks(chatID int64) {
	tasks := b.taskManager.GetAllTasks()
	
	if len(tasks) == 0 {
		b.sendMessage(chatID, "📭 Список задач пуст")
		return
	}

	var response strings.Builder
	response.WriteString("📋 *Ваши задачи:*\n\n")

	for _, task := range tasks {
		status := "🟢"
		if task.Completed {
			status = "✅"
		}

		// Эмодзи приоритета
		priorityEmoji := "⚪"
		switch task.Priority {
		case manager.PriorityLow:
			priorityEmoji = "🔵"
		case manager.PriorityMedium:
			priorityEmoji = "🟡"
		case manager.PriorityHigh:
			priorityEmoji = "🔴"
		}

		response.WriteString(fmt.Sprintf("%s%s #%d: %s", status, priorityEmoji, task.ID, task.Description))

		// Добавляем теги
		if len(task.Tags) > 0 {
			response.WriteString(fmt.Sprintf(" \\#%s", strings.Join(task.Tags, " \\#")))
		}

		response.WriteString("\n\n")
	}

	b.sendMessage(chatID, response.String())
}

func (b *Bot) completeTask(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		b.sendMessage(msg.Chat.ID, "Укажите номер задачи: /done 1")
		return
	}

	taskID, err := strconv.Atoi(args)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "Номер задачи должен быть числом")
		return
	}

	_, err = b.taskManager.ToggleComplete(taskID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Ошибка: "+err.Error())
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("✅ Задача #%d отмечена выполненной!", taskID))
}

func (b *Bot) deleteTask(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		b.sendMessage(msg.Chat.ID, "Укажите номер задачи: /delete 1")
		return
	}

	taskID, err := strconv.Atoi(args)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "Номер задачи должен быть числом")
		return
	}

	err = b.taskManager.DeleteTask(taskID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, "❌ Ошибка: "+err.Error())
		return
	}

	b.sendMessage(msg.Chat.ID, fmt.Sprintf("🗑️ Задача #%d удалена!", taskID))
}

func (b *Bot) sendHelp(chatID int64) {
	helpText := `🤖 *Помощь по командам*

*/start* - Начать работу с ботом
*/add [задача]* - Добавить новую задачу
*/list* - Показать все задачи
*/done [номер]* - Отметить задачу выполненной  
*/delete [номер]* - Удалить задачу
*/help* - Показать эту справку

*Примеры использования:*
/add Купить молоко #покупки
/add Подготовить отчет до пятницы 🚀
/done 1
/list`

	b.sendMessage(chatID, helpText)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func main() {
	ctx := context.Background()
	logger.SetLevel(logger.LevelInfo)
	logger.Info(ctx, "Запуск Telegram-бота...")

	// 🆕 Создаем директорию data если её нет
	if err := os.MkdirAll("data", 0755); err != nil {
		logger.Error(ctx, err, "Ошибка создания директории data")
		return
	}

	// 🆕 Инициализируем SQLite хранилище
	dbStorage, err := storage.NewSQLiteStorage("./data/todoapp.db")
	if err != nil {
		logger.Error(ctx, err, "Ошибка инициализации SQLite хранилища")
		return
	}
	defer dbStorage.Close()

	logger.Info(ctx, "SQLite хранилище успешно инициализировано")

	// 🆕 Создаем менеджер с хранилищем
	taskManager := manager.NewTaskManagerWithStorage(dbStorage)

	// Токен бота (ЗАМЕНИТЕ на реальный токен!)
	botToken := " "

	bot, err := NewBot(botToken, taskManager)
	if err != nil {
		logger.Error(ctx, err, "Ошибка создания бота")
		return
	}

	logger.Info(ctx, "Бот успешно инициализирован")
	bot.Start()
}