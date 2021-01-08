package portal

import (
	"context"
	"encoding/json"
	"fmt"

	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	grpcgw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/ntons/libra-go/api/v1"
	"google.golang.org/grpc"

	"github.com/ntons/libra/librad/comm"
)

func init() { comm.RegisterService("portal", create) }

const (
	xLibraToken        = "x-libra-token"
	xLibraCookieToken  = "x-libra-cookie-token"
	xLibraTicket       = "x-libra-ticket"
	xLibraCookieTicket = "x-libra-cookie-ticket"
	xLibraAppId        = "x-libra-app-id"
	xLibraUserId       = "x-libra-user-id"
	xLibraRoleId       = "x-libra-role-id"
	xLibraRoleIndex    = "x-libra-role-index"
)

type config struct {
	Redis string
	Mongo string
}

type portalServer struct {
	ctx    context.Context
	cancel context.CancelFunc
	// implements
	user   *userServer
	role   *roleServer
	authv2 *authV2Server
	authv3 *authV3Server
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	srv := &portalServer{}
	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	if db, err = dialDatabase(srv.ctx, cfg); err != nil {
		srv.Close()
		return nil, fmt.Errorf("failed to dail database: %v", err)
	}
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.authv2 = newAuthV2Server()
	srv.authv3 = newAuthV3Server()
	return srv, nil
}

func (srv *portalServer) Serve() { db.Serve() }

func (srv *portalServer) Close() { srv.cancel() }

func (srv *portalServer) RegisterGrpc(s *grpc.Server) (err error) {
	v1.RegisterUserServer(s, srv.user)
	v1.RegisterRoleServer(s, srv.role)
	authv2.RegisterAuthorizationServer(s, srv.authv2)
	authv3.RegisterAuthorizationServer(s, srv.authv3)
	return
}

func (srv *portalServer) RegisterGrpcGateway(
	cc *grpc.ClientConn, sm *grpcgw.ServeMux) (err error) {
	if err = v1.RegisterUserHandler(srv.ctx, sm, cc); err != nil {
		return
	}
	if err = v1.RegisterRoleHandler(srv.ctx, sm, cc); err != nil {
		return
	}
	return
}
