package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
)

func Token() string {
	uuid := make([]byte, 16)
	io.ReadFull(rand.Reader, uuid)

	return fmt.Sprintf("%x", uuid[:])
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
