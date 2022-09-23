package tests

import (
	"google.golang.org/grpc"
)

const (
	AppId     = "libra-tests"
	AppSecret = "80e8789f2d5bf2ab631b83c925c9be0e"
)

func DialToApi() (conn *grpc.ClientConn, err error) {
	return grpc.Dial(
		"127.0.0.1:30081",
		grpc.WithInsecure(),
	)
}

func DialToEdge() (conn *grpc.ClientConn, err error) {
	return grpc.Dial(
		"127.0.0.1:30080",
		grpc.WithInsecure(),
	)
}
