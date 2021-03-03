package ranking

import (
	"encoding/json"

	"github.com/ntons/libra-go/api/v1"
	"google.golang.org/grpc"

	"github.com/ntons/libra/librad/comm"
)

func init() {
	comm.RegisterService("ranking", factory)
}

type request interface {
	GetKey() *v1.ChartKey
	GetOptions() *v1.ChartOptions
}

type server struct {
	bubblechart *bubbleChartServer
	leaderboard *leaderboardServer
}

func factory(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	bubblechart, err := newBubbleChartServer(cfg.Bubble.Redis)
	if err != nil {
		return
	}
	leaderboard, err := newLeaderboardServer(cfg.Leaderboard.Redis)
	if err != nil {
		return
	}
	return &server{
		bubblechart: bubblechart,
		leaderboard: leaderboard,
	}, nil
}

func (r *server) RegisterGrpc(s *grpc.Server) (err error) {
	v1.RegisterBubbleChartServer(s, r.bubblechart)
	v1.RegisterLeaderboardServer(s, r.leaderboard)
	return
}

func (*server) Serve() {}
func (*server) Stop()  {}
