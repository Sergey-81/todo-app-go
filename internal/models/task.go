package models

import (
	"encoding/csv"
	"encoding/json"
	//"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Priority    Priority  `json:"priority"`
	DueDate     time.Time `json:"due_date"`
	Tags        []string  `json:"tags"`
}

type UpdateTaskRequest struct {
	Description *string    `json:"description,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Tags        *[]string  `json:"tags,omitempty"`
}

func SaveJSON(filename string, tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}

func LoadJSON(filename string) ([]Task, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return []Task{}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func SaveCSV(filename string, tasks []Task) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"id", "description", "completed", "created_at", "updated_at", "priority", "due_date", "tags"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, task := range tasks {
		record := []string{
			strconv.Itoa(task.ID),
			task.Description,
			strconv.FormatBool(task.Completed),
			task.CreatedAt.Format(time.RFC3339),
			task.UpdatedAt.Format(time.RFC3339),
			string(task.Priority),
			task.DueDate.Format(time.RFC3339),
			strings.Join(task.Tags, ","),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func LoadCSV(filename string) ([]Task, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 8 // Количество полей в заголовке

	// Пропускаем заголовок
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var tasks []Task
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		id, _ := strconv.Atoi(record[0])
		completed, _ := strconv.ParseBool(record[2])
		createdAt, _ := time.Parse(time.RFC3339, record[3])
		updatedAt, _ := time.Parse(time.RFC3339, record[4])
		dueDate, _ := time.Parse(time.RFC3339, record[6])
		tags := strings.Split(record[7], ",")
		if len(tags) == 1 && tags[0] == "" {
			tags = []string{}
		}

		task := Task{
			ID:          id,
			Description: record[1],
			Completed:   completed,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			Priority:    Priority(record[5]),
			DueDate:     dueDate,
			Tags:        tags,
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}