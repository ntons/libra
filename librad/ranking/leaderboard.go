package ranking

import (
	"context"
	"errors"

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

func (lb *leaderboardServer) Add(
	ctx context.Context, req *v1.LeaderboardAddRequest) (
	resp *v1.LeaderboardAddResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	x := lb.get(trusted.AppId, req)
	if err = x.Add(ctx, fromChartEntries(req.Entries)...); err != nil {
		if errors.Is(err, redchart.ErrChartFull) {
			err = status.Errorf(codes.OutOfRange, err.Error())
		}
		return
	}

	var ids = make([]string, 0, len(req.Entries))
	for _, e := range req.Entries {
		ids = append(ids, e.Id)
	}
	entries, err := x.GetById(ctx, ids...)
	if err != nil {
		return
	}

	resp = &v1.LeaderboardAddResponse{
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
	if err = lb.get(trusted.AppId, req).Set(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		if errors.Is(err, redchart.ErrChartFull) {
			err = status.Errorf(codes.OutOfRange, err.Error())
		}
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
	if err = lb.get(trusted.AppId, req).Incr(
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
	entries, err := lb.get(trusted.AppId, req).GetByRank(ctx, req.Offset, req.Count)
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

func (lb *leaderboardServer) RandByScore(
	ctx context.Context, req *v1.LeaderboardRandByScoreRequest) (
	resp *v1.LeaderboardRandByScoreResponse, err error) {
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

	entries, err := lb.get(trusted.AppId, req).RandByScore(ctx, args...)
	if err != nil {
		return
	}
	return &v1.LeaderboardRandByScoreResponse{
		Entries: toChartEntries(entries),
	}, nil
}

func (lb *leaderboardServer) get(appId string, req request) redchart.Leaderboard {
	return lb.cli.GetLeaderboard(
		fromChartKey(appId, req.GetKey()),
		fromChartOptions(appId, req.GetOptions())...)
}
