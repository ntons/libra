package main

import (
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// alian serving status
	xStatusUnknown    = healthpb.HealthCheckResponse_UNKNOWN
	xStatusServing    = healthpb.HealthCheckResponse_SERVING
	xStatusNotServing = healthpb.HealthCheckResponse_NOT_SERVING
)

var (
	healthServer = health.NewServer()
)

// register health server to grpc server
func registerHealthServer() {
	healthpb.RegisterHealthServer(grpcServer, healthServer)
}

// shutdown health server
func shutdownHealthServer() {
	healthServer.Shutdown()
}

// set service serving status
func setStatus(name string, status healthpb.HealthCheckResponse_ServingStatus) {
	healthServer.SetServingStatus(name, status)
}
