package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"
)

func main() {
	events = make(chan Event, 0)
	World.Players = make(map[int]*PlayerContext)

	go eventHandler(events)

	go func(timer chan Event) {
		for true {
			time.Sleep(50 * time.Millisecond)
			timer <- Event{Type: TIMER}
		}
	}(events)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", playerHandler)
	http.HandleFunc("/jquery.min.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/jquery.min.js")
	})

	http.ListenAndServe("0.0.0.0:8080", nil)
}
