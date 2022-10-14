package ranking

import (
	"context"

	L "github.com/ntons/libra-go"
	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redchart"
	"github.com/ntons/redis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type leaderboardServer struct {
	v1.UnimplementedLeaderboardServer
	cli redchart.Client
}

func newLeaderboardServer(cli redis.Client) *leaderboardServer {
	return &leaderboardServer{cli: redchart.New(cli)}
}

func (lb *leaderboardServer) Touch(
	ctx context.Context, req *v1.LeaderboardTouchRequest) (
	resp *v1.LeaderboardTouchResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	x := lb.get(trusted.AppId, req)
	if err = x.Touch(ctx); err != nil {
		return
	}

	return &v1.LeaderboardTouchResponse{}, nil
}

func (lb *leaderboardServer) Set(
	ctx context.Context, req *v1.LeaderboardSetRequest) (
	resp *v1.LeaderboardSetResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	entries, err := lb.get(trusted.AppId, req).Set(
		ctx,
		fromChartEntries(req.Entries),
		redchart.WithSetOnlyAdd(req.OnlyAdd),
		redchart.WithSetOnlyUpdate(req.OnlyUpdate),
		redchart.WithSetIncrBy(req.IncrBy),
	)
	if err != nil {
		return
	}
	resp = &v1.LeaderboardSetResponse{
		Entries: toChartEntries(entries),
	}
	return
}

func (lb *leaderboardServer) SetScore(
	ctx context.Context, req *v1.LeaderboardSetScoreRequest) (
	resp *v1.LeaderboardSetScoreResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if _, err = lb.get(trusted.AppId, req).Set(
		ctx, fromChartEntries(req.Entries)); err != nil {
		return
	}
	resp = &v1.LeaderboardSetScoreResponse{}
	return
}

func (lb *leaderboardServer) GetRange(
	ctx context.Context, req *v1.LeaderboardGetRangeRequest) (
	resp *v1.LeaderboardGetRangeResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	entries, err := lb.get(trusted.AppId, req).GetByRank(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp = &v1.LeaderboardGetRangeResponse{
		Entries: toChartEntries(entries),
	}
	return
}

func (lb *leaderboardServer) GetByRank(
	ctx context.Context, req *v1.LeaderboardGetByRankRequest) (
	resp *v1.LeaderboardGetByRankResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	entries, err := lb.get(trusted.AppId, req).GetByRank(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp = &v1.LeaderboardGetByRankResponse{
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
	entries, err := lb.get(trusted.AppId, req).GetById(ctx, req.Ids)
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
	if err = lb.get(trusted.AppId, req).RemoveById(ctx, req.Ids); err != nil {
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
		ctx, fromChartEntries(req.Entries)); err != nil {
		return
	}
	return &v1.LeaderboardSetInfoResponse{}, nil
}

func (lb *leaderboardServer) GetByScore(
	ctx context.Context, req *v1.LeaderboardGetByScoreRequest) (
	resp *v1.LeaderboardGetByScoreResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	args := make([]redchart.RandByScoreArg, 0, len(req.Intervals))
	for _, v := range req.Intervals {
		args = append(args, redchart.RandByScoreArg{
			Min:   v.Min,
			Max:   v.Max,
			Count: int(v.Count),
		})
	}

	entries, err := lb.get(trusted.AppId, req).RandByScore(ctx, args)
	if err != nil {
		return
	}
	return &v1.LeaderboardGetByScoreResponse{
		Entries: toChartEntries(entries),
	}, nil
}

func (lb *leaderboardServer) get(appId string, req request) redchart.Leaderboard {
	return lb.cli.GetLeaderboard(
		fromChartKey(appId, req.GetKey()),
		fromChartOptions(appId, req.GetOptions())...)
}
