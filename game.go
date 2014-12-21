package main

import (
	"encoding/json"
	"fmt"
	"math"
)

type Player struct {
	Position Position
	Keys     map[int]bool
	State    int
	Return   chan string
}

type Position struct {
	X         int
	Y         int
	Direction float64
	Size      int
	SpeedX    float64
	SpeedY    float64
}

var PlayerCount = 0

type Environment struct {
	Players map[int]*Player
	Bullets []*Position
}

type SerializedEnvironment struct {
	Players []*Position
	Bullets []*Position
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

					v.Position.X += int(v.Position.SpeedX)
					v.Position.Y += int(v.Position.SpeedY)
					for v.Position.X > 640 {
						v.Position.X -= 640
					}
					for v.Position.Y > 480 {
						v.Position.Y -= 480
					}
					for v.Position.X < 0 {
						v.Position.X += 640
					}
					for v.Position.Y < 0 {
						v.Position.Y += 480
					}
				}

				data := SerializedEnvironment{
					Bullets: World.Bullets,
				}
				for _, v := range World.Players {
					data.Players = append(data.Players, &v.Position)
				}
				state, err := json.Marshal(data)
				if err != nil {
					fmt.Printf("Error marshalling world: %v", err)
				}
				for _, v := range World.Players {
					go func(state []byte) {
						v.Return <- string(state)
					}(state)
				}
			}
		}
	}
}
