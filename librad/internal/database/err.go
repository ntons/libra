package database

import (
	"github.com/ntons/distlock"
	"github.com/ntons/remon"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errInvalidArgument = status.Errorf(codes.InvalidArgument, "invalid argument")
)

func remonerror(err error) error {
	code := codes.Internal
	if err == remon.ErrNotFound {
		code = codes.NotFound
	} else if err == remon.ErrAlreadyExists {
		code = codes.AlreadyExists
	}
	return status.Errorf(code, "remon: %s", err)
}

func distlockerror(err error) error {
	code := codes.Internal
	if err == distlock.ErrNotObtained {
		code = codes.Unavailable
	}
	return status.Errorf(code, "distlock: %s", err)
}

func protoerror(err error) error {
	code := codes.Internal
	return status.Errorf(code, "proto: %s", err)
}
