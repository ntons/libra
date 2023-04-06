package auth

import (
	"context"

	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/common/envoy_service_auth_v3"
)

func (srv authServer) checkSecretOrToken(
	ctx context.Context, req *authpb.CheckRequest) (
	_ *authpb.CheckResponse, err error) {
	var resp1, resp2 *authpb.CheckResponse
	if _, ok := req.Attributes.Request.Http.Headers[L.XLibraAppSecret]; ok {
		resp1, err = srv.checkSecret(ctx, req)
		if err != nil || resp1.GetOkResponse() == nil {
			return resp1, err
		}
	}
	if _, ok := req.Attributes.Request.Http.Headers[L.XLibraToken]; ok {
		resp2, err = srv.checkToken(ctx, req)
		if err != nil || resp2.GetOkResponse() == nil {
			return resp2, err
		}
	}
	if resp1 == nil {
		if resp2 == nil {
			return errResponse(errUnauthenticated)
		} else {
			return resp2, nil
		}
	} else {
		if resp2 == nil {
			return resp1, nil
		} else {
			return mergeResponse(resp1, resp2)
		}
	}
}
