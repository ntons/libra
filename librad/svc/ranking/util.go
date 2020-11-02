package ranking

import (
	"fmt"
	"time"

	"github.com/ntons/libra-go/api/v1"
	rkgo "github.com/ntons/ranking-go"
)

func toChartEntry(in *rkgo.Entry) (out *v1.ChartEntry) {
	return &v1.ChartEntry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}
func fromChartEntry(in *v1.ChartEntry) (out *rkgo.Entry) {
	return &rkgo.Entry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}

func toChartEntries(in []*rkgo.Entry) (out []*v1.ChartEntry) {
	for _, e := range in {
		out = append(out, toChartEntry(e))
	}
	return
}
func fromChartEntries(in []*v1.ChartEntry) (out []*rkgo.Entry) {
	for _, e := range in {
		out = append(out, fromChartEntry(e))
	}
	return
}

func fromChartOptions(in *v1.ChartOptions) (out []rkgo.Option) {
	if in.Capacity > 0 {
		out = append(out, rkgo.WithCapacity(in.Capacity))
	} else {
		out = append(out, rkgo.WithCapacity(1000))
	}
	if in.ConstructFrom != "" {
		out = append(out, rkgo.WithConstructFrom(in.ConstructFrom))
	}
	if in.ExpireAt > 0 {
		out = append(out, rkgo.WithExpireAt(time.Unix(in.ExpireAt, 0)))
	}
	if in.IdleExpire > 0 {
		out = append(out, rkgo.WithIdleExpire(
			time.Duration(in.IdleExpire)*time.Second))
	}
	return nil
}

func fromChartKey(ck *v1.ChartKey) string {
	s := fmt.Sprintf("chart:{%s:%s}", ck.AppId, ck.Name)
	if ck.Suffix != "" {
		s += fmt.Sprintf(":%s", ck.Suffix)
	}
	return s
}
