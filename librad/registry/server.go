package registry

import (
	"context"
	"encoding/json"
	"fmt"

	authpb "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	admv1pb "github.com/ntons/libra-go/api/libra/admin/v1"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"google.golang.org/grpc"

	"github.com/ntons/libra/librad/internal/comm"
)

func init() { comm.RegisterService("registry", create) }

type server struct {
	ctx  context.Context
	stop context.CancelFunc

	app       *appServer
	user      *userServer
	role      *roleServer
	auth      *authServer
	useradmin *userAdminServer
	roleadmin *roleAdminServer
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	} else if err = cfg.parse(); err != nil {
		return
	}

	srv := &server{}
	srv.ctx, srv.stop = context.WithCancel(context.Background())
	if err = dialDatabase(srv.ctx); err != nil {
		srv.Stop()
		return nil, fmt.Errorf("failed to dial database: %v", err)
	}
	srv.app = newAppServer()
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.auth = newAuthServer()
	srv.useradmin = newUserAdminServer()
	srv.roleadmin = newRoleAdminServer()
	return srv, nil
}

func (srv *server) Serve() { dbServe(srv.ctx) }

func (srv *server) Stop() { srv.stop() }

func (srv *server) RegisterGrpc(s *grpc.Server) (err error) {
	admv1pb.RegisterAppServer(s, srv.app)
	v1pb.RegisterUserServer(s, srv.user)
	v1pb.RegisterRoleServer(s, srv.role)
	authpb.RegisterAuthorizationServer(s, srv.auth)
	v1pb.RegisterUserAdminServer(s, srv.useradmin)
	v1pb.RegisterRoleAdminServer(s, srv.roleadmin)
	return
}
