package registry

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Unauthenticated
	errUnauthenticated  = status.Errorf(codes.Unauthenticated, "unauthenticated")
	errLoginRequired    = status.Errorf(codes.Unauthenticated, "login required")
	errInvalidToken     = status.Errorf(codes.Unauthenticated, "invalid token")
	errInvalidTicket    = status.Errorf(codes.Unauthenticated, "invalid ticket")
	errInvalidAppSecret = status.Errorf(codes.Unauthenticated, "invalid app secret")
	// NotFound
	errAppIdNotFound = status.Errorf(codes.NotFound, "app id not found")
	errUserNotFound  = status.Errorf(codes.NotFound, "user not found")
	errRoleNotFound  = status.Errorf(codes.NotFound, "role not found")
	// AlreadyExists
	errRoleAlreadyExists = status.Errorf(codes.AlreadyExists, "role already exists")
	// InvalidArgument
	errInvalidNonce     = status.Errorf(codes.InvalidArgument, "invalid nonce")
	errInvalidState     = status.Errorf(codes.InvalidArgument, "invalid state")
	errInvalidSignature = status.Errorf(codes.InvalidArgument, "invalid signature")
	errInvalidAppId     = status.Errorf(codes.InvalidArgument, "invalid app id")
	errInvalidMetadata  = status.Errorf(codes.InvalidArgument, "invalid metadata")
	// Internal
	errMalformedUserId = status.Errorf(codes.Internal, "malformed user id")
	errMalformedRoleId = status.Errorf(codes.Internal, "malformed role id")
	// Unavailable
	errDatabaseUnavailable = status.Errorf(codes.Unavailable, "database unavailable")
	// PermissionDenied
	errPermissionDenied = status.Errorf(codes.PermissionDenied, "permission denied")
)
