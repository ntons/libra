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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

var (
	// Unauthenticated
	errInvalidToken  = status.Errorf(codes.Unauthenticated, "invalid token")
	errInvalidTicket = status.Errorf(codes.Unauthenticated, "invalid ticket")
	// NotFound
	errAppIdNotFound = status.Errorf(codes.NotFound, "app id not found")
	errUserNotFound  = status.Errorf(codes.NotFound, "user not found")
	errRoleNotFound  = status.Errorf(codes.NotFound, "role not found")
	// AlreadyExists
	errRoleAlreadyExists = status.Errorf(codes.AlreadyExists, "role already exists")
	// InvalidArgument
	errInvalidNonce     = status.Errorf(codes.InvalidArgument, "invalid nonce")
	errInvalidState     = status.Errorf(codes.InvalidArgument, "invalid state")
	errInvalidSignature = status.Errorf(codes.InvalidArgument, "invalid signature")
	errInvalidAppId     = status.Errorf(codes.InvalidArgument, "invalid app id")
	// Internal
	errMalformedUserId = status.Errorf(codes.Internal, "malformed user id")
	errMalformedRoleId = status.Errorf(codes.Internal, "malformed role id")
	// Unavailable
	errDatabaseUnavailable = status.Errorf(codes.Unavailable, "database unavailable")
	// PermissionDenied
	errPermissionDenied = status.Errorf(codes.PermissionDenied, "permission denied")
)

var cfg = struct {
	Redis string
	Mongo string
}{}

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
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	}

	srv := &portalServer{}
	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	if err = dialDatabase(srv.ctx); err != nil {
		srv.Close()
		return nil, fmt.Errorf("failed to dail database: %v", err)
	}
	srv.user = newUserServer()
	srv.role = newRoleServer()
	srv.authv2 = newAuthV2Server()
	srv.authv3 = newAuthV3Server()
	return srv, nil
}

func (srv *portalServer) Serve() { dbServe(srv.ctx) }

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
