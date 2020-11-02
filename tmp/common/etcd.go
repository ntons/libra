// etcd configuration structure definations
package common

import (
	"encoding/json"
)

// @/${PREFIX}/endpoints/
// Endpoint represent dynamic registered endpoint
type Endpoint struct {
	Id      string `json:"id"`
	Cluster string `json:"cluster"`
	Weight  uint32 `json:"weight"` // load balancing weight
	Address string `json:"address"`
	Port    uint32 `json:"port"`
}

// @/${PREFIX}/profiles/
// Profie represent cluster's envoy configuration
// fileds are detailed by envoy api
type Profile struct {
	Listeners []json.RawMessage `json:"listeners"`
	Clusters  []json.RawMessage `json:"clusters"`
	Routes    []json.RawMessage `json:"routes"`
	Endpoints []json.RawMessage `json:"endpoints"`
	Runtimes  []json.RawMessage `json:"runtimes"`
}

// @/${PREFIX}/xds/endpoint
// Endpoint xds access endpoint
type XDSEndpoint struct {
	ClusterType string `json:"cluster_type"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
}
