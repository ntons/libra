package database

import (
	"github.com/ntons/redlock"
	"github.com/ntons/redmon"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errInvalidArgument = status.Errorf(codes.InvalidArgument, "invalid argument")
	errTimeoutTooLong  = status.Errorf(codes.InvalidArgument, "timeout too long")
	errUnauthenticated = status.Errorf(codes.Unauthenticated, "unauthenticated")
	errTooLarge        = status.Errorf(codes.Unauthenticated, "too large")
)

func fromRedmonError(err error) error {
	code := codes.Internal
	if err == redmon.ErrNotExists {
		code = codes.NotFound
	} else if err == redmon.ErrAlreadyExists {
		code = codes.AlreadyExists
	}
	return status.Errorf(code, "redmon: %s", err)
}

func fromRedlockError(err error) error {
	code := codes.Internal
	if err == redlock.ErrNotObtained {
		code = codes.FailedPrecondition
	}
	return status.Errorf(code, "redlock: %s", err)
}

func fromProtoError(err error) error {
	code := codes.Internal
	return status.Errorf(code, "proto: %s", err)
}
