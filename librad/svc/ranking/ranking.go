package ranking

import (
	"encoding/json"

	"github.com/ntons/libra-go/api/v1"

	"github.com/ntons/libra/librad/srv"
)

func init() {
	srv.RegisterService("ranking", create)
}

type request interface {
	GetKey() *v1.ChartKey
	GetOptions() *v1.ChartOptions
}

type ranking struct {
	srv.UnimplementedServer
	bb *bubble
	lb *leaderboard
}

func create(b json.RawMessage) (_ srv.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	bb, err := newBubble(cfg.Bubble.Redis)
	if err != nil {
		return
	}
	lb, err := newLeaderboard(cfg.Leaderboard.Redis)
	if err != nil {
		return
	}
	return &ranking{bb: bb, lb: lb}, nil
}

func (rk *ranking) RegisterGrpc(grpcSrv *srv.GrpcServer) (err error) {
	v1.RegisterBubbleServer(grpcSrv, rk.bb)
	v1.RegisterLeaderboardServer(grpcSrv, rk.lb)
	return
}
