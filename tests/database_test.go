package tests

import (
	"testing"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func TestDatabase(t *testing.T) {
	conn, err := DialToApi()
	if err != nil {
		t.Fatalf("failed to dail: %v", err)
	}
	api := v1pb.NewDatabaseClient(conn)
}
