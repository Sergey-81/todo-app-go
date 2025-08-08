package manager

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestAddTask(t *testing.T) {
	tm := &TaskManager{}

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
	tm := &TaskManager{}
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
	tm := &TaskManager{}
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
	tm := &TaskManager{}
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

func TestMetrics(t *testing.T) {
	t.Run("Метрики AddTask", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		testCounter := prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_counter"}, []string{"status"})
		registry.MustRegister(testCounter)
		
		original := addTaskCount
		addTaskCount = testCounter
		defer func() { addTaskCount = original }()

		tm := &TaskManager{}
		tm.AddTask("Тест метрик")
		
		if val := testutil.ToFloat64(testCounter.WithLabelValues("success")); val != 1 {
			t.Errorf("Ожидалось 1, получено %v", val)
		}
	})

	t.Run("Метрики UpdateTask", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		testCounter := prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_counter"}, []string{"status"})
		registry.MustRegister(testCounter)
		
		original := updateTaskCount
		updateTaskCount = testCounter
		defer func() { updateTaskCount = original }()

		tm := &TaskManager{}
		id, _ := tm.AddTask("Тест метрик")
		tm.UpdateTask(id, UpdateTaskRequest{Completed: ptr(true)})
		
		if val := testutil.ToFloat64(testCounter.WithLabelValues("success")); val != 1 {
			t.Errorf("Ожидалось 1, получено %v", val)
		}
	})
}

func ptr[T any](v T) *T {
    return &v
}