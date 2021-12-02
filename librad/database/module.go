package database

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
	servermodule "github.com/onemoreteam/httpframework/modularity/server"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func init() {
	modularity.Register(&databaseModule{})
}

type databaseModule struct {
	modularity.Skeleton
}

func (databaseModule) Name() string { return "database" }

func (m *databaseModule) Initialize(jb json.RawMessage) (err error) {
	srv, err := createServer(jb)
	if err != nil {
		return
	}
	servermodule.RegisterService(&v1pb.Database_ServiceDesc, srv)
	servermodule.RegisterService(&v1pb.Distlock_ServiceDesc, srv)
	servermodule.RegisterService(&v1pb.Mailbox_ServiceDesc, srv)
	return
}
