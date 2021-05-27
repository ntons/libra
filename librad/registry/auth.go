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
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	log "github.com/ntons/log-go"
)

const (
	xAuthByToken  = "token"
	xAuthBySecret = "secret"
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
	var authBy string
	for key, val := range req.Attributes.Request.Http.Headers {
		// 不允许请求中带有x-libra-trusted-头
		if strings.HasPrefix(key, xLibraTrustedPrefix) {
			return srv.errToResponse(errInvalidMetadata)
		}
		if key == xLibraAuthBy {
			authBy = val
		}
	}
	switch authBy {
	case xAuthByToken:
		return srv.checkToken(ctx, req)
	case xAuthBySecret:
		return srv.checkSecret(ctx, req)
	default:
		// 没有任何可用凭证
		return srv.errToResponse(errUnauthenticated)
	}
}

func (srv authServer) checkToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.CheckToken|%v", req)
	token := req.Attributes.Request.Http.Headers[xLibraToken]
	if token == "" {
		return srv.errToResponse(errUnauthenticated)
	}
	sess, err := checkToken(ctx, token)
	if err != nil {
		return srv.errToResponse(err)
	}

	headers := []*corepb.HeaderValueOption{}
	if sess.AppId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedAppId,
				Value: sess.AppId,
			},
		})
	}
	if sess.UserId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedUserId,
				Value: sess.UserId,
			},
		})
	}
	if sess.RoleId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedRoleId,
				Value: sess.RoleId,
			},
		}, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedRoleIndex,
				Value: fmt.Sprintf("%d", sess.RoleIndex),
			},
		})
	}
	for _, header := range headers {
		header.Append = wrapperspb.Bool(false)
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers:         headers,
				HeadersToRemove: []string{xLibraToken},
			},
		},
	}, nil
}

func (srv authServer) checkSecret(
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

// 获取可信数据
func getTrustedFromContext(ctx context.Context) (appId, userId string, ok bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	if v := md.Get(xLibraTrustedAppId); len(v) != 1 || v[0] == "" {
		return "", "", false
	} else {
		appId = v[0]
	}
	if v := md.Get(xLibraTrustedUserId); len(v) != 1 || v[0] == "" {
		return "", "", false
	} else {
		userId = v[0]
	}
	return
}
