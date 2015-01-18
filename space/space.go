package space

import (
	"encoding/json"
	"fmt"
	"github.com/boourns/bitarcade/game"
	"log"
	"math"
	"time"
)

const (
	SCREEN_WIDTH  = 640
	SCREEN_HEIGHT = 480
)

const (
	MAX_PLAYERS              = 6
	MAXSPEED                 = 5.0
	MAXBULLETS               = 3
	FRAMES_TILL_NEXT_SHOT    = 4
	BULLET_SPEED             = 10.0
	MAX_DISCONNECTED_SECONDS = 5
	DEATH_SECONDS            = 1
	BULLET_LIFE_FRAMES       = 25
)

type Position struct {
	X         int
	Y         int
	Direction float64
	Size      int
	SpeedX    float64 `json:"-"`
	SpeedY    float64 `json:"-"`
}

type Player struct {
	Id               int
	Position         Position
	State            int
	InvincibleFrames uint
	Score            int
	KilledBy         int
}

type PlayerContext struct {
	Player             *Player
	Keys               map[int]bool
	Return             chan string
	GameOverUntil      int64
	Token              string
	LiveBulletCount    int
	FramesTillNextShot int
	DisconnectedTime   int64
}

type Bullet struct {
	Position      *Position
	FramesTillEnd int64
	OwnerPlayerId int
}

type Space struct {
	Events       chan *game.Event       `json:"-"`
	Players      map[int]*PlayerContext `json:"-"`
	Bullets      []*Bullet              `json:"-"`
	PlayerCount  int
	nextPlayerId int
	timerChan    chan bool `json:"-"`
}

type SerializedGame struct {
	PlayerId int
	Players  []*Player
	Bullets  []*Bullet
	Events   []string
}

const (
	GAMEOVER     = 0
	DISCONNECTED = 1
	PLAYING      = 2
)

const (
	LEFT  = 37
	UP    = 38
	DOWN  = 40
	RIGHT = 39
	SPACE = 32
)

func New() *Space {
	ret := &Space{
		Players:   make(map[int]*PlayerContext),
		timerChan: make(chan bool, 0),
	}
	ret.Events = make(chan *game.Event, 0)

	go ret.gameLoop()
	go ret.timer()
	NewBot(ret, "Alice")
	NewBot(ret, "Bob")

	return ret
}

func (g *Space) handleJoin(input *game.Event) {
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
			Id:       newPlayer,
			State:    DISCONNECTED,
			Position: Position{X: 320, Y: 240, Size: 10},
			KilledBy: -1,
		},
		DisconnectedTime: time.Now().Unix(),
		Token:            input.PlayerToken,
		Keys:             make(map[int]bool, 0),
		GameOverUntil:    time.Now().Unix(),
	}
	select {
	case input.Return <- fmt.Sprintf("%d", newPlayer):
	default:
	}
}

