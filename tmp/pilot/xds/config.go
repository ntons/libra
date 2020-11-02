package xds

type config struct {
	EtcdPrefix    string `json:"etcd_prefix"`
	EtcdEndpoints string `json:"etcd_endpoints"`
}
