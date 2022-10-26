package auth

import (
	"context"
	"fmt"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/auth/envoy_service_auth_v3"
	"github.com/ntons/libra/librad/db"
	log "github.com/ntons/log-go"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

func (srv authServer) checkToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	token := req.Attributes.Request.Http.Headers[L.XLibraToken]
	if token == "" {
		log.Warnf("auth by token|invalid token|%s", token)
		return errResponse(errUnauthenticated)
	}

	var sess *db.Sess
	if sess, err = db.CheckToken(ctx, token); err != nil {
		log.Warnf("auth by token|failed to check|%s|%v", token, err)
		return errResponse(err)
	} else if !sess.App.IsPermitted(req.Attributes.Request.Http.Path) {
		log.Warnf("auth by token|request path is not permitted|%s|%s",
			sess.App.Id, req.Attributes.Request.Http.Path)
		return errResponse(errPermissionDenied)
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
		Status: &statuspb.Status{},
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
