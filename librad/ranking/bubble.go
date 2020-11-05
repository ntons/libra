package ranking

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/libra-go/api/v1"
	rgo "github.com/ntons/ranking-go"
)

type bubbleServer struct {
	v1.UnimplementedBubbleServer
	cli rgo.Client
}

func newBubbleServer(uri string) (bb *bubbleServer, err error) {
	ro, err := redis.ParseURL(uri)
	if err != nil {
		return
	}
	return &bubbleServer{cli: rgo.New(redis.NewClient(ro))}, nil
}

func (bb *bubbleServer) Append(
	ctx context.Context, req *v1.BubbleAppendRequest) (
	resp *v1.BubbleAppendResponse, err error) {
	if err = bb.get(req).Append(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (bb *bubbleServer) SwapById(
	ctx context.Context, req *v1.BubbleSwapByIdRequest) (
	resp *v1.BubbleSwapByIdResponse, err error) {
	if err = bb.get(req).SwapById(ctx, req.Id0, req.Id1); err != nil {
		return
	}
	return
}

func (bb *bubbleServer) SwapByRank(
	ctx context.Context, req *v1.BubbleSwapByRankRequest) (
	resp *v1.BubbleSwapByRankResponse, err error) {
	if err = bb.get(req).SwapByRank(ctx, req.Rank0, req.Rank1); err != nil {
		return
	}
	return
}

func (bb *bubbleServer) GetRange(
	ctx context.Context, req *v1.BubbleGetRangeRequest) (
	resp *v1.BubbleGetRangeResponse, err error) {
	entries, err := bb.get(req).GetRange(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (bb *bubbleServer) GetById(
	ctx context.Context, req *v1.BubbleGetByIdRequest) (
	resp *v1.BubbleGetByIdResponse, err error) {
	entries, err := bb.get(req).GetById(ctx, req.Ids...)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (bb *bubbleServer) RemoveById(
	ctx context.Context, req *v1.BubbleRemoveByIdRequest) (
	resp *v1.BubbleRemoveByIdResponse, err error) {
	if err = bb.get(req).RemoveById(ctx, req.Ids...); err != nil {
		return
	}
	return
}

func (bb *bubbleServer) SetInfo(
	ctx context.Context, req *v1.BubbleSetInfoRequest) (
	resp *v1.BubbleSetInfoResponse, err error) {
	if err = bb.get(req).SetInfo(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (bb *bubbleServer) get(req request) rgo.Bubble {
	return bb.cli.GetBubble(
		fromChartKey(req.GetKey()), fromChartOptions(req.GetOptions())...)
}
