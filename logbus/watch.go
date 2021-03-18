package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/log-go"
)

func GetTimeFromLog(b []byte, localTime bool) time.Time {
	v := struct {
		Timestamp time.Time `json:"@timestamp"`
	}{}
	var t time.Time
	if err := json.Unmarshal(b, &v); err != nil {
		t = time.Now()
	} else {
		t = v.Timestamp
	}
	if localTime {
		return t.Local()
	} else {
		return t.UTC()
	}
}

func WatchKey(ctx context.Context, rdb *redis.Client, key string) {
	w := NewWriter(key)
	tm := (*time.Timer)(nil)
	for {
		for {
			v, err := rdb.LRange(ctx, key, 0, cfg.BatchSize).Result()
			if err != nil {
				log.Warnf("failed to query logs: %v", err)
				break
			}
			if len(v) == 0 {
				break // empty
			}
			if err := w.Write(ctx, v...); err != nil {
				log.Warnf("failed to write logs: %v", err)
				break
			}
			if err = rdb.LTrim(ctx, key, int64(len(v)), -1).Err(); err != nil {
				log.Warnf("failed to trim list: %v", err)
				break
			}
			log.Debugf("%d logs have been written into file", len(v))
		}
		if tm == nil {
			tm = time.NewTimer(time.Second)
			defer tm.Stop()
		} else {
			tm.Reset(time.Second)
		}
		select {
		case <-ctx.Done():
			return
		case <-tm.C:
		}
	}
}

func WatchHost(ctx context.Context, rdb *redis.Client) {
	var wg sync.WaitGroup
	defer wg.Wait()

	tk := time.NewTicker(time.Second)
	keys := make(map[string]struct{})
	for {
		if v, err := rdb.Keys(ctx, "*").Result(); err != nil {
			log.Warnf("failed to list keys: %v", err)
		} else {
			for _, key := range v {
				if _, ok := keys[key]; ok {
					continue
				}
				keys[key] = struct{}{}
				wg.Add(1)
				go func(key string) {
					defer wg.Done()
					log.Debugf("start watching key: %s", key)
					WatchKey(ctx, rdb, key)
					log.Debugf("stop watching key: %s", key)
				}(key)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
		}
	}
}
