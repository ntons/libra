package portal

import (
	"context"
	"fmt"

	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcstatus "google.golang.org/grpc/status"

	log "github.com/ntons/log-go"
)

type authV2Server struct {
	authv2.UnimplementedAuthorizationServer
	db *database
}

func newAuthV2Server(db *database) *authV2Server {
	return &authV2Server{db: db}
}

func toCheckResponseV2(err error) (*authv2.CheckResponse, error) {
	s, ok := grpcstatus.FromError(err)
	if !ok {
		return nil, err
	}
	return &authv2.CheckResponse{
		Status: &status.Status{
			Code:    int32(s.Code()),
			Message: s.Message(),
		},
	}, nil
}
func (auth *authV2Server) Check(
	ctx context.Context, req *authv2.CheckRequest) (
	_ *authv2.CheckResponse, err error) {
	log.Debugf("AuthV2.Check|%#v", req)
	var (
		appId  = req.Attributes.Request.Http.Headers[xAppIdKey]
		ticket = req.Attributes.Request.Http.Headers[xTicketKey]
	)
	app, err := auth.db.getApp(appId)
	if err != nil {
		return toCheckResponseV2(errInvalidAppId)
	}
	roleId, err := auth.db.checkTicket(ctx, app, ticket)
	if err != nil {
		return toCheckResponseV2(errInvalidTicket)
	}
	role, err := auth.db.getRole(ctx, app.Id, roleId)
	if err != nil {
		return toCheckResponseV2(err)
	}
	return &authv2.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authv2.CheckResponse_OkResponse{
			OkResponse: &authv2.OkHttpResponse{
				Headers: []*corev2.HeaderValueOption{
					{
						Header: &corev2.HeaderValue{
							Key:   xUserIdKey,
							Value: role.UserId,
						},
					},
					{
						Header: &corev2.HeaderValue{
							Key:   xRoleIdKey,
							Value: role.Id,
						},
					},
					{
						Header: &corev2.HeaderValue{
							Key:   xRoleIndexKey,
							Value: fmt.Sprintf("%d", role.Index),
						},
					},
				},
			},
		},
	}, nil
}

type authV3Server struct {
	authv3.UnimplementedAuthorizationServer
	db *database
}

func newAuthV3Server(db *database) *authV3Server {
	return &authV3Server{db: db}
}

func toCheckResponseV3(err error) (*authv3.CheckResponse, error) {
	s, ok := grpcstatus.FromError(err)
	if !ok {
		return nil, err
	}
	return &authv3.CheckResponse{
		Status: &status.Status{
			Code:    int32(s.Code()),
			Message: s.Message(),
		},
	}, nil
}
func (auth *authV3Server) Check(
	ctx context.Context, req *authv3.CheckRequest) (
	_ *authv3.CheckResponse, err error) {
	log.Debugf("AuthV3.Check|%#v", req)
	var (
		appId  = req.Attributes.Request.Http.Headers[xAppIdKey]
		ticket = req.Attributes.Request.Http.Headers[xTicketKey]
	)
	app, err := auth.db.getApp(appId)
	if err != nil {
		return toCheckResponseV3(errInvalidAppId)
	}
	roleId, err := auth.db.checkTicket(ctx, app, ticket)
	if err != nil {
		return toCheckResponseV3(errInvalidTicket)
	}
	role, err := auth.db.getRole(ctx, app.Id, roleId)
	if err != nil {
		return toCheckResponseV3(err)
	}
	return &authv3.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   xUserIdKey,
							Value: role.UserId,
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   xRoleIdKey,
							Value: role.Id,
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   xRoleIndexKey,
							Value: fmt.Sprintf("%d", role.Index),
						},
					},
				},
			},
		},
	}, nil
}
