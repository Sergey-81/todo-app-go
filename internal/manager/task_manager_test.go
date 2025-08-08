package manager

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestAddTask(t *testing.T) {
	tm := NewTaskManager()

	t.Run("Успешное добавление", func(t *testing.T) {
		id, err := tm.AddTask("Купить молоко")
		if err != nil {
			t.Fatalf("Ошибка при добавлении: %v", err)
		}
		if id != 1 {
			t.Errorf("Ожидался ID=1, получен %d", id)
		}
	})

	t.Run("Пустое описание", func(t *testing.T) {
		_, err := tm.AddTask("")
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}
	})

	t.Run("Длина описания", func(t *testing.T) {
		validDesc := strings.Repeat("a", 1000)
		_, err := tm.AddTask(validDesc)
		if err != nil {
			t.Errorf("Ошибка при валидной длине: %v", err)
		}

		invalidDesc := strings.Repeat("a", 1001)
		_, err = tm.AddTask(invalidDesc)
		if err == nil {
			t.Error("Ожидалась ошибка при слишком длинном описании")
		}
	})
}

func TestUpdateTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Исходная задача")

	t.Run("Обновление описания", func(t *testing.T) {
		newDesc := "Новое описание"
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &newDesc})
		if err != nil {
			t.Fatalf("Ошибка при обновлении: %v", err)
		}
		if updated.Description != newDesc {
			t.Errorf("Описание не обновилось: ожидалось '%s', получено '%s'", newDesc, updated.Description)
		}
	})

	t.Run("Несуществующий ID", func(t *testing.T) {
		_, err := tm.UpdateTask(999, UpdateTaskRequest{})
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}
	})

	t.Run("Пустое описание", func(t *testing.T) {
		empty := ""
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &empty})
		if err == nil {
			t.Error("Ожидалась ошибка валидации пустого описания")
		}
	})
}

func TestUpdateTask_EdgeCases(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Тестовая задача")

	t.Run("Обновление только статуса", func(t *testing.T) {
		completed := true
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})
		if err != nil {
			t.Fatalf("Ошибка при обновлении статуса: %v", err)
		}
		if !updated.Completed {
			t.Error("Статус не был обновлен")
		}
	})

	t.Run("Слишком длинное описание", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &longDesc})
		if err == nil {
			t.Error("Ожидалась ошибка слишком длинного описания")
		}
	})
}

func TestGetTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Тестовая задача")

	t.Run("Существующая задача", func(t *testing.T) {
		task, err := tm.GetTask(id)
		if err != nil {
			t.Fatalf("Ошибка при получении: %v", err)
		}
		if task.ID != id {
			t.Errorf("Ожидался ID %d, получен %d", id, task.ID)
		}
	})

	t.Run("Несуществующая задача", func(t *testing.T) {
		_, err := tm.GetTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	tm := NewTaskManager()
	var wg sync.WaitGroup
	iterations := 100

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_, _ = tm.AddTask("Конкурентная задача")
		}()
	}
	wg.Wait()

	// Проверка через GetTask
	if _, err := tm.GetTask(iterations); err != nil {
		t.Errorf("Не все задачи были добавлены, последняя ошибка: %v", err)
	}
}

func TestMetrics(t *testing.T) {
	// Тест метрик AddTask
	t.Run("Метрики AddTask", func(t *testing.T) {
		// Сохраняем оригинальные метрики
		origAddCount := addTaskCount
		origAddDuration := addTaskDuration
		origDescLength := taskDescLength
		defer func() {
			addTaskCount = origAddCount
			addTaskDuration = origAddDuration
			taskDescLength = origDescLength
		}()

		// Создаем тестовый регистратор
		registry := prometheus.NewRegistry()

		// Создаем тестовые метрики
		testAddCounter := promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{Name: "test_add_counter"}, []string{"status"})
		testAddDuration := promauto.With(registry).NewHistogram(
			prometheus.HistogramOpts{Name: "test_add_duration"})
		testDescLength := promauto.With(registry).NewHistogram(
			prometheus.HistogramOpts{Name: "test_desc_length"})

		// Подменяем метрики
		addTaskCount = testAddCounter
		addTaskDuration = testAddDuration
		taskDescLength = testDescLength

		tm := NewTaskManager()
		tm.AddTask("Тест метрик")

		// Проверяем счетчик
		if val := testutil.ToFloat64(testAddCounter.WithLabelValues("success")); val != 1 {
			t.Errorf("Ожидалось 1 успешное добавление, получено %v", val)
		}
	})

	// Тест метрик UpdateTask
	t.Run("Метрики UpdateTask", func(t *testing.T) {
		// Сохраняем оригинальные метрики
		origUpdateCount := updateTaskCount
		origUpdateDuration := updateTaskDuration
		defer func() {
			updateTaskCount = origUpdateCount
			updateTaskDuration = origUpdateDuration
		}()

		// Создаем тестовый регистратор
		registry := prometheus.NewRegistry()

		// Создаем тестовые метрики
		testUpdateCounter := promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{Name: "test_update_counter"}, []string{"status"})
		testUpdateDuration := promauto.With(registry).NewHistogram(
			prometheus.HistogramOpts{Name: "test_update_duration"})

		// Подменяем метрики
		updateTaskCount = testUpdateCounter
		updateTaskDuration = testUpdateDuration

		tm := NewTaskManager()
		id, _ := tm.AddTask("Тест метрик")
		completed := true
		tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})

		// Проверяем счетчик
		if val := testutil.ToFloat64(testUpdateCounter.WithLabelValues("success")); val != 1 {
			t.Errorf("Ожидалось 1 успешное обновление, получено %v", val)
		}
	})
}