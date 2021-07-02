package registry

import (
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newError(code codes.Code, msg interface{}) error {
	switch msg := msg.(type) {
	case string:
		return status.Errorf(code, msg)
	default:
		if b, err := json.Marshal(msg); err != nil {
			return status.Errorf(code, "%v", msg)
		} else {
			return status.Errorf(code, string(b))
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

var (
	// Unauthenticated
	errUnauthenticated  = newUnauthenticatedError("unauthenticated")
	errLoginRequired    = newUnauthenticatedError("login required")
	errInvalidToken     = newUnauthenticatedError("invalid token")
	errInvalidAppSecret = newUnauthenticatedError("invalid app secret")

	errMismatchedAppSecretAndToken = newUnauthenticatedError("mismatched app secret and token")

	// NotFound
	errAppIdNotFound = newNotFoundError("app id not found")
	errUserNotFound  = newNotFoundError("user not found")
	errRoleNotFound  = newNotFoundError("role not found")

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
