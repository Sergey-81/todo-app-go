package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	//"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"todo-app/internal/logger"
	"todo-app/internal/manager"
	"todo-app/internal/storage"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	taskManager *manager.TaskManager
	storage     manager.Storage
	userManager *manager.UserManager
}

func NewBot(token string, tm *manager.TaskManager, storage manager.Storage, um *manager.UserManager) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %v", err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	return &Bot{
		api:         bot,
		taskManager: tm,
		storage:     storage,
		userManager: um,
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

	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	b.handleTextMessage(msg)
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.handleStartCommand(msg)
	case "add":
		b.addTask(msg)
	case "list":
		b.handleListCommand(msg)
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
	if strings.TrimSpace(msg.Text) != "" {
		b.addTaskFromText(msg.Chat.ID, msg.From.ID, msg.Text)
	}
}

func (b *Bot) handleStartCommand(msg *tgbotapi.Message) {
    // Создаем или получаем пользователя
    _, err := b.userManager.GetOrCreateUserByTelegramID(int64(msg.From.ID))
    if err != nil {
        b.sendMessage(msg.Chat.ID, "❌ Ошибка создания пользователя: " + err.Error())
        return
    }

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

	b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) handleListCommand(msg *tgbotapi.Message) {
    // ВСЕГДА используем default пользователя
    defaultUser, err := b.userManager.GetUserByDeviceID("default_legacy_user")
    if err != nil {
        b.sendMessage(msg.Chat.ID, "❌ Ошибка: система не настроена")
        return
    }

    tasks, err := b.taskManager.GetAllTasksForUser(defaultUser.ID)
    if err != nil {
        b.sendMessage(msg.Chat.ID, "❌ Ошибка загрузки задач: "+err.Error())
        return
    }

    if len(tasks) == 0 {
        b.sendMessage(msg.Chat.ID, "📭 Список задач пуст")
        return
    }

	// Формируем список задач
	var response strings.Builder
	response.WriteString("📋 *Ваши задачи:*\n\n")
	
	for i, task := range tasks {
		status := "❌"
		if task.Completed {
			status = "✅"
		}
		
		priorityEmoji := "⚪"
		switch task.Priority {
		case manager.PriorityLow:
			priorityEmoji = "🔵"
		case manager.PriorityMedium:
			priorityEmoji = "🟡"
		case manager.PriorityHigh:
			priorityEmoji = "🔴"
		}

		response.WriteString(fmt.Sprintf("%d. %s%s %s", i+1, status, priorityEmoji, task.Description))

		if len(task.Tags) > 0 {
			response.WriteString(fmt.Sprintf(" \\#%s", strings.Join(task.Tags, " \\#")))
		}

		if !task.DueDate.IsZero() {
			response.WriteString(fmt.Sprintf("\n   📅 %s", task.DueDate.Format("02.01.2006")))
		}

		response.WriteString("\n\n")
	}

	b.sendMessage(msg.Chat.ID, response.String())
}

func (b *Bot) addTask(msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	if args == "" {
		b.sendMessage(msg.Chat.ID, "Укажите задачу после команды: /add Купить молоко")
		return
	}

	b.addTaskFromText(msg.Chat.ID, msg.From.ID, args)
}

func (b *Bot) addTaskFromText(chatID int64, userID int, text string) {
    // ВСЕГДА используем default пользователя из веб-интерфейса
    defaultUser, err := b.userManager.GetUserByDeviceID("default_legacy_user")
    if err != nil {
        // Если default пользователь не найден, создаем его
        defaultUser, err = b.userManager.CreateUser("default_legacy_user", 0)
        if err != nil {
            b.sendMessage(chatID, "❌ Ошибка создания пользователя: "+err.Error())
            return
        }
    }

    var tags []string
    description := text
    
    // Парсим теги
    if strings.Contains(description, "#") {
        words := strings.Fields(description)
        for _, word := range words {
            if strings.HasPrefix(word, "#") {
                tags = append(tags, strings.TrimPrefix(word, "#"))
            }
        }
        description = strings.TrimSpace(strings.ReplaceAll(description, "#", ""))
        for _, tag := range tags {
            description = strings.ReplaceAll(description, tag, "")
        }
        description = strings.TrimSpace(description)
    }

    // Добавляем задачу для default пользователя
    taskID, err := b.taskManager.AddTaskForUser(defaultUser.ID, description, tags)
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

	// Токен бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		botToken = "MY_TELEGRAM_TOKEN" // fallback
	}

	bot, err := NewBot(botToken, taskManager, dbStorage, userManager)
	if err != nil {
		logger.Error(ctx, err, "Ошибка создания бота")
		return
	}

	logger.Info(ctx, "Бот успешно инициализирован")
	bot.Start()
}