package portal

import (
	"context"
	"encoding/json"
	"fmt"

	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/ntons/libra-go/api/v1"

	"github.com/ntons/libra/librad/comm"
)

func init() { comm.RegisterService("portal", create) }

const (
	xCookie       = "x-cookie"
	xTokenKey     = "x-libra-token"
	xTicketKey    = "x-libra-ticket"
	xAppIdKey     = "x-libra-app-id"
	xUserIdKey    = "x-libra-user-id"
	xRoleIdKey    = "x-libra-role-id"
	xRoleIndexKey = "x-libra-role-index"
)

type portalServer struct {
	comm.UnimplementedServer
	//
	ctx    context.Context
	cancel context.CancelFunc
	// database
	db *database
	// implements
	acct   *acctServer
	authv2 *authV2Server
	authv3 *authV3Server
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	pt := &portalServer{}
	pt.ctx, pt.cancel = context.WithCancel(context.Background())
	if pt.db, err = newDatabase(pt.ctx, cfg); err != nil {
		pt.cancel()
		return nil, fmt.Errorf("failed to dail database: %v", err)
	}
	pt.acct = newAcctServer(pt.db)
	pt.authv2 = newAuthV2Server(pt.db)
	pt.authv3 = newAuthV3Server(pt.db)
	return pt, nil
}

func (pt *portalServer) RegisterGrpc(s *comm.GrpcServer) (err error) {
	v1.RegisterAccountServer(s, pt.acct)
	authv2.RegisterAuthorizationServer(s, pt.authv2)
	authv3.RegisterAuthorizationServer(s, pt.authv3)
	return
}
func (pt *portalServer) RegisterGrpcGateway(
	cc *comm.GrpcClientConn, sm *comm.GrpcGatewayServeMux) (err error) {
	if err = v1.RegisterAccountHandler(pt.ctx, sm, cc); err != nil {
		return
	}
	return
}
func (pt *portalServer) Serve() { pt.db.Serve() }
func (pt *portalServer) Stop()  { pt.cancel() }
