package redis

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

type ScriptClient interface {
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}

type Script struct {
	src, hash string
	// mutex for loading
	mu sync.Mutex
}

func NewScript(src string) *Script {
	h := sha1.New()
	_, _ = io.WriteString(h, src)
	return &Script{src: src, hash: hex.EncodeToString(h.Sum(nil))}
}

func isNoScript(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT ")
}

func (script *Script) Run(ctx context.Context, cli ScriptClient, keys []string, args ...interface{}) (r *redis.Cmd) {
	if r = cli.EvalSha(ctx, script.hash, keys, args...); !isNoScript(r.Err()) {
		return
	}
	script.mu.Lock()
	if r = cli.EvalSha(ctx, script.hash, keys, args...); !isNoScript(r.Err()) {
		script.mu.Unlock()
		return
	}
	if err := cli.ScriptLoad(ctx, script.src).Err(); err != nil {
		script.mu.Unlock()
		r = redis.NewCmd(ctx)
		r.SetErr(err)
		return
	}
	script.mu.Unlock()
	return cli.EvalSha(ctx, script.hash, keys, args...)
}
