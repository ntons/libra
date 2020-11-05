package ranking

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/libra-go/api/v1"
	rgo "github.com/ntons/ranking-go"
)

type leaderboardServer struct {
	v1.UnimplementedLeaderboardServer
	cli rgo.Client
}

func newLeaderboardServer(uri string) (lb *leaderboardServer, err error) {
	ro, err := redis.ParseURL(uri)
	if err != nil {
		return
	}
	return &leaderboardServer{cli: rgo.New(redis.NewClient(ro))}, nil
}

func (lb *leaderboardServer) SetScore(
	ctx context.Context, req *v1.LeaderboardSetScoreRequest) (
	resp *v1.LeaderboardSetScoreResponse, err error) {
	if err = lb.get(req).SetScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboardServer) IncrScore(
	ctx context.Context, req *v1.LeaderboardIncrScoreRequest) (
	resp *v1.LeaderboardIncrScoreResponse, err error) {
	if err = lb.get(req).IncrScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboardServer) GetRange(
	ctx context.Context, req *v1.LeaderboardGetRangeRequest) (
	resp *v1.LeaderboardGetRangeResponse, err error) {
	entries, err := lb.get(req).GetRange(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (lb *leaderboardServer) GetById(
	ctx context.Context, req *v1.LeaderboardGetByIdRequest) (
	resp *v1.LeaderboardGetByIdResponse, err error) {
	entries, err := lb.get(req).GetById(ctx, req.Ids...)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (lb *leaderboardServer) RemoveById(
	ctx context.Context, req *v1.LeaderboardRemoveByIdRequest) (
	resp *v1.LeaderboardRemoveByIdResponse, err error) {
	if err = lb.get(req).RemoveById(ctx, req.Ids...); err != nil {
		return
	}
	return
}

func (lb *leaderboardServer) SetInfo(
	ctx context.Context, req *v1.LeaderboardSetInfoRequest) (
	resp *v1.LeaderboardSetInfoResponse, err error) {
	if err = lb.get(req).SetInfo(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboardServer) get(req request) rgo.Leaderboard {
	return lb.cli.GetLeaderboard(
		fromChartKey(req.GetKey()), fromChartOptions(req.GetOptions())...)
}
