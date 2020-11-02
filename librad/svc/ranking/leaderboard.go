package ranking

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/libra-go/api/v1"
	rkgo "github.com/ntons/ranking-go"
)

type leaderboard struct {
	v1.UnimplementedLeaderboardServer
	cli rkgo.Client
}

func newLeaderboard(uri string) (lb *leaderboard, err error) {
	ro, err := redis.ParseURL(uri)
	if err != nil {
		return
	}
	return &leaderboard{cli: rkgo.New(redis.NewClient(ro))}, nil
}

func (lb *leaderboard) SetScore(
	ctx context.Context, req *v1.LeaderboardSetScoreRequest) (
	resp *v1.LeaderboardSetScoreResponse, err error) {
	if err = lb.get(req).SetScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboard) IncrScore(
	ctx context.Context, req *v1.LeaderboardIncrScoreRequest) (
	resp *v1.LeaderboardIncrScoreResponse, err error) {
	if err = lb.get(req).IncrScore(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboard) GetRange(
	ctx context.Context, req *v1.LeaderboardGetRangeRequest) (
	resp *v1.LeaderboardGetRangeResponse, err error) {
	entries, err := lb.get(req).GetRange(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (lb *leaderboard) GetById(
	ctx context.Context, req *v1.LeaderboardGetByIdRequest) (
	resp *v1.LeaderboardGetByIdResponse, err error) {
	entries, err := lb.get(req).GetById(ctx, req.Ids...)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (lb *leaderboard) RemoveById(
	ctx context.Context, req *v1.LeaderboardRemoveByIdRequest) (
	resp *v1.LeaderboardRemoveByIdResponse, err error) {
	if err = lb.get(req).RemoveById(ctx, req.Ids...); err != nil {
		return
	}
	return
}

func (lb *leaderboard) SetInfo(
	ctx context.Context, req *v1.LeaderboardSetInfoRequest) (
	resp *v1.LeaderboardSetInfoResponse, err error) {
	if err = lb.get(req).SetInfo(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (lb *leaderboard) get(req request) rkgo.Leaderboard {
	return lb.cli.GetLeaderboard(
		fromChartKey(req.GetKey()), fromChartOptions(req.GetOptions())...)
}
