package main

const (
	JOIN       = iota
	CONNECT    = iota
	DISCONNECT = iota
	KEYDOWN    = iota
	KEYUP      = iota
)

type GameEvent struct {
	Type        int
	Value       int
	PlayerToken string
	PlayerID    int

	Return chan string
}

type Game interface {
	SendEvent(*GameEvent) string
	AcceptingPlayers() bool
	Summary() interface{}
}
