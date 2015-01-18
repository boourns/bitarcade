package space

import (
	"encoding/json"
	"github.com/boourns/bitarcade/game"
	"log"
	"math/rand"
	"strconv"
)

func sendKey(key int, direction int) {
}

func NewBot(g *Space, name string) {
	receiver := make(chan string, 1)

	response := g.SendEvent(&game.Event{Type: game.JOIN, PlayerToken: name, Return: receiver})
	if response == "" {
		log.Printf("Game: Not allowed")
		return
	}

	response = g.SendEvent(&game.Event{Type: game.CONNECT, PlayerToken: name, Return: receiver})
	if response == "" {
		log.Printf("Game: Not allowed")
		return
	}
	playerId64, _ := strconv.ParseInt(response, 10, 32)
	playerId := int(playerId64)

	log.Printf("Joined game as %d", playerId)

	go func(receiver chan string) {
		keyHolds := map[int]int{}
		for {
			select {
			case worldString := <-receiver:
				var world SerializedGame
				err := json.Unmarshal([]byte(worldString), &world)
				if err != nil {
					log.Printf("error decoding world state! %s", err)
				}
				var me *Player
				for _, v := range world.Players {
					if v.Id == int(playerId) {
						me = v
						if me.State == GAMEOVER {
							keyHolds[SPACE] = 5
						} else {
							keyHolds[UP] = 10
							if rand.Int()%10 < 2 {
								keyHolds[LEFT] = 10
							} else if rand.Int()%10 < 2 {
								keyHolds[RIGHT] = 10
							} else if rand.Int()%5 == 1 {
								keyHolds[SPACE] = 5
							}
						}
					}
				}
				for key, frames := range keyHolds {
					keyHolds[key] = frames - 1
					if keyHolds[key] == 0 {
						g.SendEvent(&game.Event{Type: game.KEYUP, Value: key, PlayerID: playerId, PlayerToken: name, Return: receiver})
						delete(keyHolds, key)
					} else {
						g.SendEvent(&game.Event{Type: game.KEYDOWN, Value: key, PlayerID: playerId, PlayerToken: name, Return: receiver})
					}
				}
			}
		}
	}(receiver)
}
