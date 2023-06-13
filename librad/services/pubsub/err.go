package pubsub

import (
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/ntons/libra/librad/common/util"
)

var (
	errUnauthenticated = status.Errorf(codes.Unauthenticated, "unauthenticated")
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

func newNotFoundError(msg interface{}) error {
	return newError(codes.NotFound, msg)
}
func newAlreadyExistsError(msg interface{}) error {
	return newError(codes.AlreadyExists, msg)
}
func newUnavailableError(msg interface{}) error {
	return newError(codes.Unavailable, msg)
}
func newInternalError(msg interface{}) error {
	return newError(codes.Internal, msg)
}
func newInvalidArgumentError(msg interface{}) error {
	return newError(codes.InvalidArgument, msg)
}
