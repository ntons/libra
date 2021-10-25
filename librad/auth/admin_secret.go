package auth

import (
	"context"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/auth/envoy_service_auth_v3"
	"github.com/ntons/libra/librad/db"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

func (srv authServer) checkAdminSecret(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	admId := req.Attributes.Request.Http.Headers[L.XLibraAdminId]
	admSecret := req.Attributes.Request.Http.Headers[L.XLibraAdminSecret]
	if admId == "" || admSecret == "" {
		return errResponse(errUnauthenticated)
	}

	if adm := db.FindAdminById(admId); adm == nil || adm.Secret != admSecret {
		return errResponse(errInvalidAdminSecret)
	} else if !adm.IsPermitted(req.Attributes.Request.Http.Path) {
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
				Key:   L.XLibraTrustedAdminId,
				Value: admId,
			},
		},
	}
	return &authpb.CheckResponse{
		Status: &statuspb.Status{},
		HttpResponse: &authpb.CheckResponse_OkResponse{
			OkResponse: &authpb.OkHttpResponse{
				Headers: headers,
				HeadersToRemove: []string{
					L.XLibraAdminId,
					L.XLibraAdminSecret,
				},
			},
		},
	}, nil
}