func (g *Space) gameLoop() {
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
					log.Printf("Player %s joined as ID %d", playerContext.Token, playerContext.Player.Id)
					playerContext.Player.State = GAMEOVER
					playerContext.Return = input.Return
					select {
					case input.Return <- fmt.Sprintf("%d", playerContext.Player.Id):
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
				throttle := 0.0
				x := p.Position.SpeedX
				y := p.Position.SpeedY

				curSpeed := math.Sqrt(x*x + y*y)

				if pc.FramesTillNextShot > 0 {
					pc.FramesTillNextShot--
				}
				if pc.Player.State == DISCONNECTED && time.Now().Unix() > pc.DisconnectedTime+MAX_DISCONNECTED_SECONDS {
					log.Printf("Deleting player %s, disconnected", pc.Token)
					delete(g.Players, token)
					g.PlayerCount--
					gameEvents = append(gameEvents, fmt.Sprintf("Player %d disconnected.", pc.Player.Id))
					continue
				}

				for key, down := range pc.Keys {
					if down == true {
						switch key {
						case UP:
							if p.State == GAMEOVER {
								break
							}
							throttle = 1.0
						case LEFT:
							if p.State == GAMEOVER {
								break
							}
							p.Position.Direction += (0.5 * (0.1 + curSpeed/20.0))
						case RIGHT:
							if p.State == GAMEOVER {
								break
							}
							p.Position.Direction -= (0.5 * (0.1 + curSpeed/20.0))
						case SPACE:
							if p.State == GAMEOVER {
								fmt.Printf("GameOverUntil %v Now %v\n", pc.GameOverUntil, time.Now().Unix())
								if time.Now().Unix() > pc.GameOverUntil {
									fmt.Printf("Playing now!\n")
									p.State = PLAYING
									p.InvincibleFrames = 75
									gameEvents = append(gameEvents, fmt.Sprintf("Player %d Joined.", p.Id))
								}
							} else {
								if pc.LiveBulletCount < MAXBULLETS && pc.FramesTillNextShot == 0 {
									pc.FramesTillNextShot = FRAMES_TILL_NEXT_SHOT
									pc.LiveBulletCount++
									newBullet := &Bullet{
										Position: &Position{
											Direction: p.Position.Direction,
											X:         p.Position.X,
											Y:         p.Position.Y,
										},
										FramesTillEnd: BULLET_LIFE_FRAMES,
										OwnerPlayerId: p.Id,
									}
									x, y = math.Sincos(newBullet.Position.Direction)
									newBullet.Position.SpeedX = x * BULLET_SPEED
									newBullet.Position.SpeedY = y * BULLET_SPEED
									newBullet.Position.X += int(x * 5.0)
									newBullet.Position.Y += int(y * 5.0)

									g.Bullets = append(g.Bullets, newBullet)
								}
							}
						}
					}
				}
				x, y = math.Sincos(p.Position.Direction)
				p.Position.SpeedX += x * throttle
				p.Position.SpeedY += y * throttle

				if p.State == GAMEOVER {
					p.Position.SpeedX = 0.95 * p.Position.SpeedX
					p.Position.SpeedY = 0.95 * p.Position.SpeedY
				}

				if p.Position.SpeedX > MAXSPEED {
					p.Position.SpeedX = MAXSPEED
				}
				if p.Position.SpeedX < -MAXSPEED {
					p.Position.SpeedX = -MAXSPEED
				}
				if p.Position.SpeedY > MAXSPEED {
					p.Position.SpeedY = MAXSPEED
				}
				if p.Position.SpeedY < -MAXSPEED {
					p.Position.SpeedY = -MAXSPEED
				}

				p.Position.Adjust()

				if p.InvincibleFrames > 0 {
					p.InvincibleFrames--
				}
			}

			newBullets := []*Bullet{}

			for _, v := range g.Bullets {
				v.FramesTillEnd--
				if v.FramesTillEnd > 0 {
					newBullets = append(newBullets, v)
				} else {
					g.Players[v.OwnerPlayerId].LiveBulletCount--
				}

				for _, pc := range g.Players {
					if pc.Player.Id != v.OwnerPlayerId && pc.Player.InvincibleFrames == 0 && pc.Player.State == PLAYING && distance(*v.Position, pc.Player.Position) < 10.0 {
						pc.Player.State = GAMEOVER
						pc.Player.KilledBy = v.OwnerPlayerId
						gameEvents = append(gameEvents, fmt.Sprintf("Player %d killed Player %d.", v.OwnerPlayerId, pc.Player.Id))

						if v.OwnerPlayerId != pc.Player.Id {
							g.Players[v.OwnerPlayerId].Player.Score++
						}
						pc.GameOverUntil = time.Now().Unix() + DEATH_SECONDS
					}
				}

				v.Position.Adjust()
			}

			g.Bullets = newBullets

			data := SerializedGame{
				Bullets: g.Bullets,
				Events:  gameEvents,
			}
			for _, v := range g.Players {
				data.Players = append(data.Players, v.Player)
			}
			for id, pc := range g.Players {
				if pc.Player.State == DISCONNECTED {
					continue
				}
				data.PlayerId = id
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

func (g *Space) timer() {
	for true {
		time.Sleep(33 * time.Millisecond)
		g.timerChan <- false
	}
}

func (p *Position) Adjust() {
	p.X += int(p.SpeedX)
	p.Y += int(p.SpeedY)
	for p.X > SCREEN_WIDTH {
		p.X -= SCREEN_WIDTH
	}
	for p.Y > SCREEN_HEIGHT {
		p.Y -= SCREEN_HEIGHT
	}
	for p.X < 0 {
		p.X += SCREEN_WIDTH
	}
	for p.Y < 0 {
		p.Y += SCREEN_HEIGHT
	}
}

func distance(a Position, b Position) float64 {
	diffX := a.X - b.X
	diffY := a.Y - b.Y
	return math.Sqrt(float64((diffX * diffX) + (diffY * diffY)))
}

func (s *Space) SendEvent(event *game.Event) string {
	s.Events <- event
	return <-event.Return
}

func (s *Space) AcceptingPlayers() bool {
	return s.PlayerCount < MAX_PLAYERS
}

func (s *Space) Summary() interface{} {
	return nil
}
