package registry

import (
	"context"
	"fmt"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authpb "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
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

// 只支持V3版本的验证服务，V2版本缺少Header剔除功能，无法满足安全需求。
type authServer struct {
	authpb.UnimplementedAuthorizationServer
}

func newAuthServer() *authServer { return &authServer{} }

func (srv authServer) Check(
	ctx context.Context, req *authpb.CheckRequest) (
	res *authpb.CheckResponse, err error) {
	log.Debugf("Auth.Check|%v", req)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return srv.errToResponse(errInvalidMetadata)
	}

	var authBy string
	if v := md.Get(xLibraAuthBy); len(v) != 1 {
		return srv.errToResponse(errInvalidMetadata)
	} else if authBy = v[0]; authBy == "" {
		return srv.errToResponse(errInvalidMetadata)
	}

	toRemoveHeaders := make(map[string]struct{})
	for key := range req.Attributes.Request.Http.Headers {
		// 去掉请求中带的x-libra-trusted-
		if strings.HasPrefix(key, xLibraTrustedPrefix) {
			toRemoveHeaders[key] = struct{}{}
		}
	}

	switch authBy {
	case xAuthByToken:
		res, err = srv.checkToken(ctx, req)
	case xAuthBySecret:
		res, err = srv.checkSecret(ctx, req)
	default:
		// 没有任何可用凭证
		return srv.errToResponse(errUnauthenticated)
	}
	if err != nil {
		return
	}

	if x := res.GetOkResponse(); x != nil {
		for _, e := range x.Headers {
			delete(toRemoveHeaders, e.GetHeader().GetKey())
		}
		for key := range toRemoveHeaders {
			x.HeadersToRemove = append(x.HeadersToRemove, key)
		}
	}
	return
}

func (srv authServer) checkToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	log.Debugf("Auth.CheckToken|%v", req)
	token := req.Attributes.Request.Http.Headers[xLibraToken]
	if token == "" {
		return srv.errToResponse(errUnauthenticated)
	}

	var sess *xSess
	if sess, err = checkToken(ctx, token); err != nil {
		return srv.errToResponse(err)
	} else if !sess.app.isPermitted(req.Attributes.Request.Http.Path) {
		return srv.errToResponse(errPermissionDenied)
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
	if sess.Data.RoleId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedRoleId,
				Value: sess.Data.RoleId,
			},
		}, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   xLibraTrustedRoleIndex,
				Value: fmt.Sprintf("%d", sess.Data.RoleIndex),
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
	if app := dbApps.findById(appId); app == nil || app.Secret != appSecret {
		return srv.errToResponse(errInvalidAppSecret)
	} else if !app.isPermitted(req.Attributes.Request.Http.Path) {
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
		HttpResponse: &authpb.CheckResponse_DeniedResponse{
			DeniedResponse: &authpb.DeniedHttpResponse{
				Status: &typepb.HttpStatus{
					Code: typepb.StatusCode_Unauthorized,
				},
				//Body: fmt.Sprintf("%d, %s", s.Code(), s.Message()),
			},
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
