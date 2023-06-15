package purchase

import (
	"encoding/json"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
)

func init() { modularity.Register(&module{}) }

type module struct {
	modularity.Skeleton
}

func (module) Name() string { return "purchase" }

func (module) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		// 那就算了吧
		return
	}
	server.RegisterGrpcService(&v1pb.Order_ServiceDesc, newOrderServer())
	return
}
