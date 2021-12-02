package rpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// AppId错误
	InvalidAppSecretError = NewUnauthenticatedError("Invalid app secret")
)

func NewInvalidArgumentError(format string, args ...interface{}) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}
func NewInternalError(format string, args ...interface{}) error {
	return status.Errorf(codes.Internal, format, args...)
}
func NewUnavailableError(format string, args ...interface{}) error {
	return status.Errorf(codes.Unavailable, format, args...)
}
func NewUnauthenticatedError(format string, args ...interface{}) error {
	return status.Errorf(codes.Unauthenticated, format, args...)
}
func NewNotFoundError(format string, args ...interface{}) error {
	return status.Errorf(codes.NotFound, format, args...)
}
func NewAlreadyExistsError(format string, args ...interface{}) error {
	return status.Errorf(codes.AlreadyExists, format, args...)
}
