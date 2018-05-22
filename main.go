package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var rooms []*Room
var room0 *Room

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Room struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	users []*User `json:"-"`
	bus   *Bus    `json:"-"`
}

func newRoom(name string) *Room {
	var b *Bus
	var r = &Room{Name: name, Count: 0, users: []*User{}, bus: b}
	r.bus = NewBus(r)
	go r.bus.Run()

	return r
}

type User struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func newUser(name string) *User {
	user := &User{Name: name, Id: name}
	return user
}

func (u *User) Command(command string) {
	log.Println("Command: '", command, "' received by user: ", u.Name)
}

func (u *User) GetState() string {
	return "Game state for User: " + u.Name
}

type Bus struct {
	register  chan *websocket.Conn
	broadcast chan []byte
	clients   map[*websocket.Conn]bool
	room      *Room
}

func NewBus(room *Room) *Bus {
	return &Bus{
		register:  make(chan *websocket.Conn),
		broadcast: make(chan []byte),
		clients:   make(map[*websocket.Conn]bool),
		room:      room,
	}
}

func (b *Bus) Run() {
	for {
		select {
		case message := <-b.broadcast:
			for client := range b.clients {
				for i := 0; i < 5; i++ {
					<-time.After(5 * time.Second)
					log.Println("broadcast:", message)
					w, err := client.NextWriter(websocket.TextMessage)
					if err != nil {
						b.room.Count--
						delete(b.clients, client)
						continue
					}

					w.Write(message)
				}
			}
		case client := <-b.register:
			log.Println("User registered")
			b.clients[client] = true
			b.room.Count++

			m, err := json.Marshal(rooms)
			if err != nil {
				log.Println(err.Error)
			}

			room0.bus.broadcast <- m

		}
	}
}

func runRoom(room *Room) {

	for {
		<-time.After(5 * time.Second)
		if room.Count > 0 {

			m, err := json.Marshal(room)
			if err != nil {
				log.Println(err.Error)
			}

			log.Println(room, "   ", m)
			os.Stdout.Write(m)

			room.bus.broadcast <- m
		}
	}
}

func main() {

	room0 = newRoom("0 room")
	var room1 *Room = newRoom("1 room")
	var room2 *Room = newRoom("2 room")
	var room3 *Room = newRoom("3 room")

	rooms = []*Room{room1, room2, room3}

	for _, room := range rooms {
		go runRoom(room)
	}

	go room0.bus.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
		}

		room0.bus.register <- ws
	})

	http.ListenAndServe(":8081", nil)

}
