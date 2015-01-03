package main

import (
	"encoding/json"
	"fmt"
	"github.com/boourns/bitarcade/game"
	"github.com/boourns/bitarcade/space"
	"log"
	"net/http"
	"strconv"
)

var Matcher *MatchMaker

const (
	FIND_GAME   = iota
	NEW_GAME    = iota
	JOIN_GAME   = iota
	GET_GAME    = iota
	GET_SUMMARY = iota
)

type MatchMakerEvent struct {
	Type        int
	PlayerToken string
	GameToken   string
	Game        game.Game
	Summary     []byte
	Return      chan *MatchMakerEvent
}

type MatchMaker struct {
	games  map[string]game.Game
	Events chan *MatchMakerEvent `json:"-"`
}

func NewMatchMaker() *MatchMaker {
	ret := &MatchMaker{
		games:  make(map[string]game.Game),
		Events: make(chan *MatchMakerEvent, 0),
	}
	go ret.run()
	return ret
}

func (m *MatchMaker) joinGame(gameToken string, playerToken string) game.Game {
	receiver := make(chan string, 0)
	g, ok := m.games[gameToken]
	if !ok {
		return nil
	}

	response := g.SendEvent(&game.Event{Type: game.JOIN, Return: receiver, PlayerToken: playerToken})

	playerId, _ := strconv.ParseInt(response, 10, 32)
	if playerId >= 0 {
		return g
	} else {
		return nil
	}
}

func (m *MatchMaker) newGame() (game.Game, string) {
	space := space.New()
	token := Token()
	m.games[token] = space
	return space, token
}

func (m *MatchMaker) run() {
	for true {
		select {
		case event := <-m.Events:
			log.Printf("received matchmaker event: %v", *event)
			ret := MatchMakerEvent{}

			switch event.Type {
			case FIND_GAME:
				for t, g := range m.games {
					if g.AcceptingPlayers() {
						ret.Game = g
						ret.GameToken = t
					}
				}
				if ret.Game == nil {
					ret.Game, ret.GameToken = m.newGame()
				}
			case NEW_GAME:
				ret.Game, ret.GameToken = m.newGame()

			case JOIN_GAME:
				ret.Game = m.joinGame(event.GameToken, event.PlayerToken)

			case GET_GAME:
				ret.Game = m.games[event.GameToken]

			case GET_SUMMARY:
				summary, err := json.Marshal(m)
				if err != nil {
					panic(err)
				}
				ret.Summary = summary
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
	log.Printf("Player Token = %s", playerToken)

	request := &MatchMakerEvent{
		Type:        FIND_GAME,
		Return:      response,
		PlayerToken: playerToken,
	}

	Matcher.Events <- request
	match := <-response

	http.Redirect(w, r, fmt.Sprintf("/game/%s", match.GameToken), 302)
}
