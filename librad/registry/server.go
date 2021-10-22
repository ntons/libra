package registry

import (
	"context"
	"encoding/json"
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

func createServer(_ json.RawMessage) (_ *server, err error) {
	srv := &server{}
	srv.ctx, srv.stop = context.WithCancel(context.Background())
	srv.appAdmin = newAppServer()
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.auth = newAuthServer()
	srv.userAdmin = newUserAdminServer()
	srv.roleAdmin = newRoleAdminServer()
	return srv, nil
}
