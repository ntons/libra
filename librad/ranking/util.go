package ranking

import (
	"fmt"
	"time"

	v1 "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redchart"
)

func toChartEntry(in redchart.Entry) (out *v1.ChartEntry) {
	return &v1.ChartEntry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}
func fromChartEntry(in *v1.ChartEntry) (out redchart.Entry) {
	return redchart.Entry{
		Rank:  in.Rank,
		Id:    in.Id,
		Info:  in.Info,
		Score: in.Score,
	}
}

func toChartEntries(in []redchart.Entry) (out []*v1.ChartEntry) {
	for _, e := range in {
		out = append(out, toChartEntry(e))
	}
	return
}
func fromChartEntries(in []*v1.ChartEntry) (out []redchart.Entry) {
	for _, e := range in {
		out = append(out, fromChartEntry(e))
	}
	return
}

func fromChartOptions(appId string, in *v1.ChartOptions) (out []redchart.Option) {
	if in == nil {
		return
	}
	if in.Capacity > 0 {
		out = append(out, redchart.WithCapacity(in.Capacity))
	} else {
		out = append(out, redchart.WithCapacity(1000))
	}
	if in.ConstructFrom != nil {
		out = append(out, redchart.WithConstructFrom(fromChartKey(appId, in.ConstructFrom)))
	}
	if in.ExpireAt > 0 {
		out = append(out, redchart.WithExpireAt(time.Unix(in.ExpireAt, 0)))
	}
	if in.Expire > 0 {
		out = append(out, redchart.WithExpire(time.Duration(in.Expire)*time.Second))
	}
	if in.NoTrim {
		out = append(out, redchart.WithNoTrim(true))
	}
	if in.NoInfo {
		out = append(out, redchart.WithNoInfo(true))
	}
	return
}

func fromChartKey(appId string, ck *v1.ChartKey) string {
	s := fmt.Sprintf("chart:{%s:%s}", appId, ck.Name)
	if ck.Suffix != "" {
		s += fmt.Sprintf(":%s", ck.Suffix)
	}
	return s
}
