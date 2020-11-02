package xds

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/ntons/log-go"
)

type Callbacks struct{}

func (cb Callbacks) OnStreamOpen(
	ctx context.Context, id int64, url string) (err error) {
	log.Debugf("OnStreamOpen: %v, %v", id, url)
	return
}

func (cb Callbacks) OnStreamClosed(id int64) {
	log.Debugf("OnStreamClosed: %v", id)
}

func (cb Callbacks) OnStreamRequest(
	id int64, req *discovery.DiscoveryRequest) (err error) {
	log.Debugf("OnStreamRequest: %v, %#v", id, req)
	return
}

func (cb Callbacks) OnStreamResponse(
	id int64, req *discovery.DiscoveryRequest,
	resp *discovery.DiscoveryResponse) {
	log.Debugf("OnStreamResponse: %v, %#v", id, resp)
}

func (cb Callbacks) OnFetchRequest(
	ctx context.Context, req *discovery.DiscoveryRequest) (err error) {
	log.Debugf("OnFetchRequest: %#v", req)
	return
}

func (cb Callbacks) OnFetchResponse(
	req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	log.Debugf("OnFetchResponse: %#v", resp)
}
