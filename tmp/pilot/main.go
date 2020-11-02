package main

import (
	"math/rand"
	"os"
	"time"

	log "github.com/ntons/log-go"

	"github.com/ntons/libra/common/server"
	_ "github.com/ntons/libra/pilot/xds"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	if err := server.Serve(); err != nil {
		log.Errorf("failed to serve: %v", err)
		os.Exit(1)
	}
}
