package ranking

import (
	"context"

	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redchart"
	"github.com/ntons/redis"
)

type bubbleChartServer struct {
	v1.UnimplementedBubbleChartServer
	cli redchart.Client
}

func newBubbleChartServer(cli redis.Client) *bubbleChartServer {
	return &bubbleChartServer{cli: redchart.New(cli)}
}

func (bb *bubbleChartServer) Append(
	ctx context.Context, req *v1.BubbleChartAppendRequest) (
	resp *v1.BubbleChartAppendResponse, err error) {
	if err = bb.get(req).Append(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (bb *bubbleChartServer) SwapById(
	ctx context.Context, req *v1.BubbleChartSwapByIdRequest) (
	resp *v1.BubbleChartSwapByIdResponse, err error) {
	if err = bb.get(req).SwapById(ctx, req.Id0, req.Id1); err != nil {
		return
	}
	return
}

func (bb *bubbleChartServer) SwapByRank(
	ctx context.Context, req *v1.BubbleChartSwapByRankRequest) (
	resp *v1.BubbleChartSwapByRankResponse, err error) {
	if err = bb.get(req).SwapByRank(ctx, req.Rank0, req.Rank1); err != nil {
		return
	}
	return
}

func (bb *bubbleChartServer) GetRange(
	ctx context.Context, req *v1.BubbleChartGetRangeRequest) (
	resp *v1.BubbleChartGetRangeResponse, err error) {
	entries, err := bb.get(req).GetByRank(ctx, req.Offset, req.Count)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (bb *bubbleChartServer) GetById(
	ctx context.Context, req *v1.BubbleChartGetByIdRequest) (
	resp *v1.BubbleChartGetByIdResponse, err error) {
	entries, err := bb.get(req).GetById(ctx, req.Ids...)
	if err != nil {
		return
	}
	resp.Entries = toChartEntries(entries)
	return
}

func (bb *bubbleChartServer) RemoveById(
	ctx context.Context, req *v1.BubbleChartRemoveByIdRequest) (
	resp *v1.BubbleChartRemoveByIdResponse, err error) {
	if err = bb.get(req).RemoveById(ctx, req.Ids...); err != nil {
		return
	}
	return
}

func (bb *bubbleChartServer) SetInfo(
	ctx context.Context, req *v1.BubbleChartSetInfoRequest) (
	resp *v1.BubbleChartSetInfoResponse, err error) {
	if err = bb.get(req).SetInfo(
		ctx, fromChartEntries(req.Entries)...); err != nil {
		return
	}
	return
}

func (bb *bubbleChartServer) get(req request) redchart.BubbleChart {
	return bb.cli.GetBubbleChart(
		fromChartKey("", req.GetKey()),
		fromChartOptions("", req.GetOptions())...)
}
