package manager

import (
	
	//"bytes"
	//"os"
	"strings"
	"testing"
	//"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestAddTask(t *testing.T) {
	tm := &TaskManager{}

	id, err := tm.AddTask("Купить молоко")
	if err != nil {
		t.Fatalf("Ошибка при добавлении задачи: %v", err)
	}

	if id != 1 {
		t.Errorf("Ожидался ID=1, получено %d", id)
	}

	if len(tm.tasks) != 1 {
		t.Errorf("Ожидалась 1 задача, получено %d", len(tm.tasks))
	}
}

func TestAddEmptyTask(t *testing.T) {
	tm := &TaskManager{}

	_, err := tm.AddTask("")
	if err == nil {
		t.Error("Ожидалась ошибка при пустом описании")
	}
}

func TestAddTaskWithMaxLength(t *testing.T) {
    tm := &TaskManager{}
    
    // Генерируем строку длиной ровно 1000 символов
    validDesc := strings.Repeat("a", 1000)
    _, err := tm.AddTask(validDesc)
    if err != nil {
        t.Errorf("Ожидалась успешная валидация для 1000 символов: %v", err)
    }
    
    // Тест на 1001 символ
    invalidDesc := validDesc + "a"
    _, err = tm.AddTask(invalidDesc)
    if err == nil {
        t.Error("Ожидалась ошибка при 1001 символе")
    }
}

func TestAddTaskMetrics(t *testing.T) {
	// Сохраняем оригинальные метрики
	originalAddTaskCount := addTaskCount
	originalTaskDescLength := taskDescLength

	// Создаем новый регистр для тестов
	registry := prometheus.NewRegistry()

	// Создаем тестовые метрики
	testAddTaskCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Test counter",
		},
		[]string{"status"},
	)

	testTaskDescLength := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_task_desc_length_bytes",
			Help:    "Test histogram",
			Buckets: []float64{50, 100, 500, 1000},
		},
	)

	// Регистрируем метрики
	registry.MustRegister(testAddTaskCount)
	registry.MustRegister(testTaskDescLength)

	// Подменяем глобальные метрики
	addTaskCount = testAddTaskCount
	taskDescLength = testTaskDescLength

	// Восстанавливаем оригинальные метрики после теста
	defer func() {
		addTaskCount = originalAddTaskCount
		taskDescLength = originalTaskDescLength
	}()

	tm := &TaskManager{}

	// Тест 1: Успешное добавление
	desc := "Valid description"
	_, err := tm.AddTask(desc)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Проверяем счетчик успешных операций
	if successCount := testutil.ToFloat64(testAddTaskCount.WithLabelValues("success")); successCount != 1 {
		t.Errorf("Expected 1 success, got %v", successCount)
	}

	// Проверяем гистограмму через сбор метрик
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	foundHistogram := false
	for _, mf := range metrics {
		if mf.GetName() == "todoapp_task_desc_length_bytes" {
			foundHistogram = true
			if len(mf.GetMetric()) == 0 {
				t.Error("Histogram has no samples")
			}
			break
		}
	}

	if !foundHistogram {
		t.Error("Histogram metric not found")
	}

	// Тест 2: Ошибочное добавление
	_, err = tm.AddTask("")
	if err == nil {
		t.Error("Expected error for empty description")
	}

	// Проверяем счетчик ошибок
	if errCount := testutil.ToFloat64(testAddTaskCount.WithLabelValues("error")); errCount != 1 {
		t.Errorf("Expected 1 error, got %v", errCount)
	}
}