package portal

import (
	"context"
	"fmt"

	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcstatus "google.golang.org/grpc/status"

	log "github.com/ntons/log-go"
)

type authorization struct {
	envoyauth.UnimplementedAuthorizationServer
	db *database
}

func newAuthorization(db *database) *authorization {
	return &authorization{db: db}
}

func (*authorization) toCheckResponse(
	err error) (*envoyauth.CheckResponse, error) {
	s, ok := grpcstatus.FromError(err)
	if !ok {
		return nil, err
	}
	return &envoyauth.CheckResponse{
		Status: &status.Status{
			Code:    int32(s.Code()),
			Message: s.Message(),
		},
	}, nil
}

// implement auth.AuthorizationServer
func (auth *authorization) Check(
	ctx context.Context, req *envoyauth.CheckRequest) (
	_ *envoyauth.CheckResponse, err error) {
	log.Debugf("Auth.Check|%#v", req)
	var (
		appId  = req.Attributes.Request.Http.Headers[xAppIdKey]
		ticket = req.Attributes.Request.Http.Headers[xTicketKey]
	)
	app, err := auth.db.getApp(appId)
	if err != nil {
		return auth.toCheckResponse(errInvalidAppId)
	}
	roleId, err := auth.db.checkTicket(ctx, app, ticket)
	if err != nil {
		return auth.toCheckResponse(errInvalidTicket)
	}
	role, err := auth.db.getRole(ctx, app.Id, roleId)
	if err != nil {
		return auth.toCheckResponse(err)
	}
	return &envoyauth.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &envoyauth.CheckResponse_OkResponse{
			OkResponse: &envoyauth.OkHttpResponse{
				Headers: []*envoycore.HeaderValueOption{
					{
						Header: &envoycore.HeaderValue{
							Key:   xUserIdKey,
							Value: role.UserId,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   xRoleIdKey,
							Value: role.Id,
						},
					},
					{
						Header: &envoycore.HeaderValue{
							Key:   xRoleIndexKey,
							Value: fmt.Sprintf("%d", role.Index),
						},
					},
				},
			},
		},
	}, nil
}
