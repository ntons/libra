package registry

import (
	"encoding/json"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/common/util"
)

func newError(code codes.Code, msg interface{}) error {
	switch msg := msg.(type) {
	case string:
		return status.Errorf(code, msg)
	case proto.Message:
		if b, err := protojson.Marshal(msg); err != nil {
			return status.Errorf(code, "%v", msg)
		} else {
			return status.Errorf(code, util.BytesToString(b))
		}
	default:
		if b, err := json.Marshal(msg); err != nil {
			return status.Errorf(code, "%v", msg)
		} else {
			return status.Errorf(code, util.BytesToString(b))
		}
	}
}

func newUnauthenticatedError(msg interface{}) error {
	return newError(codes.Unauthenticated, msg)
}
func newNotFoundError(msg interface{}) error {
	return newError(codes.NotFound, msg)
}
func newAlreadyExistsError(msg interface{}) error {
	return newError(codes.AlreadyExists, msg)
}
func newInvalidArgumentError(msg interface{}) error {
	return newError(codes.InvalidArgument, msg)
}
func newInternalError(msg interface{}) error {
	return newError(codes.Internal, msg)
}
func newUnavailableError(msg interface{}) error {
	return newError(codes.Unavailable, msg)
}
func newPermissionDeniedError(msg interface{}) error {
	return newError(codes.PermissionDenied, msg)
}

func newErrorDetail(code v1pb.ErrorCode, data proto.Message) *v1pb.ErrorDetail {
	r := &v1pb.ErrorDetail{Code: code}
	r.Data, _ = anypb.New(data)
	return r
}

var (
	// Unauthenticated
	errUnauthenticated    = newUnauthenticatedError("unauthenticated")
	errLoginRequired      = newUnauthenticatedError("login required")
	errInvalidToken       = newUnauthenticatedError("invalid token")
	errInvalidAppSecret   = newUnauthenticatedError("invalid app secret")
	errInvalidAdminSecret = newUnauthenticatedError("invalid admin secret")

	errMismatchedAppSecretAndToken = newUnauthenticatedError("mismatched app secret and token")

	// NotFound
	errAppIdNotFound  = newNotFoundError("app id not found")
	errUserNotFound   = newNotFoundError("user not found")
	errRoleNotFound   = newNotFoundError("role not found")
	errAcctIdNotFound = newNotFoundError("acct id not found")

	// AlreadyExists
	errRoleAlreadyExists = newAlreadyExistsError("role already exists")
	errAcctAlreadyExists = newAlreadyExistsError("acct already exists")

	// InvalidArgument
	errInvalidNonce     = newInvalidArgumentError("invalid nonce")
	errInvalidTimestamp = newInvalidArgumentError("invalid timestamp")
	errInvalidState     = newInvalidArgumentError("invalid state")
	errInvalidSignature = newInvalidArgumentError("invalid signature")
	errInvalidAppId     = newInvalidArgumentError("invalid app id")
	errInvalidMetadata  = newInvalidArgumentError("invalid metadata")
	errInvalidAcctId    = newInvalidArgumentError("invalid acct id")
	errMetadataTooLarge = newInvalidArgumentError("metadata too large")

	// Internal
	errMalformedUserId   = newInternalError("malformed user id")
	errMalformedRoleId   = newInternalError("malformed role id")
	errMalformedSessData = newInternalError("malformed session data")

	// Unavailable
	errDatabaseUnavailable = newUnavailableError("database unavailable")

	// PermissionDenied
	errPermissionDenied = newPermissionDeniedError("permission denied")
)
