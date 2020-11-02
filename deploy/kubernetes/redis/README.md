# redis

Statefulset seems not better than deployment.
Envoy cluster consistent hash by resoved address, configuration 
`common_lb_config.consistent_hashing_lb_config.use_hostname_for_hashing` 
take effect only for StrictDNS clusters with hostnames which resolve to a 
single IP address.

# presetting

* set PILOT_ENABLE_REDIS_FILTER
`kubectl -n istio-system edit deploy/istiod`

# server

* apply server
`kubectl apply -f server.yaml`

# client

* apply client with istio proxy
`istioctl kube-inject -f client.yaml | kubectl apply -f -`

