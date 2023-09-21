package auth

import (
	"context"
	"strings"

	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/common/envoy_service_auth_v3"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type handler func(context.Context, *authpb.CheckRequest) (
	*authpb.CheckResponse, error)

// 只支持V3版本的验证服务，V2版本缺少Header剔除功能，无法满足安全需求。
type authServer struct {
	authpb.UnimplementedAuthorizationServer

	handlers map[string]handler
}

func newAuthServer() *authServer {
	s := &authServer{}
	s.handlers = map[string]handler{
		L.XLibraAuthByAdminSecret:            s.checkAdminSecret,
		L.XLibraAuthBySecret:                 s.checkSecret,
		L.XLibraAuthByToken:                  s.checkToken,
		L.XLibraAuthBySecretOrToken:          s.checkSecretOrToken,
		L.XLibraAuthBySecretAndOptionalToken: s.checkSecretAndOptionalToken,
	}
	return s
}

func (srv authServer) Check(
	ctx context.Context, req *authpb.CheckRequest) (
	resp *authpb.CheckResponse, err error) {
	//log.Debugf("Auth.Check|%v", req)

	// 优先使用route上的配置的校验方式
	authBy, ok := req.Attributes.ContextExtensions[L.XLibraAuthBy]
	if !ok {
		// 如果route上没有配置，使用默认配置
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warnf("request miss metadata")
			return errResponse(errInvalidMetadata)
		}
		if v := md.Get(L.XLibraAuthBy); len(v) != 1 {
			log.Warnf("invalid auth-by metadata")
			return errResponse(errInvalidMetadata)
		} else if authBy = v[0]; authBy == "" {
			log.Warnf("invalid auth-by metadata")
			return errResponse(errInvalidMetadata)
		}
	}

	handler, ok := srv.handlers[authBy]
	if !ok {
		log.Warnf("unknown auth-by|%v", authBy)
		return errResponse(errUnauthenticated)
	}
	if resp, err = handler(ctx, req); err != nil {
		log.Warnf("auth check fail|%v", err)
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
