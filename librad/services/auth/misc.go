package auth

import (
	typepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	L "github.com/ntons/libra-go"
	authpb "github.com/ntons/libra/librad/common/envoy_service_auth_v3"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mergeResponse(
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
		return errResponse(errMismatchedAppSecretAndToken)
	}

	return &authpb.CheckResponse{
		Status: &statuspb.Status{},
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

func errResponse(err error) (*authpb.CheckResponse, error) {
	s, ok := status.FromError(err)
	if !ok {
		return nil, err
	}
	return &authpb.CheckResponse{
		Status: &statuspb.Status{
			Code:    int32(s.Code()),
			Message: s.Message(),
		},
		HttpResponse: &authpb.CheckResponse_DeniedResponse{
			DeniedResponse: &authpb.DeniedHttpResponse{
				Status: &typepb.HttpStatus{
					Code: httpStatusFromCode(s.Code()),
				},
			},
		},
	}, nil
}

func httpStatusFromCode(code codes.Code) typepb.StatusCode {
	switch code {
	case codes.OK:
		return typepb.StatusCode_OK
	case codes.Canceled:
		return typepb.StatusCode_RequestTimeout
	case codes.Unknown:
		return typepb.StatusCode_InternalServerError
	case codes.InvalidArgument:
		return typepb.StatusCode_BadRequest
	case codes.DeadlineExceeded:
		return typepb.StatusCode_GatewayTimeout
	case codes.NotFound:
		return typepb.StatusCode_NotFound
	case codes.AlreadyExists:
		return typepb.StatusCode_Conflict
	case codes.PermissionDenied:
		return typepb.StatusCode_Forbidden
	case codes.Unauthenticated:
		return typepb.StatusCode_Unauthorized
	case codes.ResourceExhausted:
		return typepb.StatusCode_TooManyRequests
	case codes.FailedPrecondition:
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return typepb.StatusCode_BadRequest
	case codes.Aborted:
		return typepb.StatusCode_Conflict
	case codes.OutOfRange:
		return typepb.StatusCode_BadRequest
	case codes.Unimplemented:
		return typepb.StatusCode_NotImplemented
	case codes.Internal:
		return typepb.StatusCode_InternalServerError
	case codes.Unavailable:
		return typepb.StatusCode_ServiceUnavailable
	case codes.DataLoss:
		return typepb.StatusCode_InternalServerError
	default:
		return typepb.StatusCode_InternalServerError
	}
}
