package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"todo-app/internal/logger"
)

type UserManager struct {
	mu      sync.Mutex
	storage Storage
	users   map[int]*User // 🆕 Добавляем in-memory хранилище
	nextID  int           // 🆕 Счетчик для in-memory режима
}

func NewUserManager(storage Storage) *UserManager {
	return &UserManager{
		storage: storage,
		users:   make(map[int]*User), // 🆕 Инициализируем
		nextID:  1,                   // 🆕 Начинаем с 1
	}
}

// GenerateDeviceID создает уникальный ID устройства
func (um *UserManager) GenerateDeviceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CreateUser создает нового пользователя
func (um *UserManager) CreateUser(deviceID string, telegramID int64) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	user := &User{
		DeviceID:   deviceID,
		TelegramID: telegramID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 🆕 Используем хранилище
	if um.storage != nil {
		id, err := um.storage.CreateUser(user)
		if err != nil {
			return nil, err
		}
		user.ID = id
		logger.Info(context.Background(), "Пользователь создан в хранилище", "userID", id, "deviceID", deviceID)
		return user, nil
	}

	// In-memory логика (для обратной совместимости)
	user.ID = um.nextID
	um.users[user.ID] = user
	um.nextID++
	logger.Info(context.Background(), "Пользователь создан в памяти", "userID", user.ID, "deviceID", deviceID)
	return user, nil
}

// GetUserByDeviceID возвращает пользователя по device_id
func (um *UserManager) GetUserByDeviceID(deviceID string) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.storage != nil {
		return um.storage.GetUserByDeviceID(deviceID)
	}

	// In-memory поиск
	for _, user := range um.users {
		if user.DeviceID == deviceID {
			return user, nil
		}
	}
	return nil, fmt.Errorf("пользователь не найден")
}

// UpdateUser обновляет данные пользователя
func (um *UserManager) UpdateUser(user *User) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.storage != nil {
		return um.storage.UpdateUser(user)
	}

	// In-memory обновление
	if _, exists := um.users[user.ID]; !exists {
		return fmt.Errorf("пользователь не найден")
	}

	user.UpdatedAt = time.Now()
	um.users[user.ID] = user
	return nil
}

// GetUserByID возвращает пользователя по ID (новый метод)
func (um *UserManager) GetUserByID(userID int) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.storage != nil {
		return um.storage.GetUserByID(userID)
	}

	// In-memory поиск
	if user, exists := um.users[userID]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("пользователь не найден")
}

// GetOrCreateUserByTelegramID - новый метод для получения или создания пользователя по Telegram ID
func (um *UserManager) GetOrCreateUserByTelegramID(telegramID int64) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.storage != nil {
		// Пытаемся найти существующего пользователя
		user, err := um.storage.GetUserByTelegramID(telegramID)
		if err == nil {
			return user, nil
		}
		
		// Если не найден - создаем нового
		deviceID := fmt.Sprintf("telegram_%d", telegramID)
		user = &User{
			DeviceID:   deviceID,
			TelegramID: telegramID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		
		id, err := um.storage.CreateUser(user)
		if err != nil {
			return nil, err
		}
		user.ID = id
		logger.Info(context.Background(), "Пользователь создан по Telegram ID", "userID", id, "telegramID", telegramID)
		return user, nil
	}

	// In-memory логика
	for _, user := range um.users {
		if user.TelegramID == telegramID {
			return user, nil
		}
	}
	
	// Создаем нового пользователя
	deviceID := fmt.Sprintf("telegram_%d", telegramID)
	user := &User{
		ID:         um.nextID,
		DeviceID:   deviceID,
		TelegramID: telegramID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	um.users[user.ID] = user
	um.nextID++
	
	logger.Info(context.Background(), "Пользователь создан в памяти по Telegram ID", "userID", user.ID, "telegramID", telegramID)
	return user, nil
}

// GetUserByTelegramID возвращает пользователя по Telegram ID
func (um *UserManager) GetUserByTelegramID(telegramID int64) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if um.storage != nil {
		return um.storage.GetUserByTelegramID(telegramID)
	}

	// In-memory поиск
	for _, user := range um.users {
		if user.TelegramID == telegramID {
			return user, nil
		}
	}
	return nil, fmt.Errorf("пользователь не найден")
}