package main

import (
	"github.com/gorilla/sessions"
	"net/http"
	_ "net/http/pprof"
)

var store = sessions.NewCookieStore([]byte("bcbce3d0e4aca94b769a4ae424ed0915"), []byte("9b1d10720c8416d195d22f6304be5b1a"))

func main() {
	Matcher = NewMatchMaker()

	http.HandleFunc("/game/", serveGame)
	http.HandleFunc("/", serveMatchMaker)
	http.HandleFunc("/ws", websocketHandler)
	http.HandleFunc("/jquery.min.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/jquery.min.js")
	})

	http.ListenAndServe("0.0.0.0:8080", nil)
}
