package main

import (
	"context"
	"fmt"
	"path"
	"time"

	etcd "go.etcd.io/etcd/v3/client"
)

func join(elem ...string) string {
	return path.Join(elem...)
}

func walk(node *etcd.Node, callback func(node *etcd.Node)) {
	if !node.Dir {
		callback(node)
	} else {
		for _, node := range node.Nodes {
			walk(node, callback)
		}
	}
}

func get(path string, args ...string) (err error) {
	var (
		target string
		key    string
		opts   *etcd.GetOptions
	)
	if len(args) > 0 {
		target = args[0]
	}
	if target == "" {
		key = join("/", cfg.Etcd.Prefix, path)
		opts = &etcd.GetOptions{Recursive: true}
	} else {
		key = join("/", cfg.Etcd.Prefix, path, target)
		opts = &etcd.GetOptions{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := kapi.Get(ctx, key, opts)
	if err != nil {
		return fmt.Errorf("failed go get: %v", err)
	}
	walk(resp.Node, func(node *etcd.Node) {
		fmt.Printf("%s: %s\n", node.Key, node.Value)
	})
	return
}

func set(path string, args ...string) (err error) {
	if len(args) < 1 {
		return fmt.Errorf("require set target")
	}
	if len(args) < 2 {
		return fmt.Errorf("require set value")
	}
	key := join("/", cfg.Etcd.Prefix, path, args[0])
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = kapi.Set(ctx, key, args[1], &etcd.SetOptions{}); err != nil {
		return fmt.Errorf("failed to set: %v", err)
	}
	return
}

func list(path string, args ...string) (err error) {
	key := join("/", cfg.Etcd.Prefix, path)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := kapi.Get(ctx, key, &etcd.GetOptions{Recursive: true})
	if err != nil {
		return
	}
	walk(resp.Node, func(node *etcd.Node) {
		fmt.Printf("%s\n", node.Key)
	})
	return
}

func flush(path string, args ...string) (err error) {
	var (
		target string
		key    string
		opts   *etcd.DeleteOptions
	)
	if len(args) > 0 {
		target = args[0]
	}
	if target == "" {
		key = join("/", cfg.Etcd.Prefix, path)
		opts = &etcd.DeleteOptions{Recursive: true}
	} else {
		key = join("/", cfg.Etcd.Prefix, path, target)
		opts = &etcd.DeleteOptions{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = kapi.Delete(ctx, key, opts); err != nil {
		return fmt.Errorf("failed to flush: %v", err)
	}
	return
}
