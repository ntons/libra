package main

import (
	"fmt"
	"sync"
	"time"
)

type Action struct {
}

type Frame struct {
	seq     int32
	actions []*Action
}

type Relay struct {
	d time.Duration // 帧间隔

	mu      sync.Mutex
	actions []*Action
	frames  []*Frame

	stopped bool

	Tick chan *Frame
}

func NewRelay(d time.Duration) *Relay {
	return &Relay{
		d:    d,
		Tick: make(chan *Frame, 10),
	}
}

func (r *Relay) Serve() (err error) {
	r.stopped = false

	seq := int32(0)
	// 立即产生0帧作为开始标志
	f := &Frame{seq: seq}
	r.mu.Lock()
	r.frames = append(r.frames, f)
	r.mu.Unlock()
	if err = r.tick(f); err != nil {
		return
	}

	tk := time.NewTicker(r.d)
	defer tk.Stop()
	for range tk.C {
		if r.stopped {
			return
		}
		seq++
		r.mu.Lock()
		f := &Frame{seq: seq, actions: r.actions}
		r.frames = append(r.frames, f)
		r.actions = nil
		r.mu.Unlock()
		if err = r.tick(f); err != nil {
			return
		}
	}
	return
}

func (r *Relay) Stop() {
	r.stopped = true
}

// retrive frames [i,j]
func (r *Relay) Frames(a ...int) []*Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n := len(a); n == 0 {
		return r.frames
	} else if n == 1 {
		if a[0] < len(r.frames) {
			return r.frames[a[0]:]
		} else {
			return nil
		}
	} else if n == 2 {
		if a[0] < len(r.frames) && a[1] < len(r.frames) {
			return r.frames[a[0] : a[1]+1]
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func (r *Relay) Input(a *Action) {
	if a != nil {
		r.mu.Lock()
		r.actions = append(r.actions, a)
		r.mu.Unlock()
	}
}

func (r *Relay) tick(f *Frame) (err error) {
	select {
	case r.Tick <- f:
		return
	default:
		return fmt.Errorf("channel full")
	}
}
