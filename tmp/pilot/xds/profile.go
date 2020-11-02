package xds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"path"
	"time"

	envoycluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoyendpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoyruntime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	envoytypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	// v2 extensions
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/mongo_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/redis_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"

	// v3 extensions
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/mongo_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"

	"github.com/golang/protobuf/jsonpb"
	log "github.com/ntons/log-go"
	"github.com/ntons/tongo/event"
	etcd "go.etcd.io/etcd/v3/client"

	"github.com/ntons/libra/common"
)

// cluster profile
type profile struct {
	// cluster name
	cluster string
	// resources for snapshot
	listeners []envoytypes.Resource
	clusters  []envoytypes.Resource
	routes    []envoytypes.Resource
	endpoints []envoytypes.Resource
	runtimes  []envoytypes.Resource
	// dynamic clusters
	dynamics map[string]int
	// event cancel functions
	cancels []event.CancelFunc
}

// load resource from configuration
func loadProfile(cluster string, x *common.Profile) (_ *profile, err error) {
	p := &profile{
		cluster:  cluster,
		dynamics: make(map[string]int),
	}
	for _, b := range x.Listeners {
		e := &envoylistener.Listener{}
		if err = jsonpb.Unmarshal(bytes.NewReader(b), e); err != nil {
			return
		}
		p.listeners = append(p.listeners, e)
	}
	for _, b := range x.Routes {
		e := &envoyroute.RouteConfiguration{}
		if err = jsonpb.Unmarshal(bytes.NewReader(b), e); err != nil {
			return
		}
		p.routes = append(p.routes, e)
	}
	for _, b := range x.Runtimes {
		e := &envoyruntime.Runtime{}
		if err = jsonpb.Unmarshal(bytes.NewReader(b), e); err != nil {
			return
		}
		p.runtimes = append(p.runtimes, e)
	}
	statics := make(map[string]bool)
	for _, b := range x.Endpoints {
		e := &envoyendpoint.ClusterLoadAssignment{}
		if err = jsonpb.Unmarshal(bytes.NewReader(b), e); err != nil {
			return
		}
		p.endpoints = append(p.endpoints, e)
		statics[e.ClusterName] = true
	}
	for _, b := range x.Clusters {
		e := &envoycluster.Cluster{}
		if err = jsonpb.Unmarshal(bytes.NewReader(b), e); err != nil {
			return
		}
		p.clusters = append(p.clusters, e)
		if e.GetType() == envoycluster.Cluster_EDS && !statics[e.Name] {
			p.dynamics[e.Name] = len(p.endpoints)
			p.endpoints = append(p.endpoints, nil)
		}
	}
	return p, nil
}

func (p *profile) String() string {
	b, _ := json.Marshal(&struct {
		Cluster   string                `json:"cluster"`
		Listeners []envoytypes.Resource `json:"listeners"`
		Clusters  []envoytypes.Resource `json:"clusters"`
		Routes    []envoytypes.Resource `json:"routes"`
		Endpoints []envoytypes.Resource `json:"endpoints"`
		Runtimes  []envoytypes.Resource `json:"runtimes"`
	}{
		Cluster:   p.cluster,
		Listeners: p.listeners,
		Clusters:  p.clusters,
		Routes:    p.routes,
		Endpoints: p.endpoints,
		Runtimes:  p.runtimes,
	})
	return string(b)
}

func (p *profile) start(cache envoycache.SnapshotCache, ew *endpointWatcher) {
	for cluster, i := range p.dynamics {
		i := i
		// callback will be invoked immediately after watching
		cancel := ew.watch(cluster, func(x interface{}) {
			// update endpoints
			p.endpoints[i] = x.(*envoyendpoint.ClusterLoadAssignment)
			log.Debugf("update profile: %s", p)
			// update cache
			snapshot := envoycache.NewSnapshot(
				fmt.Sprintf("%d", time.Now().UnixNano()+rand.Int63n(1<<62)),
				p.endpoints,
				p.clusters,
				p.routes,
				p.listeners,
				p.runtimes,
			)
			if err := cache.SetSnapshot(p.cluster, snapshot); err != nil {
				log.Warnf("failed to set snapshot: %v", err)
			}
		})
		p.cancels = append(p.cancels, cancel)
	}
}

func (p *profile) stop() {
	for _, cancel := range p.cancels {
		cancel()
	}
}

func (p *profile) clear(cache envoycache.SnapshotCache) {
	cache.ClearSnapshot(p.cluster)
}

type profileWatcher struct {
	*watcher
	m map[string]*profile // nodeKey -> profile
}

func dialProfileWatcher(endpoints []string) (pw *profileWatcher, err error) {
	w, err := dialWatcher(endpoints)
	if err != nil {
		return
	}
	return &profileWatcher{
		watcher: w,
		m:       make(map[string]*profile),
	}, nil
}

func (pw *profileWatcher) update(
	node *etcd.Node, cache envoycache.SnapshotCache, ew *endpointWatcher) {
	x := &common.Profile{}
	if err := json.Unmarshal([]byte(node.Value), x); err != nil {
		log.Warnf("failed to unmarshal profile: %s", node.Value)
		return
	}
	p, err := loadProfile(path.Base(node.Key), x)
	if err != nil {
		log.Warnf("failed to unmarshal profile: %v, %s", err, node.Value)
		return
	}
	if oldProf := pw.m[node.Key]; oldProf != nil {
		oldProf.stop()
	}
	p.start(cache, ew)
}

func (pw *profileWatcher) remove(
	node *etcd.Node, cache envoycache.SnapshotCache) {
	x := &common.Profile{}
	if err := json.Unmarshal([]byte(node.Value), x); err != nil {
		log.Warnf("failed to unmarshal profile: %s", node.Value)
		return
	}
	if oldProf := pw.m[node.Key]; oldProf != nil {
		oldProf.stop()
		oldProf.clear(cache)
	}
}
