package ranking

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ntons/libra-go/api/v1"
	"google.golang.org/grpc"

	"github.com/ntons/libra/librad/comm"
	"github.com/ntons/libra/librad/comm/redis"
)

func init() { comm.RegisterService("ranking", create) }

type request interface {
	GetKey() *v1.ChartKey
	GetOptions() *v1.ChartOptions
}

type server struct {
	bubblechart *bubbleChartServer
	leaderboard *leaderboardServer
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	srv := &server{}

	if cli, err := redis.DialCluster(
		ctx, cfg.Bubblechart.Redis, redis.WithHashTag()); err != nil {
		return nil, err
	} else {
		srv.bubblechart = newBubbleChartServer(cli)
	}

	if cli, err := redis.DialCluster(
		ctx, cfg.Leaderboard.Redis, redis.WithHashTag()); err != nil {
		return nil, err
	} else {
		srv.leaderboard = newLeaderboardServer(cli)
	}

	return srv, nil
}

func (r *server) RegisterGrpc(s *grpc.Server) (err error) {
	v1.RegisterBubbleChartServer(s, r.bubblechart)
	v1.RegisterLeaderboardServer(s, r.leaderboard)
	return
}

func (*server) Serve() {}
func (*server) Stop()  {}
