package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/distlock"
	"github.com/ntons/log-go"
	"github.com/ntons/remon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"

	v1pb "github.com/ntons/libra-go/api/v1"
	"github.com/ntons/libra/librad/comm"
	"github.com/ntons/libra/librad/comm/util"
)

const (
	xLibraTrustedAppId = "x-libra-trusted-app-id"
)

func init() {
	comm.RegisterService("database", create)
}

type server struct {
	v1pb.UnimplementedDistlockServer
	v1pb.UnimplementedDatabaseServer
	v1pb.UnimplementedMailboxServer

	db remon.Client     // database
	mb remon.MailClient // mailbox
	dl *distlock.Client // distlock
}

func create(b json.RawMessage) (comm.Service, error) {
	dialRedis := func(urls []string) (_ redisCluster, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		opts := make([]*redis.Options, len(urls))
		for i, url := range urls {
			if opts[i], err = redis.ParseURL(url); err != nil {
				return
			}
		}
		cluster := make([]*redis.Client, len(opts))
		for i, opt := range opts {
			rdb := redis.NewClient(opt)
			if err = rdb.Ping(ctx).Err(); err != nil {
				return
			}
			cluster[i] = rdb
		}
		return cluster, nil
	}
	dialMongo := func(url string) (_ *mongo.Client, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		mdb, err := mongo.NewClient(options.Client().ApplyURI(url))
		if err != nil {
			return
		}
		if err = mdb.Connect(ctx); err != nil {
			return
		}
		return mdb, nil
	}

	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, err
	} else if err = cfg.parse(); err != nil {
		return nil, err
	}

	srv := &server{}

	if rdb, err := dialRedis(cfg.Database.Redis); err != nil {
		return nil, err
	} else if mdb, err := dialMongo(cfg.Database.Mongo); err != nil {
		return nil, err
	} else {
		srv.db = remon.New(rdb, mdb)
	}

	if rdb, err := dialRedis(cfg.MailBox.Redis); err != nil {
		return nil, err
	} else if mdb, err := dialMongo(cfg.MailBox.Mongo); err != nil {
		return nil, err
	} else {
		srv.mb = remon.NewMailClient(remon.New(rdb, mdb))
	}

	if rdb, err := dialRedis(cfg.Distlock.Redis); err != nil {
		return nil, err
	} else {
		srv.dl = distlock.New(rdb)
	}

	return srv, nil
}

func (srv *server) Serve() {}
func (srv *server) Stop()  {}

func (srv *server) RegisterGrpc(g *grpc.Server) (err error) {
	v1pb.RegisterDatabaseServer(g, srv)
	v1pb.RegisterDistlockServer(g, srv)
	v1pb.RegisterMailboxServer(g, srv)
	return
}

////////////////////////////////////////////////////////////////////////////////
// Key mapping strategy
////////////////////////////////////////////////////////////////////////////////
type keyedRequest interface {
	GetKey() *v1pb.EntryKey
}

// map entry key to db key
func dbKey(ctx context.Context, req keyedRequest) (_ string, err error) {
	var isValidStr = func(s string, min, max int) bool {
		if len(s) < min || max < len(s) {
			return false
		}
		for _, r := range s {
			if (r < 'a' || 'z' < r) &&
				(r < 'A' || 'Z' < r) &&
				(r < '0' || '9' < r) &&
				(r != '_') && (r != '-') {
				return false
			}
		}
		return true
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errUnauthenticated
	}
	var appId string
	if v := md.Get(xLibraTrustedAppId); len(v) != 1 {
		return "", errUnauthenticated
	} else if appId = v[0]; !isValidStr(appId, 1, 32) {
		return "", errInvalidArgument
	}
	k := req.GetKey()
	if !isValidStr(k.GetKind(), 1, 32) || !isValidStr(k.GetId(), 1, 64) {
		return "", errInvalidArgument
	}
	// default remon key mapping strategy split key by ':'
	// into (database, collection, _id)
	return fmt.Sprintf("%s:%s:%s", appId, k.Kind, k.Id), nil
}

////////////////////////////////////////////////////////////////////////////////
// Distlock Service
////////////////////////////////////////////////////////////////////////////////
func (srv *server) lock(ctx context.Context, key string) (*anypb.Any, error) {
	lock, err := srv.dl.Obtain(ctx, key, cfg.Distlock.ttl)
	if err != nil {
		return nil, distlockerror(err)
	}
	return &anypb.Any{
		TypeUrl: "https://github.com/ntons/distlock",
		Value:   util.StringToBytes(lock.GetToken()),
	}, nil
}

func (srv *server) unlock(
	ctx context.Context, key string, token *anypb.Any) error {
	if token == nil || token.TypeUrl != "https://github.com/ntons/distlock" {
		return errInvalidArgument
	}
	lock := distlock.NewLock(key, util.BytesToString(token.Value))
	if err := srv.dl.Release(ctx, lock); err != nil {
		return distlockerror(err)
	}
	return nil
}

func (srv *server) refresh(
	ctx context.Context, key string, token *anypb.Any) error {
	if token == nil || token.TypeUrl != "https://github.com/ntons/distlock" {
		return errInvalidArgument
	}
	lock := distlock.NewLock(key, util.BytesToString(token.Value))
	if err := srv.dl.Refresh(ctx, lock, cfg.Distlock.ttl); err != nil {
		return distlockerror(err)
	}
	return nil
}

func (srv *server) Lock(
	ctx context.Context, req *v1pb.DistlockLockRequest) (
	_ *v1pb.DistlockLockResponse, err error) {
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	token, err := srv.lock(ctx, key)
	if err != nil {
		return
	}
	return &v1pb.DistlockLockResponse{LockToken: token}, nil
}

