package ranking

import (
	"context"

	L "github.com/ntons/libra-go"
	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/ranking"
	"github.com/ntons/redis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type leaderboardServer struct {
	v1.UnimplementedLeaderboardServer
	cli ranking.Client
}

func newLeaderboardServer(cli redis.Client) *leaderboardServer {
	return &leaderboardServer{cli: ranking.New(cli)}
}

func (lb *leaderboardServer) SetScore(
	ctx context.Context, req *v1.LeaderboardSetScoreRequest) (
	resp *v1.LeaderboardSetScoreResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if err = lb.get(trusted.AppId, req).SetScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	resp = &v1.LeaderboardSetScoreResponse{}
	return
}

func (lb *leaderboardServer) IncrScore(
	ctx context.Context, req *v1.LeaderboardIncrScoreRequest) (
	resp *v1.LeaderboardIncrScoreResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if err = lb.get(trusted.AppId, req).IncrScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	resp = &v1.LeaderboardIncrScoreResponse{}
	return
}

func (lb *leaderboardServer) GetRange(
	ctx context.Context, req *v1.LeaderboardGetRangeRequest) (
	resp *v1.LeaderboardGetRangeResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	entries, err := lb.get(trusted.AppId, req).GetRange(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp = &v1.LeaderboardGetRangeResponse{
		Entries: toChartEntries(entries),
	}
	return
}

func (lb *leaderboardServer) GetById(
	ctx context.Context, req *v1.LeaderboardGetByIdRequest) (
	resp *v1.LeaderboardGetByIdResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	entries, err := lb.get(trusted.AppId, req).GetById(ctx, req.Ids...)
	if err != nil {
		return
	}
	resp = &v1.LeaderboardGetByIdResponse{
		Entries: toChartEntries(entries),
	}
	return
}

func (lb *leaderboardServer) RemoveById(
	ctx context.Context, req *v1.LeaderboardRemoveByIdRequest) (
	resp *v1.LeaderboardRemoveByIdResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if err = lb.get(trusted.AppId, req).RemoveById(ctx, req.Ids...); err != nil {
		return
	}
	resp = &v1.LeaderboardRemoveByIdResponse{}
	return
}

func (lb *leaderboardServer) SetInfo(
	ctx context.Context, req *v1.LeaderboardSetInfoRequest) (
	resp *v1.LeaderboardSetInfoResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if err = lb.get(trusted.AppId, req).SetInfo(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return &v1.LeaderboardSetInfoResponse{}, nil
}

func (lb *leaderboardServer) get(appId string, req request) ranking.Leaderboard {
	return lb.cli.GetLeaderboard(
		fromChartKey(appId, req.GetKey()),
		fromChartOptions(appId, req.GetOptions())...)
}
