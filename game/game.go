package game

const (
	JOIN       = iota
	CONNECT    = iota
	DISCONNECT = iota
	KEYDOWN    = iota
	KEYUP      = iota
)

type Event struct {
	Type        int
	Value       int
	PlayerToken string
	PlayerID    int

	Return chan string
}

type Game interface {
	SendEvent(*Event) string
	AcceptingPlayers() bool
	Summary() interface{}
}
