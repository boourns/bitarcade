package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"
)

type Event struct {
	Player      int
	Type        int
	Code        int
	PlayerToken string
	Return      chan string
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	log.Printf("Player connecting")

	params := r.URL.Query()
	if len(params["game"]) != 1 {
		http.Error(w, "error getting game token", 500)
		log.Printf("error getting game token")
	}
	gameToken := params["game"][0]

	playerToken, err := getPlayerToken(w, r, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), 500)
		log.Printf("error getting player token")
		return
	}

	log.Printf("Game token %s, player token %s", gameToken, playerToken)

	// auth to matchmaker, and get game pointer
	returnChan := make(chan *MatchMakerEvent, 0)
	Matcher.Events <- &MatchMakerEvent{
		Type:        GET_GAME,
		PlayerToken: playerToken,
		GameToken:   gameToken,
		Return:      returnChan,
	}

	game := (<-returnChan).Game
	if game == nil {
		http.Error(w, "Matchmaker: Not allowed", 401)
		log.Printf("MatchMaker: Not Allowed")
		return
	}

	// connect to game
	receiver := make(chan string, 0)
	game.Events <- &Event{Type: CONNECT, Return: receiver}
	response := <-receiver
	if response == "" {
		http.Error(w, "Game: Not allowed", 401)
		log.Printf("Game: Not allowed")
		return
	}

	playerId, _ := strconv.ParseInt(response, 10, 32)

	// leave game (maybe that's crazy) if websocket disconnects, TODO: go back to game disconnect
	defer func(playerToken string) {
		Matcher.Events <- &MatchMakerEvent{
			Type:        LEAVE_GAME,
			PlayerToken: playerToken,
			GameToken:   gameToken,
		}
	}(playerToken)

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
			ev := &Event{Player: int(playerId), Code: input.Code}
			if input.Down {
				ev.Type = KEYDOWN
			} else {
				ev.Type = KEYUP
			}
			game.Events <- ev
		}
	}
}

var homeTempl = template.Must(template.ParseFiles("home.html"))

func serveGame(w http.ResponseWriter, r *http.Request) {
	if path.Dir(r.URL.Path) != "/game" {
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

func getPlayerToken(w http.ResponseWriter, r *http.Request, saveSession bool) (token string, err error) {
	session, err := store.Get(r, "incredible")
	if err != nil {
		panic(err)
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7 * 365,
		HttpOnly: true,
	}

	if token, ok := session.Values["token"]; !ok {
		if saveSession {
			token = Token()
			session.Values["token"] = token
			session.Save(r, w)
		} else {
			err = fmt.Errorf("Session has no token")
		}
	}

	return
}