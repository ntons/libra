package registry

import (
	"context"
	"encoding/json"
	"fmt"
)

type server struct {
	ctx  context.Context
	stop context.CancelFunc

	appAdmin  *appAdminServer
	user      *userServer
	role      *roleServer
	auth      *authServer
	userAdmin *userAdminServer
	roleAdmin *roleAdminServer
}

func createServer(b json.RawMessage) (_ *server, err error) {
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	} else if err = cfg.parse(); err != nil {
		return
	}

	srv := &server{}
	srv.ctx, srv.stop = context.WithCancel(context.Background())
	if err = dialDatabase(srv.ctx); err != nil {
		return nil, fmt.Errorf("failed to dial database: %v", err)
	}
	srv.appAdmin = newAppServer()
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.auth = newAuthServer()
	srv.userAdmin = newUserAdminServer()
	srv.roleAdmin = newRoleAdminServer()
	return srv, nil
}
