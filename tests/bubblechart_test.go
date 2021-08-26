package tests

import (
	"testing"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func TestBubbleChart(t *testing.T) {
	conn, err := DialApi()
	if err != nil {
		t.Fatalf("failed to dail: %v", err)
	}
	api := v1pb.NewBubbleChartClient(conn)
}