func (srv *server) Unlock(
	ctx context.Context, req *v1pb.DistlockUnlockRequest) (
	_ *v1pb.DistlockUnlockResponse, err error) {
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	if err = srv.unlock(ctx, key, req.LockToken); err != nil {
		return
	}
	return &v1pb.DistlockUnlockResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Database Service
////////////////////////////////////////////////////////////////////////////////
//func (srv *server) RegisterSchema(
//	ctx context.Context, req *v1pb.DatabaseRegisterSchemaRequest) (
//	res *v1pb.DatabaseRegisterSchemaResponse, err error) {
//	fd, err := desc.CreateFileDescriptorFromSet(req.DescriptorSet)
//	if err != nil {
//		return
//	}
//	md := fd.FindMessage(req.MessageName)
//	if md == nil {
//		return nil, fmt.Errorf("cannot find message %s", req.MessageName)
//	}
//	b, _ := pb.Marshal(req.DescriptorSet)
//	h := sha1.New()
//	h.Write(b)
//	io.WriteString(h, req.MessageName)
//	schema := fmt.Sprintf("%s\n", h.Sum(nil))
//	res = &v1pb.DatabaseRegisterSchemaResponse{Schema: schema}
//	return
//}

func (srv *server) Get(
	ctx context.Context, req *v1pb.DatabaseGetRequest) (
	_ *v1pb.DatabaseGetResponse, err error) {
	log.Debugw("database.get", "req", req)
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	resp := &v1pb.DatabaseGetResponse{Data: &anypb.Any{}}
	// get lock
	if req.LockOptions != nil {
		// this operation must be done within ttl
		if resp.LockToken, err = srv.lock(ctx, key); err != nil {
			return
		}
		defer func() {
			if err != nil {
				srv.unlock(ctx, key, resp.LockToken)
			}
		}()
	}
	// get data
	var opts []remon.GetOption
	if req.AddIfNotFound != nil {
		if buf, err := encodeMessage(req.AddIfNotFound); err != nil {
			return nil, protoerror(err)
		} else {
			opts = append(opts, remon.AddIfNotFound(buf))
		}
	}
	if rev, buf, err := srv.db.Get(ctx, key, opts...); err != nil {
		return nil, remonerror(err)
	} else if err = decodeMessage(buf, resp.Data); err != nil {
		return nil, protoerror(err)
	} else {
		resp.Revision = rev
	}
	return resp, nil
}

func (srv *server) Set(
	ctx context.Context, req *v1pb.DatabaseSetRequest) (
	_ *v1pb.DatabaseSetResponse, err error) {
	// 在处理解锁之前检查请求参数，如果请求参数错误，就很难去猜测
	// 这个请求的真正意图要不要处理锁，所以还是不要动的好
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	if req.Data == nil {
		return nil, errInvalidArgument
	}
	// check lock and unlock options
	if req.LockToken != nil {
		// this operation must be done within ttl
		if err = srv.refresh(ctx, key, req.LockToken); err != nil {
			return
		}
		if req.UnlockOptions != nil {
			defer func() {
				if err == nil {
					err = srv.unlock(ctx, key, req.LockToken)
				} else if req.UnlockOptions.EvenOnFailure {
					srv.unlock(ctx, key, req.LockToken)
				}
			}()
		}
	}
	resp := &v1pb.DatabaseSetResponse{}
	if buf, err := encodeMessage(req.Data); err != nil {
		return nil, protoerror(err)
	} else if resp.Revision, err = srv.db.Set(ctx, key, buf); err != nil {
		return nil, remonerror(err)
	}
	return resp, nil
}

////////////////////////////////////////////////////////////////////////////////
// Mailbox Service
////////////////////////////////////////////////////////////////////////////////
func (srv *server) List(
	ctx context.Context, req *v1pb.MailboxListRequest) (
	_ *v1pb.MailboxListResponse, err error) {
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	list, err := srv.mb.List(ctx, key)
	if err != nil {
		return nil, remonerror(err)
	}
	resp := &v1pb.MailboxListResponse{}
	for _, e := range list {
		m := &v1pb.Mail{
			Id:   fmt.Sprintf("%x", e.Id),
			Data: &anypb.Any{},
		}
		if err = decodeMessage(e.Val, m.Data); err != nil {
			return nil, protoerror(err)
		}
		resp.Mails = append(resp.Mails, m)
	}
	return resp, nil
}

func (srv *server) Push(
	ctx context.Context, req *v1pb.MailboxPushRequest) (
	_ *v1pb.MailboxPushResponse, err error) {
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	resp := &v1pb.MailboxPushResponse{}
	if buf, err := encodeMessage(req.Data); err != nil {
		return nil, protoerror(err)
	} else if id, err := srv.mb.Push(
		ctx, key, buf, remon.WithCapacity(int(req.Capacity))); err != nil {
		return nil, remonerror(err)
	} else {
		resp.Id = fmt.Sprintf("%x", id)
	}
	return resp, nil
}

func (srv *server) Pull(
	ctx context.Context, req *v1pb.MailboxPullRequest) (
	_ *v1pb.MailboxPullResponse, err error) {
	key, err := dbKey(ctx, req)
	if err != nil {
		return
	}
	ids := make([]int64, 0, len(req.Ids))
	for _, str := range req.Ids {
		id, err := strconv.ParseInt(str, 16, 64)
		if err != nil {
			return nil, errInvalidArgument
		}
		ids = append(ids, id)
	}
	if ids, err = srv.mb.Pull(ctx, key, ids...); err != nil {
		return nil, remonerror(err)
	}
	resp := &v1pb.MailboxPullResponse{
		PulledIds: make([]string, len(ids)),
	}
	for i, id := range ids {
		resp.PulledIds[i] = fmt.Sprintf("%x", id)
	}
	return resp, nil
}
