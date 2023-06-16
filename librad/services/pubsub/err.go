package pubsub

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errUnauthenticated = status.Errorf(codes.Unauthenticated, "unauthenticated")
)

func newInvalidArgumentError(format string, a ...interface{}) error {
	return status.Errorf(codes.InvalidArgument, format, a...)
}
