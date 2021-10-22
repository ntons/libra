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

	appAdmin  *appAdminServer
	user      *userServer
	role      *roleServer
	auth      *authServer
	userAdmin *userAdminServer
	roleAdmin *roleAdminServer
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
	srv.appAdmin = newAppServer()
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.auth = newAuthServer()
	srv.userAdmin = newUserAdminServer()
	srv.roleAdmin = newRoleAdminServer()
	return srv, nil
}

func (srv *server) Serve() { dbServe(srv.ctx) }

func (srv *server) Stop() { srv.stop() }

func (srv *server) RegisterGrpc(s *grpc.Server) (err error) {
	admv1pb.RegisterAppAdminServer(s, srv.appAdmin)
	v1pb.RegisterUserServer(s, srv.user)
	v1pb.RegisterRoleServer(s, srv.role)
	authpb.RegisterAuthorizationServer(s, srv.auth)
	v1pb.RegisterUserAdminServer(s, srv.userAdmin)
	v1pb.RegisterRoleAdminServer(s, srv.roleAdmin)
	return
}
