package auth

import (
	"context"

	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/auth/envoy_service_auth_v3"
)

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
	return mergeResponse(resp1, resp2)
}
