package manager_test

import (
    "testing"
    "todo-app/internal/manager"
)

func TestNewTaskManager(t *testing.T) {
    tm := manager.New()
    if tm == nil {
        t.Fatal("Менеджер не создан")
    }
}