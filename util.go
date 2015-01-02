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

var Prefix = []string{
	"Super",
	"Ultra",
	"Vice",
	"Captain",
	"Pro",
	"Death",
}

var Names = []string{
	"Fox",
	"Bear",
	"Wolf",
	"Chicken",
	"Bringer",
}

var Suffix = []string{
	"2000",
	"Pro",
	"Elite",
}
