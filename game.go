package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Position struct {
	X         int
	Y         int
	Direction float64
	Size      int
	SpeedX    float64
	SpeedY    float64
}

type Player struct {
	Position Position
	Keys     map[int]bool
	State    int
	Return   chan string
}

type Bullet struct {
	Position *Position
	EndTime  int64
}

var PlayerCount = 0

type Environment struct {
	Players map[int]*Player
	Bullets []*Bullet
}

type SerializedEnvironment struct {
	Playing bool
	Players []*Position
	Bullets []*Bullet
}

var World Environment

const (
	JOIN    = iota
	KEYDOWN = iota
	KEYUP   = iota
	QUIT    = iota
	TIMER   = iota
)

const (
	GAMEOVER = iota
	PLAYING  = iota
)

const (
	LEFT  = 37
	UP    = 38
	DOWN  = 40
	RIGHT = 39
	SPACE = 32
)

func eventHandler(events chan Event) {
	for true {
		select {
		case input := <-events:
			switch input.Type {
			case KEYUP:
				fmt.Printf("Player %d key %d up", input.Player, input.Code)
				World.Players[input.Player].Keys[input.Code] = false
			case KEYDOWN:
				fmt.Printf("Player %d key %d down", input.Player, input.Code)
				World.Players[input.Player].Keys[input.Code] = true
			case JOIN:
				newPlayer := PlayerCount
				PlayerCount++
				fmt.Printf("Player %d Joined\n", newPlayer)
				World.Players[newPlayer] = &Player{
					State:    GAMEOVER,
					Return:   input.Return,
					Keys:     make(map[int]bool, 0),
					Position: Position{X: 320, Y: 240, Size: 10},
				}
				input.Return <- fmt.Sprintf("%d", newPlayer)
			case QUIT:
				delete(World.Players, input.Player)
			case TIMER:
				for _, v := range World.Players {
					throttle := 0.0
					x := v.Position.SpeedX
					y := v.Position.SpeedY

					curSpeed := math.Sqrt(x*x + y*y)

					for key, down := range v.Keys {
						if down == true {
							fmt.Printf("key %d is down!\n", key)
							switch key {
							case UP:
								throttle = 1.0
							case LEFT:
								v.Position.Direction += (0.5 * (0.1 + curSpeed/20.0))
							case RIGHT:
								v.Position.Direction -= (0.5 * (0.1 + curSpeed/20.0))
							case SPACE:
								if v.State == GAMEOVER {
									v.State = PLAYING
								} else {
									newBullet := &Bullet{
										Position: &Position{
											Direction: v.Position.Direction,
											X:         v.Position.X,
											Y:         v.Position.Y,
										},
										EndTime: time.Now().Unix() + 10,
									}
									x, y = math.Sincos(newBullet.Position.Direction)
									newBullet.Position.SpeedX = x * 20.0
									newBullet.Position.SpeedY = y * 20.0

									World.Bullets = append(World.Bullets, newBullet)
								}
							}
						}
					}
					x, y = math.Sincos(v.Position.Direction)
					v.Position.SpeedX += x * throttle
					v.Position.SpeedY += y * throttle

					if v.Position.SpeedX > 10.0 {
						v.Position.SpeedX = 10.0
					}
					if v.Position.SpeedX < -10.0 {
						v.Position.SpeedX = -10.0
					}
					if v.Position.SpeedY > 10.0 {
						v.Position.SpeedY = 10.0
					}
					if v.Position.SpeedY < -10.0 {
						v.Position.SpeedY = -10.0
					}

					v.Position.Adjust()
				}

				newBullets := []*Bullet{}
				now := time.Now().Unix()

				for _, v := range World.Bullets {
					if v.EndTime > now {
						newBullets = append(newBullets, v)
					}
					v.Position.Adjust()
				}

				World.Bullets = newBullets

				data := SerializedEnvironment{
					Bullets: World.Bullets,
				}
				for _, v := range World.Players {
					data.Players = append(data.Players, &v.Position)
				}
				for _, v := range World.Players {
					data.Playing = (v.State != GAMEOVER)
					state, err := json.Marshal(data)
					if err != nil {
						fmt.Printf("Error marshalling world: %v", err)
					}
					go func(state []byte) {
						v.Return <- string(state)
					}(state)
				}
			}
		}
	}
}

func (p *Position) Adjust() {
	p.X += int(p.SpeedX)
	p.Y += int(p.SpeedY)
	for p.X > 640 {
		p.X -= 640
	}
	for p.Y > 480 {
		p.Y -= 480
	}
	for p.X < 0 {
		p.X += 640
	}
	for p.Y < 0 {
		p.Y += 480
	}
}
