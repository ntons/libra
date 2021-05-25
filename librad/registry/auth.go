package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authpb "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	log "github.com/ntons/log-go"
)

func isPermitted(app *xApp, path string) bool {
	if len(path) < 3 || path[0] != '/' {
		return false
	}
	i := strings.IndexByte(path[1:], '/')
	if i < 0 {
		return false
	}
	svc := path[1 : 1+i]
	if strings.HasPrefix(svc, "libra.") {
		return true // libra services are controlled by edge route rule
	}
	if sort.SearchStrings(app.Permissions, svc) < len(app.Permissions) {
		return true // only permitted services are allowed
	}
	return false
}

// 只支持V3版本的验证服务，V2版本缺少Header剔除功能，无法满足安全需求。
type authServer struct {
	authpb.UnimplementedAuthorizationServer
}

func newAuthServer() *authServer { return &authServer{} }

func (srv authServer) Check(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.Check|%v", req)
	for key := range req.Attributes.Request.Http.Headers {
		// 不允许请求中带有x-libra-trusted-头
		if strings.HasPrefix(key, xLibraTrustedPrefix) {
			return srv.errToResponse(errInvalidMetadata)
		}
	}
	if _, ok := req.Attributes.Request.Http.Headers[xLibraAppSecret]; ok {
		return srv.checkAppSecret(ctx, req)
	}
	if _, ok := req.Attributes.Request.Http.Headers[xLibraTicket]; ok {
		return srv.checkTicket(ctx, req)
	}
	if _, ok := req.Attributes.Request.Http.Headers[xLibraToken]; ok {
		return srv.checkToken(ctx, req)
	}
	// 没有任何可用凭证
	return srv.errToResponse(errUnauthenticated)
}

func (srv authServer) checkToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.CheckToken|%v", req)
	token := req.Attributes.Request.Http.Headers[xLibraToken]
	if token == "" {
		return srv.errToResponse(errUnauthenticated)
	}
	appId, userId, err := checkToken(ctx, token)
	if err != nil {
		return srv.errToResponse(err)
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: []*corepb.HeaderValueOption{
					{
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedAppId,
							Value: appId,
						},
						Append: wrapperspb.Bool(false),
					}, {
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedUserId,
							Value: userId,
						},
						Append: wrapperspb.Bool(false),
					},
				},
				//HeadersToRemove: []string{xLibraToken},
			},
		},
	}, nil
}

func (srv authServer) checkTicket(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.CheckTicket|%v", req)
	ticket := req.Attributes.Request.Http.Headers[xLibraTicket]
	if ticket == "" {
		return srv.errToResponse(errUnauthenticated)
	}
	app, role, err := checkTicket(ctx, ticket)
	if err != nil {
		return srv.errToResponse(err)
	}
	if !isPermitted(app, req.Attributes.Request.Http.Path) {
		return srv.errToResponse(errPermissionDenied)
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: []*corepb.HeaderValueOption{
					{
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedAppId,
							Value: app.Id,
						},
						Append: wrapperspb.Bool(false),
					}, {
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedUserId,
							Value: role.UserId,
						},
						Append: wrapperspb.Bool(false),
					}, {
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedRoleId,
							Value: role.Id,
						},
						Append: wrapperspb.Bool(false),
					}, {
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedRoleIndex,
							Value: fmt.Sprintf("%d", role.Index),
						},
						Append: wrapperspb.Bool(false),
					},
				},
				HeadersToRemove: []string{xLibraTicket},
			},
		},
	}, nil
}

func (srv authServer) checkAppSecret(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.CheckSecret|%v", req)
	appId := req.Attributes.Request.Http.Headers[xLibraAppId]
	appSecret := req.Attributes.Request.Http.Headers[xLibraAppSecret]
	if appId == "" || appSecret == "" {
		return srv.errToResponse(errUnauthenticated)
	}
	if app := getAppById(appId); app == nil || app.Secret != appSecret {
		return srv.errToResponse(errInvalidAppSecret)
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: []*corepb.HeaderValueOption{
					{
						Header: &corepb.HeaderValue{
							Key:   xLibraTrustedAppId,
							Value: appId,
						},
						Append: wrapperspb.Bool(false),
					},
				},
				HeadersToRemove: []string{xLibraAppId, xLibraAppSecret},
			},
		},
	}, nil
}

func (authServer) errToResponse(err error) (*authpb.CheckResponse, error) {
	s, ok := grpcstatus.FromError(err)
	if !ok {
		return nil, err
	}
	return &authpb.CheckResponse{
		Status: &status.Status{
			Code:    int32(s.Code()),
			Message: s.Message(),
		},
	}, nil
}
