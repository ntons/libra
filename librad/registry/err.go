package registry

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newUnauthenticatedError(msg string) error {
	return status.Errorf(codes.Unauthenticated, msg)
}
func newNotFoundError(msg string) error {
	return status.Errorf(codes.NotFound, msg)
}
func newAlreadyExistsError(msg string) error {
	return status.Errorf(codes.AlreadyExists, msg)
}
func newInvalidArgumentError(msg string) error {
	return status.Errorf(codes.InvalidArgument, msg)
}
func newInternalError(msg string) error {
	return status.Errorf(codes.Internal, msg)
}
func newUnavailableError(msg string) error {
	return status.Errorf(codes.Unavailable, msg)
}
func newPermissionDeniedError(msg string) error {
	return status.Errorf(codes.PermissionDenied, msg)
}

var (
	// Unauthenticated
	errUnauthenticated  = newUnauthenticatedError("unauthenticated")
	errLoginRequired    = newUnauthenticatedError("login required")
	errInvalidToken     = newUnauthenticatedError("invalid token")
	errInvalidAppSecret = newUnauthenticatedError("invalid app secret")

	// NotFound
	errAppIdNotFound = newNotFoundError("app id not found")
	errUserNotFound  = newNotFoundError("user not found")
	errRoleNotFound  = newNotFoundError("role not found")

	// AlreadyExists
	errRoleAlreadyExists = newAlreadyExistsError("role already exists")

	// InvalidArgument
	errInvalidNonce     = newInvalidArgumentError("invalid nonce")
	errInvalidTimestamp = newInvalidArgumentError("invalid timestamp")
	errInvalidState     = newInvalidArgumentError("invalid state")
	errInvalidSignature = newInvalidArgumentError("invalid signature")
	errInvalidAppId     = newInvalidArgumentError("invalid app id")
	errInvalidMetadata  = newInvalidArgumentError("invalid metadata")

	// Internal
	errMalformedUserId   = newInternalError("malformed user id")
	errMalformedRoleId   = newInternalError("malformed role id")
	errMalformedSessData = newInternalError("malformed session data")

	// Unavailable
	errDatabaseUnavailable = newUnavailableError("database unavailable")

	// PermissionDenied
	errPermissionDenied = newPermissionDeniedError("permission denied")
)
