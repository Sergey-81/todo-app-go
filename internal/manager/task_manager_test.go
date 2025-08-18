package manager

import (
	"testing"
	"time"
	"todo-app/internal/models"
)

func TestAddTask(t *testing.T) {
	tm := NewTaskManager()

	id, err := tm.AddTask("Test task", []string{"test"})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected ID 1, got %d", id)
	}

	tasks := tm.GetAllTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Description != "Test task" {
		t.Errorf("Expected description 'Test task', got '%s'", tasks[0].Description)
	}
}

func TestAddTaskEmptyDescription(t *testing.T) {
	tm := NewTaskManager()

	_, err := tm.AddTask("", nil)
	if err == nil {
		t.Error("Expected error for empty description, got nil")
	}
}

func TestGetTask(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Test task", nil)

	task, err := tm.GetTask(1)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.Description != "Test task" {
		t.Errorf("Expected description 'Test task', got '%s'", task.Description)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	tm := NewTaskManager()

	_, err := tm.GetTask(1)
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestUpdateTask(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Original task", nil)

	newDesc := "Updated task"
	updatedTask, err := tm.UpdateTask(1, models.UpdateTaskRequest{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	if updatedTask.Description != "Updated task" {
		t.Errorf("Expected description 'Updated task', got '%s'", updatedTask.Description)
	}
}

func TestDeleteTask(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Task to delete", nil)

	err := tm.DeleteTask(1)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	tasks := tm.GetAllTasks()
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after deletion, got %d", len(tasks))
	}
}

func TestToggleComplete(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Test task", nil)

	// First toggle (should mark as completed)
	task, err := tm.ToggleComplete(1)
	if err != nil {
		t.Fatalf("ToggleComplete failed: %v", err)
	}
	if !task.Completed {
		t.Error("Expected task to be completed after first toggle")
	}

	// Second toggle (should mark as pending)
	task, err = tm.ToggleComplete(1)
	if err != nil {
		t.Fatalf("ToggleComplete failed: %v", err)
	}
	if task.Completed {
		t.Error("Expected task to be pending after second toggle")
	}
}

func TestFilterTasks(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Task 1", nil)
	tm.AddTask("Task 2", nil)
	tm.ToggleComplete(1)

	// Test completed filter
	completed := true
	completedTasks := tm.FilterTasks(&completed)
	if len(completedTasks) != 1 {
		t.Errorf("Expected 1 completed task, got %d", len(completedTasks))
	}

	// Test pending filter
	pending := false
	pendingTasks := tm.FilterTasks(&pending)
	if len(pendingTasks) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pendingTasks))
	}

	// Test all tasks
	allTasks := tm.FilterTasks(nil)
	if len(allTasks) != 2 {
		t.Errorf("Expected 2 tasks total, got %d", len(allTasks))
	}
}

func TestFilterByTag(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Task with tag", []string{"important"})
	tm.AddTask("Task without tag", nil)

	taggedTasks := tm.FilterByTag("important")
	if len(taggedTasks) != 1 {
		t.Errorf("Expected 1 task with tag, got %d", len(taggedTasks))
	}
}

func TestGetUpcomingTasks(t *testing.T) {
	tm := NewTaskManager()
	
	// Add task due tomorrow
	id, _ := tm.AddTask("Task due tomorrow", nil)
	dueDate := time.Now().Add(24 * time.Hour)
	_, err := tm.UpdateTask(id, models.UpdateTaskRequest{DueDate: &dueDate})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Add completed task
	tm.AddTask("Completed task", nil)
	tm.ToggleComplete(2)

	// Add task due in 8 days (should not be included)
	tm.AddTask("Task due in 8 days", nil)
	farDueDate := time.Now().Add(8 * 24 * time.Hour)
	tm.UpdateTask(3, models.UpdateTaskRequest{DueDate: &farDueDate})

	upcomingTasks := tm.GetUpcomingTasks(7) // 7 days
	if len(upcomingTasks) != 1 {
		t.Errorf("Expected 1 upcoming task, got %d", len(upcomingTasks))
	}

	if upcomingTasks[0].Description != "Task due tomorrow" {
		t.Errorf("Expected task 'Task due tomorrow', got '%s'", upcomingTasks[0].Description)
	}
}

func TestFilterByPriority(t *testing.T) {
	tm := NewTaskManager()
	
	// Создаем переменные для каждого приоритета
	low := models.PriorityLow
	medium := models.PriorityMedium
	high := models.PriorityHigh

	// Add tasks with different priorities
	tm.AddTask("Low priority", nil)
	tm.AddTask("Medium priority", nil)
	tm.AddTask("High priority", nil)
	
	// Update priorities (теперь используем переменные вместо &constants)
	tm.UpdateTask(1, models.UpdateTaskRequest{Priority: &low})
	tm.UpdateTask(2, models.UpdateTaskRequest{Priority: &medium})
	tm.UpdateTask(3, models.UpdateTaskRequest{Priority: &high})

	// Test low priority filter
	lowTasks := tm.FilterByPriority(models.PriorityLow)
	if len(lowTasks) != 1 {
		t.Errorf("Expected 1 low priority task, got %d", len(lowTasks))
	} else if lowTasks[0].Description != "Low priority" {
		t.Errorf("Expected 'Low priority' task, got '%s'", lowTasks[0].Description)
	}

	// Test medium priority filter
	mediumTasks := tm.FilterByPriority(models.PriorityMedium)
	if len(mediumTasks) != 1 {
		t.Errorf("Expected 1 medium priority task, got %d", len(mediumTasks))
	} else if mediumTasks[0].Description != "Medium priority" {
		t.Errorf("Expected 'Medium priority' task, got '%s'", mediumTasks[0].Description)
	}

	// Test high priority filter
	highTasks := tm.FilterByPriority(models.PriorityHigh)
	if len(highTasks) != 1 {
		t.Errorf("Expected 1 high priority task, got %d", len(highTasks))
	} else if highTasks[0].Description != "High priority" {
		t.Errorf("Expected 'High priority' task, got '%s'", highTasks[0].Description)
	}
}

