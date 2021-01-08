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
}

func newAuthV2Server() *authV2Server { return &authV2Server{} }

func (srv *authV2Server) Check(
	ctx context.Context, req *authv2.CheckRequest) (
	_ *authv2.CheckResponse, err error) {
	log.Debugf("AuthV2.Check|%#v", req)
	ticket := req.Attributes.Request.Http.Headers[xLibraTicket]
	appId, roleId, err := db.checkTicket(ctx, ticket)
	if err != nil {
		return srv.toResponse(err)
	}
	role, err := db.getRole(ctx, appId, roleId)
	if err != nil {
		return srv.toResponse(err)
	}
	return &authv2.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authv2.CheckResponse_OkResponse{
			OkResponse: &authv2.OkHttpResponse{
				Headers: []*corev2.HeaderValueOption{
					{
						Header: &corev2.HeaderValue{
							Key:   xLibraUserId,
							Value: role.UserId,
						},
					},
					{
						Header: &corev2.HeaderValue{
							Key:   xLibraRoleId,
							Value: role.Id,
						},
					},
					{
						Header: &corev2.HeaderValue{
							Key:   xLibraRoleIndex,
							Value: fmt.Sprintf("%d", role.Index),
						},
					},
				},
			},
		},
	}, nil
}
func (*authV2Server) toResponse(err error) (*authv2.CheckResponse, error) {
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

type authV3Server struct {
	authv3.UnimplementedAuthorizationServer
}

func newAuthV3Server() *authV3Server { return &authV3Server{} }

func (srv *authV3Server) Check(
	ctx context.Context, req *authv3.CheckRequest) (
	_ *authv3.CheckResponse, err error) {
	log.Debugf("AuthV3.Check|%#v", req)
	ticket := req.Attributes.Request.Http.Headers[xLibraTicket]
	appId, roleId, err := db.checkTicket(ctx, ticket)
	if err != nil {
		return srv.toResponse(err)
	}
	role, err := db.getRole(ctx, appId, roleId)
	if err != nil {
		return srv.toResponse(err)
	}
	return &authv3.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   xLibraUserId,
							Value: role.UserId,
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   xLibraRoleId,
							Value: role.Id,
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   xLibraRoleIndex,
							Value: fmt.Sprintf("%d", role.Index),
						},
					},
				},
			},
		},
	}, nil
}

func (*authV3Server) toResponse(err error) (*authv3.CheckResponse, error) {
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
