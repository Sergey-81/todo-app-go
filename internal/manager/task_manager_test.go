package manager

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager()
	if tm == nil {
		t.Fatal("NewTaskManager() вернул nil")
	}
	if tm.nextID != 1 {
		t.Errorf("Ожидался nextID=1, получен %d", tm.nextID)
	}
	if len(tm.tasks) != 0 {
		t.Errorf("Ожидался пустой список задач, получено %d задач", len(tm.tasks))
	}
}

func TestAddTask(t *testing.T) {
	tm := NewTaskManager()

	t.Run("Успешное добавление задачи", func(t *testing.T) {
		id, err := tm.AddTask("Новая задача", []string{"тег1", "тег2"})
		if err != nil {
			t.Fatalf("Ошибка при добавлении задачи: %v", err)
		}
		if id != 1 {
			t.Errorf("Ожидался ID=1, получен %d", id)
		}
		
		task, _ := tm.GetTask(id)
		if len(task.Tags) != 2 {
			t.Errorf("Ожидалось 2 тега, получено %d", len(task.Tags))
		}
	})

	t.Run("Пустое описание задачи", func(t *testing.T) {
		_, err := tm.AddTask("", nil)
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}
	})

	t.Run("Слишком длинное описание", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		_, err := tm.AddTask(longDesc, nil)
		if err == nil {
			t.Error("Ожидалась ошибка при слишком длинном описании")
		}
	})

	t.Run("Нормализация тегов", func(t *testing.T) {
		id, _ := tm.AddTask("Задача", []string{" ТЕГ1 ", " тег1 ", "тег2", "", "  "})
		task, _ := tm.GetTask(id)
		
		if len(task.Tags) != 2 {
			t.Errorf("Ожидалось 2 уникальных тега после нормализации, получено %d: %v", 
				len(task.Tags), task.Tags)
		}
		
		// Проверяем что теги нормализованы (пробелы убраны)
		for _, tag := range task.Tags {
			if strings.Contains(tag, " ") {
				t.Errorf("Тег содержит пробелы: '%s'", tag)
			}
		}
	})

	t.Run("Нормализация регистра тегов", func(t *testing.T) {
		id, _ := tm.AddTask("Задача", []string{"Тег", "тег", "ТЕГ"})
		task, _ := tm.GetTask(id)
		
		if len(task.Tags) != 1 {
			t.Errorf("Ожидалось 1 уникальный тег (регистронезависимый), получено %d: %v", 
				len(task.Tags), task.Tags)
		}
	})
}

func TestUpdateTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Исходная задача", nil)

	t.Run("Обновление только описания", func(t *testing.T) {
		newDesc := "Новое описание"
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &newDesc})
		if err != nil {
			t.Fatalf("Ошибка при обновлении: %v", err)
		}
		if updated.Description != newDesc {
			t.Errorf("Описание не обновилось, ожидалось '%s', получено '%s'", newDesc, updated.Description)
		}
	})

	t.Run("Обновление тегов", func(t *testing.T) {
		newTags := []string{"новый", "тег"}
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Tags: &newTags})
		if err != nil {
			t.Fatalf("Ошибка при обновлении тегов: %v", err)
		}
		if len(updated.Tags) != 2 {
			t.Errorf("Ожидалось 2 тега, получено %d", len(updated.Tags))
		}
	})

	t.Run("Обновление только статуса", func(t *testing.T) {
		completed := true
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})
		if err != nil {
			t.Fatalf("Ошибка при обновлении: %v", err)
		}
		if !updated.Completed {
			t.Error("Статус должен был измениться на завершенный")
		}
	})

	t.Run("Пустое описание", func(t *testing.T) {
		empty := ""
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &empty})
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}
	})

	t.Run("Слишком длинное описание", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &longDesc})
		if err == nil {
			t.Error("Ожидалась ошибка при слишком длинном описании")
		}
	})

	t.Run("Несуществующая задача", func(t *testing.T) {
		_, err := tm.UpdateTask(999, UpdateTaskRequest{})
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}
	})
}

func TestDeleteTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Задача для удаления", nil)

	t.Run("Успешное удаление", func(t *testing.T) {
		err := tm.DeleteTask(id)
		if err != nil {
			t.Fatalf("Ошибка при удалении: %v", err)
		}
	})

	t.Run("Удаление несуществующей задачи", func(t *testing.T) {
		err := tm.DeleteTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка при удалении несуществующей задачи")
		}
	})
}

func TestGetTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Тестовая задача", []string{"тест"})

	t.Run("Получение существующей задачи", func(t *testing.T) {
		task, err := tm.GetTask(id)
		if err != nil {
			t.Fatalf("Ошибка при получении задачи: %v", err)
		}
		if task.ID != id {
			t.Errorf("Ожидался ID %d, получен %d", id, task.ID)
		}
		if task.Description != "Тестовая задача" {
			t.Errorf("Ожидалось описание 'Тестовая задача', получено '%s'", task.Description)
		}
		if len(task.Tags) != 1 || task.Tags[0] != "тест" {
			t.Errorf("Ожидался тег 'тест', получено %v", task.Tags)
		}
	})

	t.Run("Получение несуществующей задачи", func(t *testing.T) {
		_, err := tm.GetTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка при получении несуществующей задачи")
		}
	})
}

func TestGetAllTasks(t *testing.T) {
	tm := NewTaskManager()

	t.Run("Пустой список задач", func(t *testing.T) {
		tasks := tm.GetAllTasks()
		if len(tasks) != 0 {
			t.Errorf("Ожидался пустой список, получено %d задач", len(tasks))
		}
	})

	t.Run("Список с задачами", func(t *testing.T) {
		tm.AddTask("Задача 1", []string{"тег1"})
		tm.AddTask("Задача 2", []string{"тег2"})
		tasks := tm.GetAllTasks()
		if len(tasks) != 2 {
			t.Errorf("Ожидалось 2 задачи, получено %d", len(tasks))
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	tm := NewTaskManager()
	var wg sync.WaitGroup
	count := 100

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			_, _ = tm.AddTask("Конкурентная задача", nil)
		}()
	}
	wg.Wait()

	if len(tm.tasks) != count {
		t.Errorf("Ожидалось %d задач, получено %d", count, len(tm.tasks))
	}
}

func TestToggleComplete(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Тестовая задача", nil)

	t.Run("Переключение с false на true", func(t *testing.T) {
		task, err := tm.ToggleComplete(id)
		if err != nil {
			t.Fatalf("Ошибка при переключении статуса: %v", err)
		}
		if !task.Completed {
			t.Error("Ожидалось completed=true после первого переключения")
		}
	})

	t.Run("Переключение с true на false", func(t *testing.T) {
		task, err := tm.ToggleComplete(id)
		if err != nil {
			t.Fatalf("Ошибка при переключении статуса: %v", err)
		}
		if task.Completed {
			t.Error("Ожидалось completed=false после второго переключения")
		}
	})

	t.Run("Несуществующая задача", func(t *testing.T) {
		_, err := tm.ToggleComplete(999)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}
	})

	t.Run("Обновление времени модификации", func(t *testing.T) {
		initialTask, _ := tm.GetTask(id)
		time.Sleep(10 * time.Millisecond)
		
		task, err := tm.ToggleComplete(id)
		if err != nil {
			t.Fatalf("Ошибка при переключении статуса: %v", err)
		}
		
		if !task.UpdatedAt.After(initialTask.UpdatedAt) {
			t.Error("Время обновления должно было измениться")
		}
	})
}

func TestConcurrentToggle(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Конкурентное переключение", nil)
	var wg sync.WaitGroup
	iterations := 100

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_, _ = tm.ToggleComplete(id)
		}()
	}
	wg.Wait()

	task, _ := tm.GetTask(id)
	if task.Completed != (iterations%2 == 1) {
		t.Errorf("Неожиданное состояние задачи после %d переключений", iterations)
	}
}

func TestMetrics(t *testing.T) {
	AddTaskCount.Reset()
	UpdateTaskCount.Reset()
	DeleteTaskCount.Reset()

	tm := NewTaskManager()

	t.Run("Метрики AddTask", func(t *testing.T) {
		_, err := tm.AddTask("Тестовая задача", nil)
		if err != nil {
			t.Fatalf("Ошибка при добавлении задачи: %v", err)
		}

		if got := testutil.ToFloat64(AddTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("AddTaskCount success = %v, want 1", got)
		}

		_, err = tm.AddTask("", nil)
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}

		if got := testutil.ToFloat64(AddTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("AddTaskCount error = %v, want 1", got)
		}
	})

	t.Run("Метрики UpdateTask", func(t *testing.T) {
		id, _ := tm.AddTask("Тестовая задача", nil)

		completed := true
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})
		if err != nil {
			t.Fatalf("Ошибка при обновлении задачи: %v", err)
		}

		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("UpdateTaskCount success = %v, want 1", got)
		}

		_, err = tm.UpdateTask(999, UpdateTaskRequest{})
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}

		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("UpdateTaskCount error = %v, want 1", got)
		}
	})

	t.Run("Метрики DeleteTask", func(t *testing.T) {
		id, _ := tm.AddTask("Тестовая задача", nil)

		err := tm.DeleteTask(id)
		if err != nil {
			t.Fatalf("Ошибка при удалении задачи: %v", err)
		}

		if got := testutil.ToFloat64(DeleteTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("DeleteTaskCount success = %v, want 1", got)
		}

		err = tm.DeleteTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}

		if got := testutil.ToFloat64(DeleteTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("DeleteTaskCount error = %v, want 1", got)
		}
	})

	t.Run("Метрики ToggleComplete", func(t *testing.T) {
		id, _ := tm.AddTask("Тестовая задача для toggle", nil)

		_, err := tm.ToggleComplete(id)
		if err != nil {
			t.Fatalf("Ошибка при переключении статуса: %v", err)
		}

		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("success")); got != 2 {
			t.Errorf("UpdateTaskCount success = %v, want 2", got)
		}

		if got := testutil.CollectAndCount(UpdateTaskDuration); got < 1 {
			t.Errorf("UpdateTaskDuration не был записан")
		}

		_, err = tm.ToggleComplete(999)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}

		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("error")); got != 2 {
			t.Errorf("UpdateTaskCount error = %v, want 2", got)
		}
	})
}

