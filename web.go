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

var events chan Event

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	log.Printf("Player connecting")

	receiver := make(chan string, 0)
	events <- Event{Type: JOIN, Return: receiver}
	response := <-receiver

	playerId, _ := strconv.ParseInt(response, 10, 32)

	defer func(playerId int64) {
		events <- Event{
			Type:   QUIT,
			Player: int(playerId),
		}
	}(playerId)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	reader := make(chan InputEvent, 0)

	ws.SetReadLimit(maxMessageSize)
	//	ws.SetReadDeadline(time.Now().Add(pongWait))
	//	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	go func(reader chan InputEvent) {
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				log.Printf("error reading from websocket:%s", err)
				ws.Close()
				break
			}
			var ev InputEvent
			if string(message) != "PONG" {
				err = json.Unmarshal(message, &ev)
				if err != nil {
					fmt.Printf("failed to unmarshal input: %s", err)
				}
				reader <- ev
			}
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
