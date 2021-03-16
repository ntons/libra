package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/log-go"
)

// build time variables
var (
	Version   string
	Built     string
	GitCommit string
	GoVersion string
	OSArch    string
)

func _main() (err error) {
	var (
		configFilePath string
		showVersion    bool
	)
	flag.StringVar(&configFilePath, "c", "", "[C]onfig file path")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.Parse()

	log.Debugw("compiled information",
		"Version", Version,
		"Built", Built,
		"GitCommit", GitCommit,
		"GoVersion", GoVersion,
		"OSArch", OSArch,
	)
	if showVersion {
		return // only show version
	}

	if err = loadConfig(configFilePath); err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	log.Debugf("runtime configuration: %#v", cfg)

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, addr := range cfg.Hosts {
		ropt := &redis.Options{
			Addr:     addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}
		rdb := redis.NewClient(ropt)
		if err = func(ctx context.Context) (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			return rdb.Ping(ctx).Err()
		}(ctx); err != nil {
			return fmt.Errorf("failed to connect %s: %v", addr, err)
		}
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			log.Debugf("start watching host: %s", addr)
			WatchHost(ctx, rdb)
			log.Debugf("stop watching host: %s", addr)
		}(addr)
	}

	sig := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGPIPE)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	<-sig

	return
}

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}
