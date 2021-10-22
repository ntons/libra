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
	v1pb.RegisterDatabaseServer(servermodule.Default, srv)
	v1pb.RegisterDistlockServer(servermodule.Default, srv)
	v1pb.RegisterMailboxServer(servermodule.Default, srv)
	return
}
