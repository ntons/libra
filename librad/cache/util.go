package cache

import (
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func getCacheKey(appId string, key string) string {
	return appId + "-" + key
}

func getTimeout(o *v1pb.CacheOptions) time.Duration {
	if o == nil || o.TimeoutMilliseconds <= 0 {
		return 24 * time.Hour // by default
	}
	return time.Duration(o.TimeoutMilliseconds) * time.Millisecond
}
