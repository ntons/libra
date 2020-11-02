package xds

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"sync"

	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoycluster "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	envoydiscovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoyendpoint "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	envoylistener "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	envoyroute "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoyserver "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	log "github.com/ntons/log-go"

	"github.com/ntons/libra/common/server"
)

func init() {
	server.RegisterService("xds", create)
}

type clusterHash struct{}

func (clusterHash) ID(node *envoycore.Node) string {
	if node == nil {
		return ""
	}
	return node.Cluster
}

// This server is driven by callbacks from xds server
// All methods are invoked synchronously, due to the mutex in callbacks
type xdsServer struct {
	server.UnimplementedServer
	envoyserver.Server
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func create(b json.RawMessage) (_ server.Service, err error) {
	// parse config
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	// snapshot cache
	m := log.M{"source": "envoy"}
	cache := envoycache.NewSnapshotCache(false, clusterHash{}, log.With(m))
	// server
	ctx, cancel := context.WithCancel(context.Background())
	xds := &xdsServer{
		Server: envoyserver.NewServer(ctx, cache, Callbacks{}),
		cancel: cancel,
	}
	etcdEndpoints := strings.Split(cfg.EtcdEndpoints, ",")
	// etcd watchers
	pw, err := dialProfileWatcher(etcdEndpoints)
	if err != nil {
		return
	}
	ew, err := dialEndpointWatcher(etcdEndpoints)
	if err != nil {
		return
	}
	xds.wg.Add(1)
	go func() {
		defer xds.wg.Done()
		pw.serve(ctx, path.Join(cfg.EtcdPrefix, "profiles"), etcdEndpoints)
	}()
	xds.wg.Add(1)
	go func() {
		defer xds.wg.Done()
		ew.serve(ctx, path.Join(cfg.EtcdPrefix, "endpoints"), etcdEndpoints)
	}()
	xds.wg.Add(1)
	go func() {
		defer xds.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case node := <-ew.onUpdate:
				log.Debug("endpoint update: ", node)
				ew.update(node)
			case node := <-ew.onRemove:
				log.Debug("endpoint remove: ", node)
				ew.remove(node)
			case node := <-pw.onUpdate:
				log.Debug("profile update: ", node)
				pw.update(node, cache, ew)
			case node := <-pw.onRemove:
				log.Debug("profile delete: ", node)
				pw.remove(node, cache)
			}
		}
	}()
	return xds, nil
}

func (xds *xdsServer) RegisterGrpc(grpcSrv *server.GrpcServer) (err error) {
	envoydiscovery.RegisterAggregatedDiscoveryServiceServer(grpcSrv, xds)
	envoyendpoint.RegisterEndpointDiscoveryServiceServer(grpcSrv, xds)
	envoycluster.RegisterClusterDiscoveryServiceServer(grpcSrv, xds)
	envoyroute.RegisterRouteDiscoveryServiceServer(grpcSrv, xds)
	envoylistener.RegisterListenerDiscoveryServiceServer(grpcSrv, xds)
	return
}

func (xds *xdsServer) Stop() {
	xds.cancel()
	xds.wg.Wait()
}
