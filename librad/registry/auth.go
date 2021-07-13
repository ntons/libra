package registry

import (
	"context"
	"fmt"
	"strings"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authpb "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	L "github.com/ntons/libra-go"
	log "github.com/ntons/log-go"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	if v := md.Get(L.XLibraAuthBy); len(v) != 1 {
		return srv.errToResponse(errInvalidMetadata)
	} else if authBy = v[0]; authBy == "" {
		return srv.errToResponse(errInvalidMetadata)
	}

	toRemoveHeaders := make(map[string]struct{})
	for key := range req.Attributes.Request.Http.Headers {
		// 去掉请求中自带的可信元数据
		if strings.HasPrefix(key, L.XLibraTrustedPrefix) {
			toRemoveHeaders[key] = struct{}{}
		}
	}

	switch authBy {
	case L.XLibraAuthByToken:
		res, err = srv.checkToken(ctx, req)
	case L.XLibraAuthBySecret:
		res, err = srv.checkSecret(ctx, req)
	//case C.XAuthBySecretAndOptionalToken:
	//	res, err = srv.checkSecretAndOptionalToken(ctx, req)
	default:
		// 没有任何可用凭证
		return srv.errToResponse(errUnauthenticated)
	}
	if err != nil {
		return
	}
	if okResp := res.GetOkResponse(); okResp != nil {
		okResp.Headers = append(okResp.Headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAuthBy,
				Value: authBy,
			},
			Append: wrapperspb.Bool(false),
		})
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
	//log.Debugf("Auth.CheckToken|%v", req)
	token := req.Attributes.Request.Http.Headers[L.XLibraToken]
	if token == "" {
		return srv.errToResponse(errUnauthenticated)
	}

	var sess *dbSess
	if sess, err = checkToken(ctx, token); err != nil {
		return srv.errToResponse(err)
	} else if !sess.app.isPermitted(req.Attributes.Request.Http.Path) {
		return srv.errToResponse(errPermissionDenied)
	}

	headers := []*corepb.HeaderValueOption{}
	if sess.AppId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAppId,
				Value: sess.AppId,
			},
			Append: wrapperspb.Bool(false),
		})
	}
	if sess.UserId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedUserId,
				Value: sess.UserId,
			},
			Append: wrapperspb.Bool(false),
		})
	}
	if sess.Data.RoleId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedRoleId,
				Value: sess.Data.RoleId,
			},
			Append: wrapperspb.Bool(false),
		}, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedRoleIndex,
				Value: fmt.Sprintf("%d", sess.Data.RoleIndex),
			},
			Append: wrapperspb.Bool(false),
		})
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers:         headers,
				HeadersToRemove: []string{L.XLibraToken},
			},
		},
	}, nil
}

func (srv authServer) checkSecret(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	//log.Debugf("Auth.CheckSecret|%v", req)
	appId := req.Attributes.Request.Http.Headers[L.XLibraAppId]
	appSecret := req.Attributes.Request.Http.Headers[L.XLibraAppSecret]
	if appId == "" || appSecret == "" {
		return srv.errToResponse(errUnauthenticated)
	}
	if app := findAppById(appId); app == nil || app.Secret != appSecret {
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
							Key:   L.XLibraTrustedAppId,
							Value: appId,
						},
						Append: wrapperspb.Bool(false),
					},
				},
				HeadersToRemove: []string{L.XLibraAppId, L.XLibraAppSecret},
			},
		},
	}, nil
}

//func (srv authServer) checkSecretAndOptionalToken(
//	ctx context.Context, req *authpb.CheckRequest) (
//	_ *authpb.CheckResponse, err error) {
//	if _, ok := req.Attributes.Request.Http.Headers[C.XLibraToken]; !ok {
//		// check secret only
//		return srv.checkSecret(ctx, req)
//	}
//	// check secret and token
//	resp1, err := srv.checkSecret(ctx, req)
//	if err != nil || resp1.GetOkResponse() == nil {
//		return resp1, err
//	}
//	resp2, err := srv.checkToken(ctx, req)
//	if err != nil || resp2.GetOkResponse() == nil {
//		return resp2, err
//	}
//	var appId1, appId2 string
//	for _, header := range resp1.GetOkResponse().Headers {
//		if header.GetHeader().GetKey() == C.XLibraTrustedAppId {
//			appId1 = header.GetHeader().GetValue()
//			break
//		}
//	}
//	for _, header := range resp2.GetOkResponse().Headers {
//		if header.GetHeader().GetKey() == C.XLibraTrustedAppId {
//			appId2 = header.GetHeader().GetValue()
//			break
//		}
//	}
//	if appId1 != appId2 {
//		return srv.errToResponse(errMismatchedAppSecretAndToken)
//	}
//	return resp2, nil
//}

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
