package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("static/*.html")),
	}
}

type StatusType int

const (
	Todo StatusType = iota
	Doing
	Done
)

type Task struct {
	Id          int64
	Name        string
	Description string
	Status      StatusType
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
		CREATE TABLE IF NOT EXISTS tasks (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT,
			"description" TEXT,
			"status" INTEGER
		);
	`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database initialized successfully")
}

func getTask(id int) Task {
	query := "SELECT * FROM tasks WHERE ID = ?"
	row := db.QueryRow(query, id)

	var task = Task{}
	err := row.Scan(&task.Id, &task.Name, &task.Description, &task.Status)
	if err != nil {
		fmt.Println("Could not find task with that id")
	}

	return task
}

func updateStatus(id int) error {
	query := "UPDATE tasks SET status = ? WHERE id = ?"

	task := getTask(id)
	status := task.Status + 1

	_, err := db.Exec(query, status, id)
	return err
}

func deleteTask(id int) error {
	query := "DELETE FROM tasks WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

func addTask(name string, description string) Task {
	query := "INSERT INTO tasks (name, description, status) VALUES (?, ?, ?)"
	result, _ := db.Exec(query, name, description, 0)

	id, _ := result.LastInsertId()
	task := Task{
		Id:          id,
		Name:        name,
		Description: description,
		Status:      StatusType(0),
	}

	return task
}

func getAllTasks() []Task {
	query := "SELECT * FROM tasks"
	rows, err := db.Query(query)
	if err != nil {
		return nil
	}

	var tasks []Task
	for rows.Next() {
		var id int64
		var status int
		var name, description string

		err = rows.Scan(&id, &name, &description, &status)
		if err != nil {
			return nil
		}

		tasks = append(tasks, Task{
			Id:          id,
			Name:        name,
			Description: description,
			Status:      StatusType(status),
		})
	}
	return tasks
}

type PageData struct {
	TodoTasks  []Task
	DoingTasks []Task
	DoneTasks  []Task
}

func main() {
	initDB()
	defer db.Close()

	e := echo.New()
	e.Use(middleware.Logger())

	e.Static("/static", "static")
	e.Renderer = newTemplate()
	e.GET("/", func(c echo.Context) error {
		data := categorizeData(getAllTasks())

		return c.Render(200, "index", data)
	})

	e.POST("/addtask", func(c echo.Context) error {
		name := c.FormValue("name")
		description := c.FormValue("description")

		task := addTask(name, description)

		return c.Render(200, "task", task)
	})

	e.PUT("/changeStatus/:id", func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.String(400, "Invalid id")
		}

		err = updateStatus(id)
		if err != nil {
			return c.String(400, "Could not update status")
		}

		data := categorizeData(getAllTasks())
		return c.Render(200, "tasks", data)
	})

	e.DELETE("/deleteTask/:id", func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.String(400, "invalid id")
		}

		err = deleteTask(id)
		if err != nil {
			return c.String(400, "Could not find task")
		}

		return c.NoContent(200)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func categorizeData(tasks []Task) PageData {
	var todoTasks []Task
	var doingTasks []Task
	var doneTasks []Task

	for _, task := range tasks {
		switch task.Status {
		case 0:
			todoTasks = append(todoTasks, task)
		case 1:
			doingTasks = append(doingTasks, task)
		case 2:
			doneTasks = append(doneTasks, task)
		}
	}

	data := PageData{
		TodoTasks:  todoTasks,
		DoingTasks: doingTasks,
		DoneTasks:  doneTasks,
	}
	return data
}
