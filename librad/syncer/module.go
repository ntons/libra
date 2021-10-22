package syncer

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
)

func init() {
	modularity.Register(&module{})
}

type module struct {
	modularity.Skeleton

	srv *server
}

func (module) Name() string { return "syncer" }

func (m *module) Initialize(jb json.RawMessage) (err error) {
	if m.srv, err = createServer(jb); err != nil {
		return
	}
	return
}

func (m *module) Serve() error {
	m.srv.Serve()
	return nil
}

func (m *module) Shutdown() {
	m.srv.Stop()
}
