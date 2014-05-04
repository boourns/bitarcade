package main

import (
	"github.com/boourns/incredible/scene"
	"github.com/boourns/mux"
	"github.com/gorilla/websocket"
	"time"
	"net/http"
	"log"
	"html/template"
	"math/rand"
)

var players *mux.Mux
var world   *scene.Scene

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

var timeToNextBall = 1000

func eventHandler(input interface{}) []interface{} {
	output := make([]interface{}, 0)

	timeToNextBall--
	if timeToNextBall <= 0 {
		world.AddBall(rand.Int() % 600, rand.Int() % 300 + 300)
		timeToNextBall = 1000
	}

	world.Step(1.0 / 60.0)
	state, _ := world.Render()
	output = append(output, string(state))
	return output
}

func main() {
	world = scene.New()
	for i := 0; i < 20; i++ {
		world.AddBall(rand.Int() % 600, rand.Int() % 300 + 300)
	}

	players = mux.New(eventHandler)

	timer := make(chan interface{}, 1)
	go func(timer chan interface{}) {
		for true {
			time.Sleep(17 * time.Millisecond)
			timer <- "tick"
		}
	}(timer)
	players.AddInput(timer)

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

	stateReceiver := make(chan interface{}, 1)
	players.AddOutput(stateReceiver)
	defer players.RemoveOutput(stateReceiver)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	for true {
		select {
		case payload := <- stateReceiver:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			ws.WriteMessage(websocket.TextMessage, []byte(mux.String(payload)))
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

