package snake

import (
	"encoding/json"
	"fmt"
	"github.com/boourns/bitarcade/game"
	"log"
	"time"
)

const (
	EMPTY = 0
	// values 1-4 reserved for player 1-4 occupance
	WALL = 5
)

const (
	BOARD_HEIGHT             = 40
	BOARD_WIDTH              = 40
	MAX_PLAYERS              = 4
	FRAMES_TILL_MOVE         = 10
	MAX_DISCONNECTED_SECONDS = 5
)

const (
	DEAD                = 0
	DISCONNECTED        = 1
	PLAYING             = 2
	WAITING_FOR_PLAYERS = 3
	COUNTING_DOWN       = 4
)

const (
	LEFT  = 37
	UP    = 38
	DOWN  = 40
	RIGHT = 39
)

type Player struct {
	ID        int
	X         int
	Y         int
	Direction int
	Score     int
	KilledBy  int
	State     int
}

type PlayerContext struct {
	Player           *Player
	Keys             map[int]bool
	Return           chan string
	Token            string
	DisconnectedTime int64
}

type Snake struct {
	Events  chan *game.Event
	Players map[int]*PlayerContext `json:"-"`
	Board   []int

	State                 int
	PlayerCount           int
	SecondsTillGameStarts int

	nextPlayerId int
	timerChan    chan bool
}

type SerializedGame struct {
	PlayerID int

	Players []*Player
	Events  []string
	Board   []int
}

func New() game.Game {
	ret := &Snake{
		Players:   make(map[int]*PlayerContext),
		timerChan: make(chan bool, 0),
	}
	ret.Events = make(chan *game.Event, 0)

	go ret.gameLoop()
	go ret.timer()
	return ret
}

func (g *Snake) handleJoin(input *game.Event) {
	for id, player := range g.Players {
		if player.Token == input.PlayerToken {
			select {
			case input.Return <- fmt.Sprintf("%d", id):
			default:
			}
			return
		}
	}

	newPlayer := g.nextPlayerId
	g.nextPlayerId++
	g.PlayerCount++
	log.Printf("Player %d Joined (token %s)\n", newPlayer, input.PlayerToken)
	g.Players[newPlayer] = &PlayerContext{
		Player: &Player{
			ID:       newPlayer,
			State:    DEAD,
			KilledBy: -1,
		},
		DisconnectedTime: time.Now().Unix(),
		Token:            input.PlayerToken,
		Keys:             make(map[int]bool, 0),
	}
	select {
	case input.Return <- fmt.Sprintf("%d", newPlayer):
	default:
	}
}

func (g *Snake) gameLoop() {
	for true {
		select {
		case input := <-g.Events:
			switch input.Type {
			case game.JOIN:
				// sends return event in handleJoin
				g.handleJoin(input)
			case game.CONNECT:
				log.Printf("Player connecting to game")
				var playerContext *PlayerContext
				for _, p := range g.Players {
					log.Printf("Comparing player %s and %s", p.Token, input.PlayerToken)
					if p.Token == input.PlayerToken {
						playerContext = p
					}
				}
				if playerContext == nil {
					select {
					case input.Return <- "":
					default:
					}
				} else {
					log.Printf("Player %s joined as ID %d", playerContext.Token, playerContext.Player.ID)
					playerContext.Player.State = DEAD
					playerContext.Return = input.Return
					select {
					case input.Return <- fmt.Sprintf("%d", playerContext.Player.ID):
					default:
					}
				}
			case game.DISCONNECT:
				if g.Players[input.PlayerID] != nil {
					g.Players[input.PlayerID].Player.State = DISCONNECTED
					g.Players[input.PlayerID].DisconnectedTime = time.Now().Unix()
				}
				input.Return <- ""
			case game.KEYUP:
				if g.Players[input.PlayerID] != nil {
					g.Players[input.PlayerID].Keys[input.Value] = false
				}
				input.Return <- ""
			case game.KEYDOWN:
				if g.Players[input.PlayerID] != nil {
					g.Players[input.PlayerID].Keys[input.Value] = true
				}
				input.Return <- ""
			}
		case <-g.timerChan:
			var gameEvents = []string{}

			for token, pc := range g.Players {
				p := pc.Player
				if pc.Player.State == DISCONNECTED && time.Now().Unix() > pc.DisconnectedTime+MAX_DISCONNECTED_SECONDS {
					log.Printf("Deleting player %s, disconnected", pc.Token)
					delete(g.Players, token)
					g.PlayerCount--
					gameEvents = append(gameEvents, fmt.Sprintf("Player %d disconnected.", pc.Player.ID))
					continue
				}

				for key, down := range pc.Keys {
					if down == true {
						switch key {
						case UP:
							if p.State == DEAD {
								break
							}
						case LEFT:
							if p.State == DEAD {
								break
							}
						case RIGHT:
							if p.State == DEAD {
								break
							}
						}
					}
				}
			}

			data := SerializedGame{
				Events: gameEvents,
			}
			for _, v := range g.Players {
				data.Players = append(data.Players, v.Player)
			}
			for id, pc := range g.Players {
				if pc.Player.State == DISCONNECTED {
					continue
				}
				data.PlayerID = id
				state, err := json.Marshal(data)
				if err != nil {
					fmt.Printf("Error marshalling world: %v", err)
				}
				select {
				case pc.Return <- string(state):
				default:
				}
			}
		}
	}
}

func (g *Snake) timer() {
	for true {
		time.Sleep(33 * time.Millisecond)
		g.timerChan <- false
	}
}

func (s *Snake) SendEvent(event *game.Event) string {
	s.Events <- event
	return <-event.Return
}

func (s *Snake) AcceptingPlayers() bool {
	return s.PlayerCount < MAX_PLAYERS
}

func (s *Snake) Summary() interface{} {
	return nil
}
