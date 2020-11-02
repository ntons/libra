package xds

import (
	"context"
	"time"

	log "github.com/ntons/log-go"
	etcd "go.etcd.io/etcd/v3/client"
)

type watcher struct {
	kapi     etcd.KeysAPI
	onUpdate chan *etcd.Node
	onRemove chan *etcd.Node
}

func dialWatcher(endpoints []string) (w *watcher, err error) {
	cli, err := etcd.New(etcd.Config{
		Endpoints:               endpoints,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = cli.GetVersion(ctx); err != nil {
		return
	}
	return &watcher{
		kapi:     etcd.NewKeysAPI(cli),
		onUpdate: make(chan *etcd.Node),
		onRemove: make(chan *etcd.Node),
	}, nil
}

func (w *watcher) walk(node *etcd.Node) {
	if !node.Dir {
		w.onUpdate <- node
	} else {
		for _, node := range node.Nodes {
			w.walk(node)
		}
	}
}

func (w *watcher) serve(
	ctx context.Context, dirKey string, endpoints []string) (err error) {
	resp, err := func() (*etcd.Response, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return w.kapi.Get(ctx, dirKey, &etcd.GetOptions{Recursive: true})
	}()
	if err != nil {
		return
	}

	w.walk(resp.Node)

	x := w.kapi.Watcher(dirKey, &etcd.WatcherOptions{
		AfterIndex: resp.Index,
		Recursive:  true,
	})
	for {
		if resp, err = x.Next(ctx); err != nil {
			log.Warnf("failed to watch next: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		if resp.Node.Dir {
			continue
		}
		switch resp.Action {
		case "set", "update":
			w.onUpdate <- resp.Node
		case "delete", "expire":
			w.onRemove <- resp.PrevNode
		}
	}
}