func TestUpdateTaskEmptyDescription(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Original task", nil)

	emptyDesc := ""
	_, err := tm.UpdateTask(1, models.UpdateTaskRequest{
		Description: &emptyDesc,
	})
	if err == nil {
		t.Error("Expected error for empty description, got nil")
	}
}

func TestUpdateTaskNotFound(t *testing.T) {
	tm := NewTaskManager()

	newDesc := "New description"
	_, err := tm.UpdateTask(999, models.UpdateTaskRequest{
		Description: &newDesc,
	})
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	tm := NewTaskManager()

	err := tm.DeleteTask(999)
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestToggleCompleteNotFound(t *testing.T) {
	tm := NewTaskManager()

	_, err := tm.ToggleComplete(999)
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestGetUpcomingTasksEmpty(t *testing.T) {
	tm := NewTaskManager()

	upcomingTasks := tm.GetUpcomingTasks(7)
	if len(upcomingTasks) != 0 {
		t.Errorf("Expected 0 upcoming tasks, got %d", len(upcomingTasks))
	}
}

func TestFilterByPriorityEmpty(t *testing.T) {
	tm := NewTaskManager()
	
	// Не добавляем задач
	emptyTasks := tm.FilterByPriority(models.PriorityHigh)
	if len(emptyTasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(emptyTasks))
	}
}

func TestGetUpcomingTasksEdgeCases(t *testing.T) {
	tm := NewTaskManager()
	
	// 1. Тест с задачей без due date (должна быть пропущена)
	idNoDate, _ := tm.AddTask("Task without due date", nil)
	taskNoDate, _ := tm.GetTask(idNoDate)
	if taskNoDate.DueDate.IsZero() == false {
		t.Error("New task should have zero due date by default")
	}

	// 2. Тест с завершенной задачей (должна быть пропущена)
	idCompleted, _ := tm.AddTask("Completed task", nil)
	completed := true
	tm.UpdateTask(idCompleted, models.UpdateTaskRequest{Completed: &completed})

	// 3. Тест с задачей, у которой due date сегодня
	idToday, _ := tm.AddTask("Task due today", nil)
	today := time.Now().Truncate(24 * time.Hour) // Начало дня
	tm.UpdateTask(idToday, models.UpdateTaskRequest{DueDate: &today})

	// 4. Тест с задачей, у которой due date ровно через 7 дней
	idExact7Days, _ := tm.AddTask("Task due in exactly 7 days", nil)
	exact7Days := today.Add(7 * 24 * time.Hour)
	tm.UpdateTask(idExact7Days, models.UpdateTaskRequest{DueDate: &exact7Days})

	upcomingTasks := tm.GetUpcomingTasks(7)
	
	// Должны попасть только задачи с due date сегодня и через 7 дней
	if len(upcomingTasks) != 2 {
		t.Errorf("Expected 2 upcoming tasks, got %d", len(upcomingTasks))
	}
	
	// Проверяем порядок сортировки (должны идти от ближайшей к дальней)
	if !upcomingTasks[0].DueDate.Equal(today) || !upcomingTasks[1].DueDate.Equal(exact7Days) {
		t.Error("Tasks should be sorted by due date ascending")
	}
}

func TestUpdateTaskTags(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Original task", []string{"old1", "old2"})

	newTags := []string{"new1", "new2"}
	updatedTask, err := tm.UpdateTask(1, models.UpdateTaskRequest{
		Tags: &newTags,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	if len(updatedTask.Tags) != 2 || updatedTask.Tags[0] != "new1" || updatedTask.Tags[1] != "new2" {
		t.Errorf("Tags were not updated correctly, got %v", updatedTask.Tags)
	}
}

func TestUpdateMultipleFields(t *testing.T) {
	tm := NewTaskManager()
	tm.AddTask("Original task", nil)

	newDesc := "Updated description"
	newPriority := models.PriorityHigh
	completed := true
	newTags := []string{"important"}
	
	updatedTask, err := tm.UpdateTask(1, models.UpdateTaskRequest{
		Description: &newDesc,
		Priority:    &newPriority,
		Completed:   &completed,
		Tags:        &newTags,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Проверяем все обновленные поля
	if updatedTask.Description != newDesc {
		t.Errorf("Description not updated, got '%s'", updatedTask.Description)
	}
	if updatedTask.Priority != newPriority {
		t.Errorf("Priority not updated, got '%s'", updatedTask.Priority)
	}
	if !updatedTask.Completed {
		t.Error("Task should be completed")
	}
	if len(updatedTask.Tags) != 1 || updatedTask.Tags[0] != "important" {
		t.Errorf("Tags not updated correctly, got %v", updatedTask.Tags)
	}
}