package pubsub

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errUnauthenticated = status.Errorf(codes.Unauthenticated, "unauthenticated")
)

func newInvalidArgumentError(format string, a ...interface{}) error {
	return status.Errorf(codes.InvalidArgument, format, a...)
}
func newOutOfRangeError(format string, a ...interface{}) error {
	return status.Errorf(codes.OutOfRange, format, a...)
}
func newUnavailableError(format string, a ...interface{}) error {
	return status.Errorf(codes.Unavailable, format, a...)
}

func errHasPrefix(err error, prefix string) bool {
	return strings.HasPrefix(strings.TrimLeft(err.Error(), " "), prefix)
}
func isNoGroupError(err error) bool {
	return err != nil && errHasPrefix(err, "NOGROUP")
}
func isBusyGroupError(err error) bool {
	return err != nil && errHasPrefix(err, "BUSYGROUP")
}
func isCanceledError(err error) bool {
	return err == context.Canceled
}
