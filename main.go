package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", serveIndex)

	for name, _ := range Games {
		http.HandleFunc(fmt.Sprintf("/%s", name), serveMatchMaker)
		http.HandleFunc(fmt.Sprintf("/%s/", name), serveGame)
	}

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
	http.ListenAndServe(bind, Log(http.DefaultServeMux))
}