func TestFilterTasks(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Активная задача", nil)
	completedID, _ := tm.AddTask("Выполненная задача", nil)
	tm.ToggleComplete(completedID)

	t.Run("Фильтр Все", func(t *testing.T) {
		tasks := tm.FilterTasks(nil)
		if len(tasks) != 2 {
			t.Errorf("Ожидалось 2 задачи, получено %d", len(tasks))
		}
	})

	t.Run("Фильтр Выполненные", func(t *testing.T) {
		completed := true
		tasks := tm.FilterTasks(&completed)
		if len(tasks) != 1 || !tasks[0].Completed {
			t.Error("Ожидалась 1 выполненная задача")
		}
	})

	t.Run("Фильтр Активные", func(t *testing.T) {
		active := false
		tasks := tm.FilterTasks(&active)
		if len(tasks) != 1 || tasks[0].Completed {
			t.Error("Ожидалась 1 активная задача")
		}
	})
}

func TestFilterByPriority(t *testing.T) {
	tm := NewTaskManager()
	
	lowID, _ := tm.AddTask("Низкий приоритет", nil)
	medID, _ := tm.AddTask("Средний приоритет", nil)
	highID, _ := tm.AddTask("Высокий приоритет", nil)
	
	lowPriority := PriorityLow
	medPriority := PriorityMedium
	highPriority := PriorityHigh
	
	tm.UpdateTask(lowID, UpdateTaskRequest{Priority: &lowPriority})
	tm.UpdateTask(medID, UpdateTaskRequest{Priority: &medPriority})
	tm.UpdateTask(highID, UpdateTaskRequest{Priority: &highPriority})
	
	t.Run("Фильтр по высокому приоритету", func(t *testing.T) {
		tasks := tm.FilterByPriority(PriorityHigh)
		if len(tasks) != 1 {
			t.Fatalf("Ожидалась 1 задача с высоким приоритетом, получено %d", len(tasks))
		}
		if tasks[0].Priority != PriorityHigh {
			t.Errorf("Ожидался приоритет 'high', получен '%s'", tasks[0].Priority)
		}
	})
	
	t.Run("Фильтр по среднему приоритету", func(t *testing.T) {
		tasks := tm.FilterByPriority(PriorityMedium)
		if len(tasks) != 1 {
			t.Fatalf("Ожидалась 1 задача со средним приоритетом, получено %d", len(tasks))
		}
		if tasks[0].Priority != PriorityMedium {
			t.Errorf("Ожидался приоритет 'medium', получен '%s'", tasks[0].Priority)
		}
	})
	
	t.Run("Фильтр по низкому приоритету", func(t *testing.T) {
		tasks := tm.FilterByPriority(PriorityLow)
		if len(tasks) != 1 {
			t.Fatalf("Ожидалась 1 задача с низким приоритетом, получено %d", len(tasks))
		}
		if tasks[0].Priority != PriorityLow {
			t.Errorf("Ожидался приоритет 'low', получен '%s'", tasks[0].Priority)
		}
	})
	
	t.Run("Нет задач с указанным приоритетом", func(t *testing.T) {
		tasks := tm.FilterByPriority("unknown")
		if len(tasks) != 0 {
			t.Errorf("Ожидалось 0 задач, получено %d", len(tasks))
		}
	})
}

