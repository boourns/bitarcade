package main

import (
	"github.com/gorilla/sessions"
	"net/http"
	"os"
)

// yes I know hard-coded credentials - they don't store anything secure yet, don't worry about it :)
var store = sessions.NewCookieStore([]byte("bcbce3d0e4aca94b769a4ae424ed0915"), []byte("9b1d10720c8416d195d22f6304be5b1a"))

func main() {
	Matcher = NewMatchMaker()

	http.HandleFunc("/", serveMatchMaker)
	http.HandleFunc("/game/", serveGame)
	http.HandleFunc("/summary.json", serveSummary)
	http.HandleFunc("/ws", websocketHandler)

	http.HandleFunc("/jquery.min.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/jquery.min.js")
	})
	http.HandleFunc("/background.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/background.png")
	})

	bind := "0.0.0.0:8000"
	if os.Getenv("BITARCADE_BIND") != "" {
		bind = os.Getenv("BITARCADE_BIND")
	}
	http.ListenAndServe(bind, nil)
}
