package portal

import (
	"context"
	"encoding/json"
	"fmt"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"github.com/ntons/libra-go/api/v1"
	"github.com/ntons/log-go"

	"github.com/ntons/libra/librad/srv"
)

func init() {
	srv.RegisterService("portal", create)
}

const (
	xCookie       = "x-cookie"
	xTokenKey     = "x-libra-token"
	xTicketKey    = "x-libra-ticket"
	xAppIdKey     = "x-libra-app-id"
	xUserIdKey    = "x-libra-user-id"
	xRoleIdKey    = "x-libra-role-id"
	xRoleIndexKey = "x-libra-role-index"
)

type portal struct {
	srv.UnimplementedServer
	//
	ctx    context.Context
	cancel context.CancelFunc
	// database
	db *database
	// implements
	acct *account
	auth *authorization
}

func create(b json.RawMessage) (_ srv.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	pt := &portal{}
	pt.ctx, pt.cancel = context.WithCancel(context.Background())
	if pt.db, err = newDatabase(pt.ctx, cfg); err != nil {
		pt.cancel()
		return nil, fmt.Errorf("failed to dail database: %v", err)
	}
	pt.acct = newAccount(pt.db)
	pt.auth = newAuthorization(pt.db)
	return pt, nil
}

func (pt *portal) RegisterGrpc(grpcSrv *srv.GrpcServer) (err error) {
	v1.RegisterAccountServer(grpcSrv, pt.acct)
	envoyauth.RegisterAuthorizationServer(grpcSrv, pt.auth)
	return
}

func (pt *portal) RegisterGrpcGateway(
	grpcConn *srv.GrpcClientConn,
	gatewayMux *srv.GrpcGatewayServeMux) (err error) {
	if err = v1.RegisterAccountHandler(
		pt.ctx, gatewayMux, grpcConn); err != nil {
		return
	}
	return
}

func (pt *portal) Serve() {
	log.Infof("portal::Serve")
	pt.db.Serve()
}

func (pt *portal) Stop() {
	log.Infof("portal::Stop")
	pt.cancel()
}
