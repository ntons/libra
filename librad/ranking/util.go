package ranking

import (
	"fmt"
	"time"

	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/ranking"
)

func toChartEntry(in *ranking.Entry) (out *v1.ChartEntry) {
	return &v1.ChartEntry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}
func fromChartEntry(in *v1.ChartEntry) (out *ranking.Entry) {
	return &ranking.Entry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}

func toChartEntries(in []*ranking.Entry) (out []*v1.ChartEntry) {
	for _, e := range in {
		out = append(out, toChartEntry(e))
	}
	return
}
func fromChartEntries(in []*v1.ChartEntry) (out []*ranking.Entry) {
	for _, e := range in {
		out = append(out, fromChartEntry(e))
	}
	return
}

func fromChartOptions(appId string, in *v1.ChartOptions) (out []ranking.Option) {
	if in == nil {
		return
	}
	if in.Capacity > 0 {
		out = append(out, ranking.WithCapacity(in.Capacity))
	} else {
		out = append(out, ranking.WithCapacity(1000))
	}
	if in.ConstructFrom != nil {
		out = append(out, ranking.WithConstructFrom(
			fromChartKey(appId, in.ConstructFrom)))
	}
	if in.ExpireAt > 0 {
		out = append(out, ranking.WithExpireAt(time.Unix(in.ExpireAt, 0)))
	}
	if in.IdleExpire > 0 {
		out = append(out, ranking.WithIdleExpire(
			time.Duration(in.IdleExpire)*time.Second))
	}
	if in.NotTrim {
		out = append(out, ranking.WithNotTrim())
	}
	return nil
}

func fromChartKey(appId string, ck *v1.ChartKey) string {
	s := fmt.Sprintf("chart:{%s:%s}", appId, ck.Name)
	if ck.Suffix != "" {
		s += fmt.Sprintf(":%s", ck.Suffix)
	}
	return s
}
