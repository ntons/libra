package xds

import (
	"encoding/json"
	"sort"

	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	log "github.com/ntons/log-go"
	"github.com/ntons/tongo/event"
	etcd "go.etcd.io/etcd/v3/client"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ntons/libra/common"
)

// wrapped ClusterLoadAssignment for indexing endpoints by stream id
type loadAssignment struct {
	*envoyendpoint.ClusterLoadAssignment
	index []string
}

func newLoadAssignment(cluster string) *loadAssignment {
	return &loadAssignment{
		ClusterLoadAssignment: &envoyendpoint.ClusterLoadAssignment{
			ClusterName: cluster,
		},
	}
}

func (x *loadAssignment) update(e *common.Endpoint) {
	i := sort.SearchStrings(x.index, e.Id)
	if i == len(x.index) || x.index[i] != e.Id {
		// update index
		x.index = append(x.index, "")
		copy(x.index[i+1:], x.index[i:])
		x.index[i] = e.Id
		// add endpoints
		if len(x.Endpoints) == 0 {
			x.Endpoints = []*envoyendpoint.LocalityLbEndpoints{{}}
		}
		x.Endpoints[0].LbEndpoints = append(x.Endpoints[0].LbEndpoints, nil)
		copy(x.Endpoints[0].LbEndpoints[i+1:], x.Endpoints[0].LbEndpoints[i:])
	}
	// update endpoints
	x.Endpoints[0].LbEndpoints[i] = &envoyendpoint.LbEndpoint{
		HostIdentifier: &envoyendpoint.LbEndpoint_Endpoint{
			Endpoint: &envoyendpoint.Endpoint{
				Address: &envoycore.Address{
					Address: &envoycore.Address_SocketAddress{
						SocketAddress: &envoycore.SocketAddress{
							Address: e.Address,
							PortSpecifier: &envoycore.SocketAddress_PortValue{
								PortValue: e.Port,
							},
						},
					},
				},
			},
		},
		LoadBalancingWeight: wrapperspb.UInt32(e.Weight),
	}
}

func (x *loadAssignment) remove(e *common.Endpoint) bool {
	i := sort.SearchStrings(x.index, e.Id)
	if i == len(x.index) || x.index[i] != e.Id {
		return false
	}
	// remove from index
	x.index = append(x.index[:i], x.index[i+1:]...)
	// remove from endpoints
	x.Endpoints[0].LbEndpoints = append(
		x.Endpoints[0].LbEndpoints[:i],
		x.Endpoints[0].LbEndpoints[i+1:]...)
	if len(x.Endpoints[0].LbEndpoints) == 0 {
		x.Endpoints = nil
	}
	return true
}

type endpointWatcher struct {
	*watcher
	m  map[string]*loadAssignment // cluster -> loadAssignment
	ev event.Events
}

func dialEndpointWatcher(endpoints []string) (ew *endpointWatcher, err error) {
	w, err := dialWatcher(endpoints)
	if err != nil {
		return
	}
	return &endpointWatcher{
		watcher: w,
		m:       make(map[string]*loadAssignment),
		ev:      make(event.Events),
	}, nil
}

func (ew *endpointWatcher) update(node *etcd.Node) {
	e := &common.Endpoint{}
	if err := json.Unmarshal([]byte(node.Value), e); err != nil {
		log.Warnf("failed to unmarshal node: %s", node.Value)
		return
	}
	x := ew.m[e.Cluster]
	if x == nil {
		x = newLoadAssignment(e.Cluster)
		ew.m[e.Cluster] = x
	}
	x.update(e)
	ew.ev.Emit(e.Cluster, x.ClusterLoadAssignment)
}

func (ew *endpointWatcher) remove(node *etcd.Node) {
	e := &common.Endpoint{}
	if err := json.Unmarshal([]byte(node.Value), e); err != nil {
		log.Warnf("failed to unmarshal node: %s", node.Value)
		return
	}
	if x := ew.m[e.Cluster]; x != nil && x.remove(e) {
		ew.ev.Emit(e.Cluster, x.ClusterLoadAssignment)
	}
}

func (ew *endpointWatcher) watch(
	cluster string, callback event.CallbackFunc) event.CancelFunc {
	if x := ew.m[cluster]; x != nil {
		callback(x.ClusterLoadAssignment)
	} else {
		callback(&envoyendpoint.ClusterLoadAssignment{ClusterName: cluster})
	}
	return ew.ev.Watch(cluster, callback)
}
