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
	resp *authpb.CheckResponse, err error) {
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

	switch authBy {
	case L.XLibraAuthByToken:
		resp, err = srv.checkToken(ctx, req)
	case L.XLibraAuthBySecret:
		resp, err = srv.checkSecret(ctx, req)
	case L.XLibraAuthBySecretAndOptionalToken:
		resp, err = srv.checkSecretAndOptionalToken(ctx, req)
	case L.XLibraAuthBySecretOrToken, L.XLibraAuthByTokenOrSecret:
		resp, err = srv.checkSecretOrToken(ctx, req)
	default:
		return srv.errToResponse(errUnauthenticated)
	}
	if err != nil {
		return
	}

	// 集中处理可信元数据的增减
	if okResp := resp.GetOkResponse(); okResp != nil {
		m := make(map[string]struct{})
		for _, header := range okResp.Headers {
			_, found := m[header.Header.Key]
			header.Append = wrapperspb.Bool(found)
			m[header.Header.Key] = struct{}{}
		}
		for key := range req.Attributes.Request.Http.Headers {
			if !strings.HasPrefix(key, L.XLibraTrustedPrefix) {
				continue
			}
			if _, found := m[key]; found {
				continue
			}
			okResp.HeadersToRemove = append(okResp.HeadersToRemove, key)
		}
	}

	return
}

func (srv authServer) checkToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
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

	headers := []*corepb.HeaderValueOption{
		{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAuthBy,
				Value: L.XLibraAuthByToken,
			},
		},
		{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAppId,
				Value: sess.AppId,
			},
		},
		{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedUserId,
				Value: sess.UserId,
			},
		},
	}
	if sess.Data.RoleId != "" {
		headers = append(headers, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedRoleId,
				Value: sess.Data.RoleId,
			},
		}, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedRoleIndex,
				Value: fmt.Sprintf("%d", sess.Data.RoleIndex),
			},
		})
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: headers,
				HeadersToRemove: []string{
					L.XLibraToken,
				},
			},
		},
	}, nil
}

func (srv authServer) checkSecret(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
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

	headers := []*corepb.HeaderValueOption{
		{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAuthBy,
				Value: L.XLibraAuthBySecret,
			},
		},
		{
			Header: &corepb.HeaderValue{
				Key:   L.XLibraTrustedAppId,
				Value: appId,
			},
		},
	}
	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: headers,
				HeadersToRemove: []string{
					L.XLibraAppId,
					L.XLibraAppSecret,
				},
			},
		},
	}, nil
}

func (srv authServer) checkSecretAndOptionalToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	if _, ok := req.Attributes.Request.Http.Headers[L.XLibraToken]; !ok {
		return srv.checkSecret(ctx, req)
	}

	var resp1, resp2 *authpb.CheckResponse
	if resp1, err = srv.checkSecret(ctx, req); err != nil {
		return resp1, err
	}
	if resp2, err = srv.checkToken(ctx, req); err != nil {
		return resp2, err
	}
	return srv.mergeResponse(resp1, resp2)
}

func (srv authServer) checkSecretOrToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	var resp1, resp2 *authpb.CheckResponse
	if _, ok := req.Attributes.Request.Http.Headers[L.XLibraAppSecret]; ok {
		if resp1, err = srv.checkSecret(ctx, req); err != nil {
			return resp1, err
		}
	}
	if _, ok := req.Attributes.Request.Http.Headers[L.XLibraToken]; ok {
		if resp2, err = srv.checkToken(ctx, req); err != nil {
			return resp2, err
		}
	}
	if resp1 == nil && resp2 == nil {
		return srv.errToResponse(errUnauthenticated)
	}
	if resp1 == nil && resp2 != nil {
		return resp2, nil
	}
	if resp1 != nil && resp2 == nil {
		return resp1, nil
	}
	return srv.mergeResponse(resp1, resp2)
}

func (srv authServer) mergeResponse(
	resp1, resp2 *authpb.CheckResponse) (_ *authpb.CheckResponse, err error) {
	okResp1 := resp1.GetOkResponse()
	okResp2 := resp2.GetOkResponse()

	var appId1, appId2 string
	for _, header := range okResp1.Headers {
		if header.GetHeader().GetKey() == L.XLibraTrustedAppId {
			appId1 = header.GetHeader().GetValue()
			break
		}
	}
	for i, header := range okResp2.Headers {
		if header.GetHeader().GetKey() == L.XLibraTrustedAppId {
			appId2 = header.GetHeader().GetValue()
			// 只保留1个AppId
			n := len(okResp2.Headers)
			if i < n-1 {
				okResp2.Headers[i] = okResp2.Headers[n-1]
			}
			okResp2.Headers = okResp2.Headers[:n-1]
			break
		}
	}
	if appId1 != appId2 {
		return srv.errToResponse(errMismatchedAppSecretAndToken)
	}

	return &authpb.CheckResponse{
		Status: &status.Status{Code: int32(code.Code_OK)},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: append(
					okResp1.Headers,
					okResp2.Headers...,
				),
				HeadersToRemove: append(
					okResp1.HeadersToRemove,
					okResp2.HeadersToRemove...,
				),
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
