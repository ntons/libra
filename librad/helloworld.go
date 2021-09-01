package main

import (
	"context"

	"google.golang.org/grpc/examples/helloworld/helloworld"
)

type greeterServer struct {
	helloworld.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(
	ctx context.Context, in *helloworld.HelloRequest) (
	*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}
