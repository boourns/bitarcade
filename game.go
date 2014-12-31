package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

const (
	SCREEN_WIDTH  = 640
	SCREEN_HEIGHT = 480
)

const (
	MAXSPEED   = 10.0
	MAXBULLETS = 10
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
	Player          *Player
	Keys            map[int]bool
	Return          chan string
	GameOverUntil   int64
	Token           string
	LiveBulletCount int
}

type Bullet struct {
	Position      *Position
	FramesTillEnd int64
	OwnerPlayerId int
}

type Game struct {
	Playing      bool
	EndScore     int
	Players      map[int]*PlayerContext
	Bullets      []*Bullet
	PlayerCount  int
	Events       chan *Event
	nextPlayerId int
}

type SerializedGame struct {
	PlayerId int
	Players  []*Player
	Bullets  []*Bullet
}

const (
	JOIN    = iota
	CONNECT = iota
	KEYDOWN = iota
	KEYUP   = iota
	QUIT    = iota
	TIMER   = iota
)

const (
	GAMEOVER     = iota
	DISCONNECTED = iota
	PLAYING      = iota
)

const (
	LEFT  = 37
	UP    = 38
	DOWN  = 40
	RIGHT = 39
	SPACE = 32
)

func newGame() *Game {
	ret := &Game{
		Events:  make(chan *Event, 0),
		Players: make(map[int]*PlayerContext),
	}

	go ret.eventHandler(ret.Events)
	go ret.timer()
	return ret
}

func (g *Game) handleJoin(input *Event) {
	for id, player := range g.Players {
		if player.Token == input.PlayerToken {
			input.Return <- fmt.Sprintf("%d", id)
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
		},
		Token:         input.PlayerToken,
		Keys:          make(map[int]bool, 0),
		GameOverUntil: time.Now().Unix() + 3,
	}
	input.Return <- fmt.Sprintf("%d", newPlayer)
}

func (g *Game) eventHandler(events chan *Event) {
	for true {
		select {
		case input := <-events:
			switch input.Type {
			case KEYUP:
				g.Players[input.Player].Keys[input.Code] = false
			case KEYDOWN:
				g.Players[input.Player].Keys[input.Code] = true
			case JOIN:
				g.handleJoin(input)
			case CONNECT:
				log.Printf("Player connecting to game")
				var playerContext *PlayerContext
				for _, p := range g.Players {
					log.Printf("Comparing player %s and %s", p.Token, input.PlayerToken)
					if p.Token == input.PlayerToken {
						playerContext = p
					}
				}
				if playerContext == nil {
					input.Return <- ""
				} else {
					playerContext.Player.State = GAMEOVER
					playerContext.Return = input.Return
					input.Return <- fmt.Sprintf("%d", playerContext.Player.Id)
				}
			case QUIT:
				delete(g.Players, input.Player)
				g.PlayerCount--
			case TIMER:
				for _, pc := range g.Players {
					p := pc.Player
					throttle := 0.0
					x := p.Position.SpeedX
					y := p.Position.SpeedY

					curSpeed := math.Sqrt(x*x + y*y)

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
										p.InvincibleFrames = 180
									}
								} else {
									if pc.LiveBulletCount < MAXBULLETS {
										pc.LiveBulletCount++
										newBullet := &Bullet{
											Position: &Position{
												Direction: p.Position.Direction,
												X:         p.Position.X,
												Y:         p.Position.Y,
											},
											FramesTillEnd: 60,
											OwnerPlayerId: p.Id,
										}
										x, y = math.Sincos(newBullet.Position.Direction)
										newBullet.Position.SpeedX = x * 15.0
										newBullet.Position.SpeedY = y * 15.0

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
						if distance(*v.Position, pc.Player.Position) < 2.0 && pc.Player.InvincibleFrames == 0 && pc.Player.State == PLAYING {
							pc.Player.State = GAMEOVER
							pc.Player.KilledBy = v.OwnerPlayerId
							if v.OwnerPlayerId != pc.Player.Id {
								g.Players[v.OwnerPlayerId].Player.Score++
							} else {
								// suicide
								pc.Player.Score--
							}
							pc.GameOverUntil = time.Now().Unix() + 3
						}
					}

					v.Position.Adjust()
				}

				g.Bullets = newBullets

				data := SerializedGame{
					Bullets: g.Bullets,
				}
				for _, v := range g.Players {
					data.Players = append(data.Players, v.Player)
				}
				for id, v := range g.Players {
					data.PlayerId = id
					state, err := json.Marshal(data)
					if err != nil {
						fmt.Printf("Error marshalling world: %v", err)
					}
					go func(output chan string, state []byte) {
						output <- string(state)
					}(v.Return, state)
				}
			}
		}
	}
}

func (g *Game) timer() {
	for true {
		time.Sleep(33 * time.Millisecond)
		g.Events <- &Event{Type: TIMER}
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
