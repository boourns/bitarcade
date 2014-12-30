package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type MatchMaker struct {
	Games  map[string]*Game
	Events chan *MatchMakerEvent
}

var Matcher *MatchMaker

const (
	FIND_GAME  = iota
	GET_GAME   = iota
	LEAVE_GAME = iota
	NEW_GAME   = iota
)

const (
	MIN_PLAYERS = 2
	MAX_PLAYERS = 8
)

type MatchMakerEvent struct {
	Type        int
	PlayerToken string
	GameToken   string
	Game        *Game
	Return      chan *MatchMakerEvent
}

func NewMatchMaker() *MatchMaker {
	ret := &MatchMaker{
		Games:  make(map[string]*Game),
		Events: make(chan *MatchMakerEvent, 0),
	}
	go ret.run()
	return ret
}

func (m *MatchMaker) joinGame(game *Game, playerToken string) int {
	receiver := make(chan string, 0)
	game.Events <- &Event{Type: JOIN, Return: receiver, PlayerToken: playerToken}
	response := <-receiver

	playerId, _ := strconv.ParseInt(response, 10, 32)
	return int(playerId)
}

func (m *MatchMaker) newGame(playerToken string) (*Game, string) {
	game := newGame()
	token := Token()
	m.Games[token] = game
	m.joinGame(game, playerToken)
	return game, token
}

func (m *MatchMaker) run() {
	for true {
		select {
		case event := <-m.Events:
			log.Printf("received matchmaker event: %v", *event)
			ret := MatchMakerEvent{}

			switch event.Type {
			case FIND_GAME:
				for t, g := range m.Games {
					if g.PlayerCount < MIN_PLAYERS {
						m.joinGame(g, event.PlayerToken)
						ret.Game = g
						ret.GameToken = t
					}
				}
				for t, g := range m.Games {
					if g.PlayerCount < MAX_PLAYERS {
						m.joinGame(g, event.PlayerToken)
						ret.Game = g
						ret.GameToken = t
					}
				}
				if ret.Game == nil {
					ret.Game, ret.GameToken = m.newGame(event.PlayerToken)
				}
			case NEW_GAME:
				ret.Game, ret.GameToken = m.newGame(event.PlayerToken)
			case LEAVE_GAME:
				log.Printf("TODO: LEAVE_GAME")
				//m.leaveGame()
			case GET_GAME:
				ret.Game = m.Games[event.GameToken]
			}
			event.Return <- &ret
		}
	}
}

func serveMatchMaker(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method nod allowed", 405)
		return
	}

	response := make(chan *MatchMakerEvent, 0)

	playerToken, _ := getPlayerToken(w, r, true)

	request := &MatchMakerEvent{
		Type:        FIND_GAME,
		Return:      response,
		PlayerToken: playerToken,
	}

	Matcher.Events <- request
	match := <-response

	log.Printf("Returned %#v", match)

	http.Redirect(w, r, fmt.Sprintf("/game/%s", match.GameToken), 302)
}
