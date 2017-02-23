package main


import (
	"os"
	"strconv"
	"strings"
	"fmt"
	"encoding/json"
	"errors"
	"gopkg.in/redis.v5"
)


const REDIS_DB_KEY string = "touilledo_db"


type TodoEntry struct {
	IsDone bool `json:"done"`
	Label string `json:"label"`
}


func (todo *TodoEntry) done() {
	todo.IsDone = !todo.IsDone
}


type Todos struct {
	Items []TodoEntry `json:"items"`
}


func (todos *Todos) new(label string) {
	todo := TodoEntry{IsDone: false, Label: label}
	todos.Items = append(todos.Items, todo)
}


func (todos *Todos) del(id int) {
	todos.Items = append(todos.Items[:id], todos.Items[id+1:]...)
}


func (todos *Todos) get(id int) (*TodoEntry, error) {
	if id >= len(todos.Items) {
		return nil, errors.New("Bad id")
	}
	return &todos.Items[id], nil
}


func (todos *Todos) dump() (s string) {
	for i := range todos.Items {
		todo := todos.Items[i]
		if todo.IsDone {
			s += fmt.Sprintf("[%d] ", i)
			// Strikethrough style
			for _, c := range todo.Label {
				s += string(c) + "\u0336"
			}
			s += "\n"
		} else {
			s += fmt.Sprintf("[%d] %s\n", i, todo.Label)
		}
	}
	return
}


func clearTodos(client *redis.Client) {
    if err := client.Del(REDIS_DB_KEY).Err(); err != nil {
    	panic("Cannot clear redis base : " + err.Error())
    }
}


func loadTodos(client *redis.Client) *Todos {
    todos := new(Todos)
    val, err := client.Get(REDIS_DB_KEY).Result()
    if err != nil {
    	panic("Cannot retrieve redis base : " + err.Error())
    }
    if err = json.Unmarshal([]byte(val), todos); err != nil {
    	panic("Cannot decode todo list : " + err.Error())
    }
    return todos
}


func saveTodos(client *redis.Client, todos *Todos) {
	out, _ := json.Marshal(todos)
    if err := client.Set(REDIS_DB_KEY, out, 0).Err(); err != nil {
    	panic("Cannot save in redis base : " + err.Error())
    }
}


func getAndCheckIdFromStr(todos *Todos, idstr string) (id int, err error) {
	if id, err = strconv.Atoi(idstr); err != nil {
		return
	}
	if _, err = todos.get(id); err != nil {
		return
	}
	return
}


func main() {
	redisURL := os.Getenv("TOUILLEDO_URL")
	options, _ := redis.ParseURL(redisURL)
    client := redis.NewClient(options)

	todos := loadTodos(client)
	if len(os.Args) < 2 {
		fmt.Print(todos.dump())
		return
	}
	switch os.Args[1] {
	case "new":
		fallthrough
	case "n":
		todos.new(strings.Join(os.Args[2:], " "))
		saveTodos(client, todos)
	case "done":
		fallthrough
	case "d":
		id, err := getAndCheckIdFromStr(todos, os.Args[2])
		if err != nil {
			fmt.Println("Bad todo id")
			return
		}
		todo, _ := todos.get(id)
		todo.done()
		saveTodos(client, todos)
	case "del":
		id, err := getAndCheckIdFromStr(todos, os.Args[2])
		if err != nil {
			fmt.Println("Bad todo id")
			return
		}
		todos.del(id)
		saveTodos(client, todos)
	case "clear":
		var todos Todos
		saveTodos(client, &todos)
	default:
		fmt.Println("usage : %s [done|del|clear] {todo}", os.Args[0])
	}
}
