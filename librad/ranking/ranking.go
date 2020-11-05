package ranking

import (
	"encoding/json"

	"github.com/ntons/libra-go/api/v1"

	"github.com/ntons/libra/librad/comm"
)

func init() {
	comm.RegisterService("ranking", create)
}

type request interface {
	GetKey() *v1.ChartKey
	GetOptions() *v1.ChartOptions
}

type rankingServer struct {
	comm.UnimplementedServer
	bb *bubbleServer
	lb *leaderboardServer
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	bb, err := newBubbleServer(cfg.Bubble.Redis)
	if err != nil {
		return
	}
	lb, err := newLeaderboardServer(cfg.Leaderboard.Redis)
	if err != nil {
		return
	}
	return &rankingServer{bb: bb, lb: lb}, nil
}

func (r *rankingServer) RegisterGrpc(s *comm.GrpcServer) (err error) {
	v1.RegisterBubbleServer(s, r.bb)
	v1.RegisterLeaderboardServer(s, r.lb)
	return
}
