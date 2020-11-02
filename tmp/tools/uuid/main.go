package main

import (
	"crypto/rand"
	"fmt"
	"io"
)

func main() {
	var uuid = make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, uuid); err != nil {
		panic(err)
	}
	fmt.Printf("%x", uuid)
}
