package registry

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
func (aw *xAppWatcher) watch(ch chan<- []*xApp) (cancel func()) {
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
func (aw *xAppWatcher) trigger(as []*xApp) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	for e := aw.ls.Front(); e != nil; e = e.Next() {
		select {
		case e.Value.(chan<- []*xApp) <- as:
		default:
		}
	}
}
