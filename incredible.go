package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Player struct {
	Position Position
	Keys     map[int]bool
	State    int
	Return   chan string
}

type Position struct {
	X         int
	Y         int
	Direction float64
	Size      int
	Speed     float64
}

var PlayerCount = 0

type Environment struct {
	Players map[int]Player
	Bullets []Position
}

type SerializedEnvironment struct {
	Players []Position
	Bullets []Position
}

var World Environment

const (
	JOIN    = iota
	KEYDOWN = iota
	KEYUP   = iota
	QUIT    = iota
	TIMER   = iota
)

const (
	GAMEOVER = iota
	PLAYING  = iota
)

type Event struct {
	Player int
	Type   int
	Code   int
	Return chan string
}

type InputEvent struct {
	Code int
	Down bool
}

// boring consts for websocket
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func eventHandler(events chan Event) {
	for true {
		select {
		case input := <-events:
			switch input.Type {
			case KEYUP:
				fmt.Printf("Player %d key %d up", input.Player, input.Code)
				World.Players[input.Player].Keys[input.Code] = false
			case KEYDOWN:
				fmt.Printf("Player %d key %d down", input.Player, input.Code)
				World.Players[input.Player].Keys[input.Code] = true
			case JOIN:
				newPlayer := PlayerCount
				PlayerCount++
				fmt.Printf("Player %d Joined\n", newPlayer)
				World.Players[newPlayer] = Player{
					State:  GAMEOVER,
					Return: input.Return,
					Keys:   make(map[int]bool, 0),
				}
				input.Return <- fmt.Sprintf("%d", newPlayer)
			case QUIT:
				delete(World.Players, input.Player)
			case TIMER:
				data := SerializedEnvironment{
					Bullets: World.Bullets,
				}
				for _, v := range World.Players {
					data.Players = append(data.Players, v.Position)
				}
				state, err := json.Marshal(data)
				if err != nil {
					fmt.Printf("Error marshalling world: %v", err)
				}

				for _, v := range World.Players {
					v.Return <- string(state)
				}
			}
		}
	}
}

var events chan Event

func main() {
	events = make(chan Event, 0)
	World.Players = make(map[int]Player)

	go eventHandler(events)

	go func(timer chan Event) {
		for true {
			time.Sleep(1700 * time.Millisecond)
			timer <- Event{Type: TIMER}
		}
	}(events)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", playerHandler)
	http.ListenAndServe("0.0.0.0:8080", nil)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	receiver := make(chan string, 0)
	events <- Event{Type: JOIN, Return: receiver}
	response := <-receiver

	playerId, _ := strconv.ParseInt(response, 10, 32)

	defer func() {
		events <- Event{
			Type:   QUIT,
			Player: int(playerId),
		}
	}()

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	reader := make(chan InputEvent, 0)

	ws.SetReadLimit(maxMessageSize)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	go func(reader chan InputEvent) {
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				ws.Close()
				break
			}
			var ev InputEvent
			err = json.Unmarshal(message, &ev)
			if err != nil {
				fmt.Printf("failed to unmarshal input: %s", err)
			}
			reader <- ev
		}
	}(reader)

	for true {
		select {
		case payload := <-receiver:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			ws.WriteMessage(websocket.TextMessage, []byte(payload))
		case input := <-reader:
			ev := Event{Player: int(playerId), Code: input.Code}
			if input.Down {
				ev.Type = KEYDOWN
			} else {
				ev.Type = KEYUP
			}
			events <- ev
		}
	}
}

var homeTempl = template.Must(template.ParseFiles("home.html"))

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, r.Host)
}
