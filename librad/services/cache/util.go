package cache

import (
	"time"
)

func getCacheKey(appId string, key string) string {
	return appId + "-" + key
}

func getTimeout(milliseconds int32) time.Duration {
	if milliseconds <= 0 {
		return 24 * time.Hour // by default
	}
	return time.Duration(milliseconds) * time.Millisecond
}
