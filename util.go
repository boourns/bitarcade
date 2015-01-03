package main

import (
	"crypto/rand"
	"fmt"
	"io"
)

func Token() string {
	uuid := make([]byte, 16)
	io.ReadFull(rand.Reader, uuid)

	return fmt.Sprintf("%x", uuid[:])
}
