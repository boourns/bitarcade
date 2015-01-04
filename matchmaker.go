package main

import (
	"encoding/json"
	"fmt"
	"github.com/boourns/bitarcade/game"
	"github.com/boourns/bitarcade/snake"
	"github.com/boourns/bitarcade/space"
	"log"
	"net/http"
	"strconv"
)

var Games = map[string]func() game.Game{
	"space": space.New,
	"snake": snake.New,
}

var Matchers map[string]*MatchMaker

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
	gameGenerator func() game.Game
	games         map[string]game.Game
	Events        chan *MatchMakerEvent `json:"-"`
}

func init() {
	for name, _ := range Games {
		Matchers[name] = NewMatchMaker(name)
	}
}

func NewMatchMaker(name string) *MatchMaker {
	ret := &MatchMaker{
		gameGenerator: Games[name],
		games:         make(map[string]game.Game),
		Events:        make(chan *MatchMakerEvent, 0),
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
	space := m.gameGenerator()
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
	gameName := r.URL.Path[1:]

	if _, ok := Games[gameName]; !ok {
		http.Error(w, "Game not found", 404)
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

	Matchers[gameName].Events <- request
	match := <-response

	http.Redirect(w, r, fmt.Sprintf("/game/%s", match.GameToken), 302)
}
