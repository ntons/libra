package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
	"google.golang.org/protobuf/proto"

	"github.com/ntons/libra/librad/common/util"
	"github.com/ntons/libra/librad/services/indexing/internal"
)

const (
	substrIndex1FallbackTimeoutDuration = 3 * time.Second
)

func getKey(k *v1pb.EntryKey) string {
	return fmt.Sprintf("%s:%s", k.Kind, k.Id)
}

type substrIndex1 struct {
	// ID
	Id string
	// 读写锁
	mu sync.RWMutex
	// 后缀索引
	tree *internal.SuffixTree
	// Id->Entry索引
	keyIndex map[string]*v1pb.SubstrIndex1_Entry
	// Value->Entry索引
	valIndex map[string]*v1pb.SubstrIndex1_Entry
}

func newSubstrIndex1(id string) *substrIndex1 {
	return &substrIndex1{
		Id: id,
	}
}

func (idx *substrIndex1) tryLoad(ctx context.Context) (err error) {
	if idx.tree != nil {
		return
	}

	var (
		tree     = internal.NewSuffixTree()
		keyIndex = make(map[string]*v1pb.SubstrIndex1_Entry)
		valIndex = make(map[string]*v1pb.SubstrIndex1_Entry)
	)

	d, err := cli.HGetAll(ctx, idx.getRedisKey()).Result()
	if err != nil {
		return newUnavailableError("db error")
	}

	for _, v := range d {
		e := &v1pb.SubstrIndex1_Entry{}
		if err = proto.Unmarshal(util.StringToBytes(v), e); err != nil {
			log.Warnf("failed to unmarshal index entry data")
			return newInternalError("data error")
		}
		k := getKey(e.Key)
		if _, ok := keyIndex[k]; ok {
			log.Warnf("duplicate entry key: %s", k)
			continue
		}
		if _, ok := valIndex[e.Value]; ok {
			log.Warnf("duplicate entry val: %s", e.Value)
			continue
		}
		tree.Add(e.Value)
		keyIndex[k] = e
		valIndex[e.Value] = e
	}

	idx.tree = tree
	idx.keyIndex = keyIndex
	idx.valIndex = valIndex

	return
}

func (idx *substrIndex1) getRedisKey() string {
	return fmt.Sprintf("substrindex1:{%s}", idx.Id)
}

func (idx *substrIndex1) Update(ctx context.Context, e *v1pb.SubstrIndex1_Entry) (err error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if err = idx.tryLoad(ctx); err != nil {
		return
	}

	b, err := proto.Marshal(e)
	if err != nil {
		log.Warnf("failed to marshal entry data: %v", err)
		err = newInvalidArgumentError("data error")
		return
	}

	k := getKey(e.Key)

	// 删除之前的索引
	if _e, ok := idx.keyIndex[k]; ok {
		idx.tree.Del(_e.Value)
		delete(idx.keyIndex, k)
		delete(idx.valIndex, _e.Value)
		defer func() {
			if err == nil {
				return
			}
			idx.tree.Add(_e.Value)
			idx.keyIndex[k] = _e
			idx.valIndex[_e.Value] = _e
		}()
	}

	// 构建索引
	if !idx.tree.Add(e.Value) {
		err = newAlreadyExistsError("duplicate entry key")
		return
	}
	defer func() {
		if err == nil {
			return
		}
		idx.tree.Del(e.Value)
	}()

	// 保存数据
	if err = cli.HSet(ctx, idx.getRedisKey(), k, util.BytesToString(b)).Err(); err != nil {
		log.Warnf("failed to set entry data: %v", err)
		err = newUnavailableError("db error")
		return
	}
	idx.keyIndex[k] = e
	idx.valIndex[e.Value] = e

	return
}

func (idx *substrIndex1) Remove(ctx context.Context, keys []*v1pb.EntryKey) (err error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if err = idx.tryLoad(ctx); err != nil {
		return
	}

	var a []string
	for _, key := range keys {
		a = append(a, getKey(key))
	}

	// 删除数据
	if err = cli.HDel(ctx, idx.getRedisKey(), a...).Err(); err != nil {
		log.Warnf("failed to delete entry data: %v", err)
		err = newUnavailableError("db error")
		// 并不能确定删除成功没有，有可能造成不一致
		return
	}

	// 删除索引
	for _, k := range a {
		if e, ok := idx.keyIndex[k]; ok {
			delete(idx.keyIndex, k)
			delete(idx.valIndex, e.Value)
			idx.tree.Del(e.Value)
		}
	}

	return
}

func (idx *substrIndex1) Search(ctx context.Context, value string) (a []*v1pb.SubstrIndex1_Entry, err error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 加载数据
	if err = idx.tryLoad(ctx); err != nil {
		return
	}

	// 搜索所有匹配值
	for _, v := range idx.tree.Search(value) {
		if e, ok := idx.valIndex[v]; ok {
			a = append(a, e)
		} else {
			log.Warnf("no entry found in value index")
		}
	}

	return
}

type substrIndex1Server struct {
	v1pb.UnimplementedSubstrIndex1ServiceServer
	mu sync.Mutex
	m  map[string]*substrIndex1
}

func newSubstrIndex1Server() *substrIndex1Server {
	return &substrIndex1Server{
		m: make(map[string]*substrIndex1),
	}
}

func (srv *substrIndex1Server) getIndex(id string) *substrIndex1 {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	idx, ok := srv.m[id]
	if !ok {
		idx = newSubstrIndex1(id)
		srv.m[id] = idx
	}
	return idx
}

func (srv *substrIndex1Server) Update(
	ctx context.Context, req *v1pb.SubstrIndex1_UpdateRequest) (
	_ *v1pb.SubstrIndex1_UpdateResponse, err error) {
	if err = srv.getIndex(req.IndexId).Update(ctx, req.Entry); err != nil {
		return
	}
	return &v1pb.SubstrIndex1_UpdateResponse{}, nil
}

func (srv *substrIndex1Server) Remove(
	ctx context.Context, req *v1pb.SubstrIndex1_RemoveRequest) (
	_ *v1pb.SubstrIndex1_RemoveResponse, err error) {
	if err = srv.getIndex(req.IndexId).Remove(ctx, req.Keys); err != nil {
		return
	}
	return &v1pb.SubstrIndex1_RemoveResponse{}, nil
}

func (srv *substrIndex1Server) Search(
	ctx context.Context, req *v1pb.SubstrIndex1_SearchRequest) (
	_ *v1pb.SubstrIndex1_SearchResponse, err error) {
	a, err := srv.getIndex(req.IndexId).Search(ctx, req.Value)
	if err != nil {
		return
	}
	return &v1pb.SubstrIndex1_SearchResponse{Entries: a}, nil
}
