package main

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type Health struct {
	grpc_health_v1.UnimplementedHealthServer
	quit   chan struct{}
	closed int32
}

func NewHealth() *Health {
	return &Health{quit: make(chan struct{}, 1)}
}

func (h *Health) Close() {
	if atomic.CompareAndSwapInt32(&h.closed, 0, 1) {
		close(h.quit)
	}
}

func (h *Health) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (resp *grpc_health_v1.HealthCheckResponse, err error) {
	var status = grpc_health_v1.HealthCheckResponse_UNKNOWN
	select {
	case <-h.quit:
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		status = grpc_health_v1.HealthCheckResponse_SERVING
	}
	resp = &grpc_health_v1.HealthCheckResponse{Status: status}
	return
}

func (h *Health) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) (err error) {
	var status = grpc_health_v1.HealthCheckResponse_UNKNOWN
	select {
	case <-h.quit:
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		stream.Send(&grpc_health_v1.HealthCheckResponse{Status: status})
	default:
		status = grpc_health_v1.HealthCheckResponse_SERVING
		stream.Send(&grpc_health_v1.HealthCheckResponse{Status: status})
	}
	select {
	case <-h.quit:
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		stream.Send(&grpc_health_v1.HealthCheckResponse{Status: status})
	case <-stream.Context().Done():
	}
	return
}
