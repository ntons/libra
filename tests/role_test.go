package tests

import (
	"testing"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func TestRole(t *testing.T) {
	conn, err := DialToEdge()
	if err != nil {
		t.Fatalf("failed to dail: %v", err)
	}
	api := v1pb.NewRoleClient(conn)
}
