package ranking

import (
	"context"
	"encoding/json"
	"time"

	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/tongo/redis"
)

type request interface {
	GetKey() *v1.ChartKey
	GetOptions() *v1.ChartOptions
}

type server struct {
	bubblechart *bubbleChartServer
	leaderboard *leaderboardServer
}

func createServer(b json.RawMessage) (_ *server, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	srv := &server{}

	if cli, err := redis.Dial(
		ctx, cfg.Bubblechart.Redis, redis.WithPingTest()); err != nil {
		return nil, err
	} else {
		srv.bubblechart = newBubbleChartServer(cli)
	}

	if cli, err := redis.Dial(
		ctx, cfg.Leaderboard.Redis, redis.WithPingTest()); err != nil {
		return nil, err
	} else {
		srv.leaderboard = newLeaderboardServer(cli)
	}

	return srv, nil
}
