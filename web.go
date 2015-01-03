package main

import (
	"encoding/json"
	"fmt"
	"github.com/boourns/bitarcade/game"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"
)

type InputEvent struct {
	Code       int
	Down       bool
	Disconnect bool
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

func getGame(eventType int, playerToken string, gameToken string) game.Game {
	// auth to matchmaker, and get game pointer
	returnChan := make(chan *MatchMakerEvent, 0)
	Matcher.Events <- &MatchMakerEvent{
		Type:        eventType,
		PlayerToken: playerToken,
		GameToken:   gameToken,
		Return:      returnChan,
	}

	return (<-returnChan).Game
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	log.Printf("Player websocket connecting")

	params := r.URL.Query()
	if len(params["game"]) != 1 {
		http.Error(w, "error getting game token", 500)
		log.Printf("error getting game token")
	}
	gameToken := params["game"][0]

	playerToken, err := getPlayerToken(w, r, false)
	log.Printf("Player Token = %s", playerToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), 500)
		log.Printf("error getting player token")
		return
	}

	log.Printf("Game token %s, player token %s", gameToken, playerToken)
	g := getGame(JOIN_GAME, playerToken, gameToken)

	if g == nil {
		http.Error(w, "Matchmaker: Not allowed", 401)
		log.Printf("MatchMaker: Not Allowed")
		return
	}

	// connect to game
	receiver := make(chan string, 1)
	response := g.SendEvent(&game.Event{Type: game.CONNECT, PlayerToken: playerToken, Return: receiver})
	if response == "" {
		http.Error(w, "Game: Not allowed", 401)
		log.Printf("Game: Not allowed")
		return
	}

	playerId, _ := strconv.ParseInt(response, 10, 32)

	log.Printf("Received player id %d", playerId)

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
				reader <- InputEvent{Disconnect: true}

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
			ev := &game.Event{PlayerID: int(playerId), Value: input.Code, Return: make(chan string, 1)}
			if input.Down {
				ev.Type = game.KEYDOWN
			} else if input.Disconnect {
				ev.Type = game.DISCONNECT
			} else {
				ev.Type = game.KEYUP
			}
			g.SendEvent(ev)
			if ev.Type == game.DISCONNECT {
				return
			}
		}
	}
}

var gameTempl = template.Must(template.ParseFiles("static/game.html"))

func serveGame(w http.ResponseWriter, r *http.Request) {
	if path.Dir(r.URL.Path) != "/game" {
		http.Error(w, "Not found", 404)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}

	playerToken, err := getPlayerToken(w, r, true)
	log.Printf("Player Token = %s", playerToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), 500)
		log.Printf("error getting player token")
		return
	}

	gameToken := path.Base(r.URL.Path)
	game := getGame(GET_GAME, playerToken, gameToken)

	// TODO redirect back to index with "that game has ended" message
	if game == nil {
		http.Error(w, "Game not found", 404)
		log.Printf("error getting player token")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	gameTempl.Execute(w, r.Host)
}

func getPlayerToken(w http.ResponseWriter, r *http.Request, saveSession bool) (token string, err error) {
	var existing *http.Cookie

	if existing, err = r.Cookie("bitarcade_session"); err == nil {
		token = existing.Value
	} else {
		if saveSession {
			token = Token()
			cookie := &http.Cookie{
				Name:     "bitarcade_session",
				Value:    token,
				Expires:  time.Now().Add(5 * 365 * 24 * time.Hour),
				HttpOnly: true,
			}
			http.SetCookie(w, cookie)
		} else {
			return "", fmt.Errorf("Request has no session")
		}
	}

	return
}

var indexTempl = template.Must(template.ParseFiles("static/index.html"))

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTempl.Execute(w, r.Host); err != nil {
		panic(err)
	}
}

func serveSummary(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/summary.json" {
		http.Error(w, "Not found", 404)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}

	response := make(chan *MatchMakerEvent, 0)

	request := &MatchMakerEvent{
		Type:   GET_SUMMARY,
		Return: response,
	}

	Matcher.Events <- request
	event := <-response

	fmt.Fprintf(w, "%s", string(event.Summary))

	w.Header().Set("Content-Type", "application/json")
}
