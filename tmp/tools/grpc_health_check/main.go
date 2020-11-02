// 健康检查工具

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func Check(c grpc_health_v1.HealthClient) {
	req := &grpc_health_v1.HealthCheckRequest{Service: "xxx"}
	resp, err := c.Check(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "remote status: %v\n", resp.Status)
}

func Watch(c grpc_health_v1.HealthClient) {
	req := grpc_health_v1.HealthCheckRequest{Service: "xxx"}
	stream, err := c.Watch(context.Background(), &req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to watch: %v\n", err)
		os.Exit(1)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to recv: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "remote status: %v\n", resp.Status)
	}
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s address(host:port) check|watch\n", os.Args[0])
}

func main() {
	if len(os.Args) != 3 {
		Usage()
		os.Exit(1)
	}
	if s := strings.ToLower(os.Args[2]); s != "check" && s != "watch" {
		Usage()
		os.Exit(1)
	}
	conn, err := grpc.Dial(os.Args[1], grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dail: %v\n", err)
		os.Exit(1)
	}
	c := grpc_health_v1.NewHealthClient(conn)
	switch os.Args[2] {
	case "check":
		Check(c)
	case "watch":
		Watch(c)
	}
}
