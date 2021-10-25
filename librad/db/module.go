package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/onemoreteam/httpframework/modularity"
)

func init() {
	modularity.Register(newModule())
}

type module struct {
	modularity.Skeleton
	ctx    context.Context
	cancel context.CancelFunc
}

func newModule() *module {
	m := &module{}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

func (module) Name() string { return "db" }

func (m *module) Initialize(jb json.RawMessage) (err error) {
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err = dialDatabase(ctx); err != nil {
		return fmt.Errorf("failed to dial database: %v", err)
	}
	return
}

func (m *module) Serve() (err error) {
	dbServe(m.ctx)
	return
}

func (m *module) Shutdown() {
	m.cancel()
}