func TestFilterByTag(t *testing.T) {
	tm := NewTaskManager()
	
	tm.AddTask("Задача 1", []string{"тег1"})
	tm.AddTask("Задача 2", []string{"тег2"})
	tm.AddTask("Задача 3", []string{"тег1", "тег2"})
	tm.AddTask("Задача 4", []string{"ТеГ1"}) // Тест на регистронезависимость
	
	t.Run("Фильтр по тегу1", func(t *testing.T) {
		tasks := tm.FilterByTag("тег1")
		if len(tasks) != 3 {
			t.Errorf("Ожидалось 3 задачи с тегом 'тег1', получено %d", len(tasks))
		}
	})
	
	t.Run("Фильтр по тегу2", func(t *testing.T) {
		tasks := tm.FilterByTag("тег2")
		if len(tasks) != 2 {
			t.Errorf("Ожидалось 2 задачи с тегом 'тег2', получено %d", len(tasks))
		}
	})
	
	t.Run("Фильтр по несуществующему тегу", func(t *testing.T) {
		tasks := tm.FilterByTag("тег3")
		if len(tasks) != 0 {
			t.Errorf("Ожидалось 0 задач, получено %d", len(tasks))
		}
	})
	
	t.Run("Регистронезависимый поиск", func(t *testing.T) {
		tasks := tm.FilterByTag("ТЕГ1")
		if len(tasks) != 3 {
			t.Errorf("Ожидалось 3 задачи при регистронезависимом поиске, получено %d", len(tasks))
		}
	})
}

func TestGetUpcomingTasks(t *testing.T) {
	tm := NewTaskManager()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	testTasks := []struct {
		desc     string
		dueDate  time.Time
		priority Priority
		completed bool
	}{
		{"Today early", today.Add(2 * time.Hour), PriorityMedium, false},
		{"Today late", today.Add(23 * time.Hour), PriorityHigh, false},
		{"Tomorrow", today.AddDate(0, 0, 1), PriorityHigh, false},
		{"Future task", today.AddDate(0, 0, 3), PriorityLow, false},
		{"Completed task", today.AddDate(0, 0, 2), PriorityMedium, true},
		{"Past task", today.AddDate(0, 0, -1), PriorityHigh, false},
		{"Edge case task", today.AddDate(0, 0, 7), PriorityMedium, false},
		{"No date task", time.Time{}, PriorityLow, false},
		{"Next week task", today.AddDate(0, 0, 8), PriorityLow, false},
	}
	
	for _, tt := range testTasks {
		id, _ := tm.AddTask(tt.desc, nil)
		tm.UpdateTask(id, UpdateTaskRequest{
			DueDate:  &tt.dueDate,
			Priority: &tt.priority,
			Completed: &tt.completed,
		})
	}
	
	tests := []struct {
		name string
		days int
		want []string
	}{
		{
			name: "Today only",
			days: 0,
			want: []string{"Today early", "Today late"},
		},
		{
			name: "Next 3 days",
			days: 3,
			want: []string{"Today early", "Today late", "Tomorrow", "Future task"},
		},
		{
			name: "Next week",
			days: 7,
			want: []string{"Today early", "Today late", "Tomorrow", "Future task", "Edge case task"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.GetUpcomingTasks(tt.days)
			
			if len(got) != len(tt.want) {
				t.Errorf("GetUpcomingTasks() returned %d tasks (%v), want %d (%v)", 
					len(got), getTaskDescriptions(got), len(tt.want), tt.want)
				return
			}
			
			for i := range got {
				if got[i].Description != tt.want[i] {
					t.Errorf("Position %d: got %q, want %q", 
						i, got[i].Description, tt.want[i])
				}
			}
		})
	}
}

func getTaskDescriptions(tasks []Task) []string {
	descs := make([]string, len(tasks))
	for i, task := range tasks {
		descs[i] = task.Description
	}
	return descs
}

func TestGetAllTags(t *testing.T) {
    tm := NewTaskManager()

    // Добавляем задачи с тегами (включая разные регистры)
    tm.AddTask("Задача 1", []string{"тег1", "тег2"})
    tm.AddTask("Задача 2", []string{"ТЕГ2", "тег3"})
    tm.AddTask("Задача 3", []string{"ТеГ1", "тег4"})

    // Получаем все уникальные теги (должны быть нормализованы)
    tags := tm.GetAllTags()

    // Ожидаемые теги (в нижнем регистре)
    expectedTags := []string{"тег1", "тег2", "тег3", "тег4"}
    
    // Проверка количества
    if len(tags) != len(expectedTags) {
        t.Fatalf("Ожидалось %d уникальных тегов, получено %d: %v", 
            len(expectedTags), len(tags), tags)
    }

    // Проверка наличия всех тегов
    tagMap := make(map[string]bool)
    for _, tag := range tags {
        tagMap[tag] = true
    }

    for _, expectedTag := range expectedTags {
        if !tagMap[expectedTag] {
            t.Errorf("Отсутствует ожидаемый тег: %s", expectedTag)
        }
    }

    // Проверка сортировки
    for i := 0; i < len(tags)-1; i++ {
        if tags[i] > tags[i+1] {
            t.Errorf("Теги не отсортированы: %s > %s", tags[i], tags[i+1])
        }
    }
}