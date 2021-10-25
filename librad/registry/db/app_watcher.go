package db

import (
	"container/list"
	"sync"
)

var (
	appWatcher = &xAppWatcher{}
)

type xAppWatcher struct {
	mu sync.Mutex
	ls list.List
}

// 监视App列表变更
func (aw *xAppWatcher) watch(ch chan<- []*App) (cancel func()) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	var e = aw.ls.PushBack(ch)
	return func() {
		aw.mu.Lock()
		defer aw.mu.Unlock()
		aw.ls.Remove(e)
	}
}

// 变更触发
func (aw *xAppWatcher) trigger(as []*App) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	for e := aw.ls.Front(); e != nil; e = e.Next() {
		select {
		case e.Value.(chan<- []*App) <- as:
		default:
		}
	}
}

func WatchApps(ch chan<- []*App) func() {
	return appWatcher.watch(ch)
}
