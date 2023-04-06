package auth

import (
	"context"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/common/envoy_service_auth_v3"
	"github.com/ntons/libra/librad/db"
	log "github.com/ntons/log-go"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

func (srv authServer) checkSecret(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	appId := req.Attributes.Request.Http.Headers[L.XLibraAppId]
	appSecret := req.Attributes.Request.Http.Headers[L.XLibraAppSecret]
	if appId == "" || appSecret == "" {
		log.Warnf("auth by secret|invalid app id or secret|%s|%s",
			appId, appSecret)
		return errResponse(errUnauthenticated)
	}

	if app := db.FindAppById(appId); app == nil || app.Secret != appSecret {
		log.Warnf("auth by secret|invalid app id or secret|%s|%s",
			appId, appSecret)
		return errResponse(errInvalidAppSecret)
	} else if !app.IsPermitted(req.Attributes.Request.Http.Path) {
		log.Warnf("auth by secret|request path is not permitted|%s|%s",
			appId, req.Attributes.Request.Http.Path)
		return errResponse(errPermissionDenied)
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
		Status: &statuspb.Status{},
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
