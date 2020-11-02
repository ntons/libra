package main

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"
)

// 排除网络条件情况测试帧间隔
func TestRelayPeriod(t *testing.T) {
	var (
		r   = NewRelay(30 * time.Millisecond)
		err error
		wg  sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = r.Serve()
	}()

	var (
		// 标准差
		cnt time.Duration
		sum time.Duration = 0
	)

	stop := time.After(3 * time.Second)
	<-r.C // frame0
	tm := time.Now()
loop:
	for {
		select {
		case <-r.C:
			cnt++
			d := time.Since(tm)
			d -= 30 * time.Millisecond
			sum += d * d
			tm = time.Now()
		case <-stop:
			r.Stop()
			break loop
		}
	}
	wg.Wait()
	if err != nil {
		t.Errorf("Relay错误退出: %v", err)
		return
	}
	sd := time.Duration(math.Sqrt(float64(sum / cnt)))
	if sd > time.Millisecond {
		t.Errorf("帧间隔标准差: %v", sd)
	}
	fmt.Printf("帧间隔标准差: %v\n", sd)
}
