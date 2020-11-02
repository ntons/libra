// 进程看门狗，附带服务注册功能

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ntons/libra/common"
	etcd "go.etcd.io/etcd/v3/client"
)

const (
	HeartbeatTTL      time.Duration = 30 * time.Second
	HeartbeatInterval time.Duration = 10 * time.Second
)

var (
	// node informations
	NodeId      string
	NodeCluster string
	NodeWeight  uint
	// etcd configurations
	EtcdPrefix    string
	EtcdEndpoints string
	// ingress configrations
	Iface   string
	Address string
	Port    uint64
)

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func getIfaceAddr(name string) (_ string, err error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return
	}
	var ip net.IP // ipv4 is preferred
	for _, a1 := range addrs {
		switch a2 := a1.(type) {
		case *net.IPNet:
			if ip == nil || len(a2.IP) < len(ip) {
				ip = a2.IP
			}
		case *net.IPAddr:
			if ip == nil || len(a2.IP) < len(ip) {
				ip = a2.IP
			}
		}
	}
	if ip == nil {
		return "", fmt.Errorf("no ip on iface %s", Iface)
	}
	return ip.String(), nil
}

func dial() (kapi etcd.KeysAPI, err error) {
	cli, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(EtcdEndpoints, ","),
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
	return etcd.NewKeysAPI(cli), nil
}

func heartbeat(kapi etcd.KeysAPI, key string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// refresh
	if _, err = kapi.Set(ctx, key, "", &etcd.SetOptions{
		TTL:     HeartbeatTTL,
		Refresh: true,
	}); etcd.IsKeyNotFound(err) {
		// register
		value := toJSON(&common.Endpoint{
			Id:      NodeId,
			Cluster: NodeCluster,
			Weight:  uint32(NodeWeight),
			Address: Address,
			Port:    uint32(Port),
		})
		_, err = kapi.Set(ctx, key, value, &etcd.SetOptions{
			TTL: HeartbeatTTL,
		})
	}
	if err != nil {
		fmt.Printf("heartbeat fail: %v\n", err)
	}
	return
}

func serve() (err error) {
	// dail to etcd
	kapi, err := dial()
	if err != nil {
		return
	}
	// start service heartbeat
	key := path.Join(EtcdPrefix, "endpoints", NodeCluster, NodeId)
	defer kapi.Delete(context.Background(), key, &etcd.DeleteOptions{})
	tk := time.NewTicker(HeartbeatInterval)
	defer tk.Stop()
	for range tk.C {
		heartbeat(kapi, key)
	}
	return
}

func main() {
	flag.StringVar(&NodeId, "node-id", "", "node id to register")
	flag.StringVar(&NodeCluster, "node-cluster", "", "node cluster to register")
	flag.UintVar(&NodeWeight, "node-weight", 10, "load balancing weight")
	flag.StringVar(&EtcdPrefix, "etcd-prefix", "", "etcd prefix")
	flag.StringVar(&EtcdEndpoints, "etcd-endpoints", "", "etcd endpoints which seperated by comma")
	flag.StringVar(&Iface, "iface", "eth0", "iface which provide service from")
	flag.Uint64Var(&Port, "port", 10000, "service/ingress port")
	flag.Parse()

	var err error
	if Address, err = getIfaceAddr(Iface); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err = serve(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
