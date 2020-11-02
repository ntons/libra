package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	etcd "go.etcd.io/etcd/v3/client"

	"github.com/ntons/tongo/template"
)

type profile struct {
	Template string `json:"template"`
	Args     map[string]interface{}
}

type config struct {
	Etcd struct {
		Prefix    string   `json:"prefix"`
		Endpoints []string `json:"endpoints"`
	} `json:"etcd"`

	Profiles map[string]profile `json:"profiles"`
}

type CmdFunc func(args ...string) error

var (
	cfg  = &config{}
	kapi etcd.KeysAPI
	cmds = map[string]CmdFunc{}
)

const usage = `
Usage: %s [options] command

Commands:
  list  profile
  get   profile name
  set   profile [name]
  flush profile [name]

Options:
`

func run() (err error) {
	var (
		// options
		cfgfp         string
		etcdPrefix    string
		etcdEndpoints string
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&cfgfp, "config-file", "cfg.yaml", "config file path")
	flag.StringVar(&etcdPrefix, "etcd-prefix", "/libra.net", "etcd prefix")
	flag.StringVar(&etcdEndpoints, "etcd-endpoints", "", "etcd endpoints")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err = template.UnmarshalFile(cfgfp, nil, &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}
	if etcdPrefix != "" {
		cfg.Etcd.Prefix = etcdPrefix
	}
	if etcdEndpoints != "" {
		cfg.Etcd.Endpoints = strings.Split(etcdEndpoints, ",")
	}
	if len(cfg.Etcd.Endpoints) > 0 {
		var cli etcd.Client
		if cli, err = etcd.New(etcd.Config{
			Endpoints:               cfg.Etcd.Endpoints,
			Transport:               etcd.DefaultTransport,
			HeaderTimeoutPerRequest: time.Second,
		}); err != nil {
			return
		}
		kapi = etcd.NewKeysAPI(cli)
	}
	cmd, ok := cmds[fmt.Sprintf("%s %s", flag.Args()[0], flag.Args()[1])]
	if !ok {
		return fmt.Errorf("unknown command: %v", flag.Args()[0])
	}
	return cmd(flag.Args()[2:]...)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
