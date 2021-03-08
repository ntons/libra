package tests

import (
	"testing"

	v1pb "github.com/ntons/libra-go/api/v1"
)

func TestLeaderboard(t *testing.T) {
	conn, err := DialToApi()
	if err != nil {
		t.Fatalf("failed to dail: %v", err)
	}
	api := v1pb.NewLeaderboardClient(conn)
}
