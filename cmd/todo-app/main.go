package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"todo-app/internal/manager"
	"todo-app/internal/models"
)

const (
	storageFileJSON = "tasks.json"
	storageFileCSV  = "tasks.csv"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	tm := manager.NewTaskManager()
	if err := loadInitialTasks(tm); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "add":
		handleAddCommand(tm)
	case "list":
		handleListCommand(tm)
	case "complete":
		handleCompleteCommand(tm)
	case "delete":
		handleDeleteCommand(tm)
	case "export":
		handleExportCommand(tm)
	case "load":
		handleLoadCommand(tm)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func loadInitialTasks(tm *manager.TaskManager) error {
	// Попробуем загрузить из JSON
	tasks, err := models.LoadJSON(storageFileJSON)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		// Если в JSON нет задач, попробуем CSV
		tasks, err = models.LoadCSV(storageFileCSV)
		if err != nil {
			return err
		}
	}

	for _, task := range tasks {
		// В реальном приложении нужно обновить nextID в TaskManager
		// Здесь упрощённая логика для примера
		_, err := tm.AddTask(task.Description, task.Tags)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleAddCommand(tm *manager.TaskManager) {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	desc := addCmd.String("desc", "", "Task description")
	tags := addCmd.String("tags", "", "Comma-separated list of tags")
	addCmd.Parse(os.Args[2:])

	if *desc == "" {
		fmt.Fprintln(os.Stderr, "Error: --desc is required")
		os.Exit(1)
	}

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}

	id, err := tm.AddTask(*desc, tagList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding task: %v\n", err)
		os.Exit(1)
	}

	if err := saveTasks(tm); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving tasks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added task with ID %d\n", id)
}

func handleListCommand(tm *manager.TaskManager) {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	filter := listCmd.String("filter", "all", "Filter tasks (all|completed|pending)")
	listCmd.Parse(os.Args[2:])

	var completed *bool
	switch *filter {
	case "completed":
		val := true
		completed = &val
	case "pending":
		val := false
		completed = &val
	}

	tasks := tm.FilterTasks(completed)
	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return
	}

	for _, task := range tasks {
		status := "Pending"
		if task.Completed {
			status = "Completed"
		}
		fmt.Printf("%d: %s [%s]\n", task.ID, task.Description, status)
	}
}

func handleCompleteCommand(tm *manager.TaskManager) {
	completeCmd := flag.NewFlagSet("complete", flag.ExitOnError)
	id := completeCmd.Int("id", 0, "Task ID to complete")
	completeCmd.Parse(os.Args[2:])

	if *id == 0 {
		fmt.Fprintln(os.Stderr, "Error: --id is required")
		os.Exit(1)
	}

	_, err := tm.ToggleComplete(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error completing task: %v\n", err)
		os.Exit(1)
	}

	if err := saveTasks(tm); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving tasks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task %d marked as completed\n", *id)
}

func handleDeleteCommand(tm *manager.TaskManager) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	id := deleteCmd.Int("id", 0, "Task ID to delete")
	deleteCmd.Parse(os.Args[2:])

	if *id == 0 {
		fmt.Fprintln(os.Stderr, "Error: --id is required")
		os.Exit(1)
	}

	err := tm.DeleteTask(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting task: %v\n", err)
		os.Exit(1)
	}

	if err := saveTasks(tm); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving tasks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task %d deleted\n", *id)
}

func handleExportCommand(tm *manager.TaskManager) {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	format := exportCmd.String("format", "json", "Export format (json|csv)")
	outFile := exportCmd.String("out", "", "Output file path")
	exportCmd.Parse(os.Args[2:])

	if *outFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --out is required")
		os.Exit(1)
	}

	tasks := tm.GetAllTasks()
	var err error

	switch *format {
	case "json":
		err = models.SaveJSON(*outFile, tasks)
	case "csv":
		err = models.SaveCSV(*outFile, tasks)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %s\n", *format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting tasks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tasks exported to %s in %s format\n", *outFile, *format)
}

func handleLoadCommand(tm *manager.TaskManager) {
	loadCmd := flag.NewFlagSet("load", flag.ExitOnError)
	file := loadCmd.String("file", "", "File to load tasks from")
	loadCmd.Parse(os.Args[2:])

	if *file == "" {
		fmt.Fprintln(os.Stderr, "Error: --file is required")
		os.Exit(1)
	}

	var tasks []models.Task
	var err error

	if strings.HasSuffix(*file, ".json") {
		tasks, err = models.LoadJSON(*file)
	} else if strings.HasSuffix(*file, ".csv") {
		tasks, err = models.LoadCSV(*file)
	} else {
		fmt.Fprintln(os.Stderr, "Error: unsupported file format, use .json or .csv")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	// Очищаем текущие задачи
	for _, task := range tm.GetAllTasks() {
		_ = tm.DeleteTask(task.ID)
	}

	// Добавляем загруженные задачи
	for _, task := range tasks {
		_, err := tm.AddTask(task.Description, task.Tags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding task: %v\n", err)
			os.Exit(1)
		}
	}

	if err := saveTasks(tm); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving tasks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d tasks from %s\n", len(tasks), *file)
}

func saveTasks(tm *manager.TaskManager) error {
	tasks := tm.GetAllTasks()
	if err := models.SaveJSON(storageFileJSON, tasks); err != nil {
		return err
	}
	return models.SaveCSV(storageFileCSV, tasks)
}

func printHelp() {
	fmt.Println(`Usage: todo <command> [flags]

Commands:
  add     --desc="..." [--tags="tag1,tag2"]  Add new task
  list    [--filter=all|completed|pending]   List tasks
  complete --id=ID                           Mark task as completed
  delete  --id=ID                            Delete task
  export  --format=json|csv --out=FILE       Export tasks
  load    --file=FILE                        Load tasks from file

Storage:
  Tasks are automatically saved to tasks.json and tasks.csv in the current directory.
  On startup, the application tries to load tasks from these files.`)
}
