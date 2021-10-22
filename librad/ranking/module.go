package ranking

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
	servermodule "github.com/onemoreteam/httpframework/modularity/server"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func init() {
	modularity.Register(&rankingModule{})
}

type rankingModule struct {
	modularity.Skeleton
}

func (rankingModule) Name() string { return "ranking" }

func (m *rankingModule) Initialize(jb json.RawMessage) (err error) {
	srv, err := createServer(jb)
	if err != nil {
		return
	}
	v1pb.RegisterBubbleChartServer(servermodule.Default, srv.bubblechart)
	v1pb.RegisterLeaderboardServer(servermodule.Default, srv.leaderboard)
	return
}
