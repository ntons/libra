package database

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jhump/protoreflect/desc"
	"github.com/ntons/distlock"
	"github.com/ntons/remon"
	"github.com/ntons/remon/mailing"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	v1pb "github.com/ntons/libra-go/api/v1"
	"github.com/ntons/libra/librad/internal/comm"
)

func init() {
	comm.RegisterService("database", create)
}

type server struct {
	v1pb.UnimplementedDistlockServer
	v1pb.UnimplementedDatabaseServer
	v1pb.UnimplementedMailboxServer

	remon    *remon.Client
	distlock *distlock.Client
	mailing  *mailing.Client
}

func create(b json.RawMessage) (s comm.Service, err error) {
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	} else if err = cfg.parse(); err != nil {
		return
	}
	srv := &server{}
	// initalize remon
	var ro *redis.Options
	if ro, err = redis.ParseURL(cfg.ReMon.Redis); err != nil {
		return
	}
	r := redis.NewClient(ro)
	m, err := mongo.NewClient(options.Client().ApplyURI(cfg.ReMon.Mongo))
	if err != nil {
		return
	}
	if err = m.Connect(context.Background()); err != nil {
		return
	}
	srv.remon = remon.New(r, m)
	// initalize distlock
	if ro, err = redis.ParseURL(cfg.Distlock.Redis); err != nil {
		return
	}
	srv.distlock = distlock.New(redis.NewClient(ro))

	srv.mailing = mailing.New(srv.remon)

	return srv, nil
}

func (srv *server) Serve() {}
func (srv *server) Stop()  {}

func (srv *server) RegisterGrpc(s *grpc.Server) (err error) {
	v1pb.RegisterDatabaseServer(s, srv)
	v1pb.RegisterDistlockServer(s, srv)
	v1pb.RegisterMailboxServer(s, srv)
	return
}

func (srv *server) RegisterSchema(
	ctx context.Context, req *v1pb.DatabaseRegisterSchemaRequest) (
	res *v1pb.DatabaseRegisterSchemaResponse, err error) {
	fd, err := desc.CreateFileDescriptorFromSet(req.DescriptorSet)
	if err != nil {
		return
	}
	md := fd.FindMessage(req.MessageName)
	if md == nil {
		return nil, fmt.Errorf("cannot find message %s", req.MessageName)
	}

	b, _ := pb.Marshal(req.DescriptorSet)
	h := sha1.New()
	h.Write(b)
	io.WriteString(h, req.MessageName)
	schema := fmt.Sprintf("%s\n", h.Sum(nil))

	res = &v1pb.DatabaseRegisterSchemaResponse{Schema: schema}
	return
}

type keyedRequest interface {
	GetKey() *v1pb.EntryKey
}

func dbKey(req keyedRequest) string {
	return ""
	//k := req.GetKey()
	//return fmt.Sprintf("%s:%s:%s", k.AppId, k.Collection, k.Id)
}
func lockKey(req keyedRequest) string {
	return ""
	//return fmt.Sprintf("lock:{%s}", dbKey(req))
}

func hasValidKey(req keyedRequest) bool {
	return true
	//k := req.GetKey()
	//return k != nil && k.AppId != "" && k.Collection != "" && k.Id != ""
}

////////////////////////////////////////////////////////////////////////////////
// Distlock Service
////////////////////////////////////////////////////////////////////////////////
func (srv *server) Lock(
	ctx context.Context, req *v1pb.DistlockLockRequest) (
	*v1pb.DistlockLockResponse, error) {
	lock, err := srv.distlock.Obtain(ctx, req.Key, cfg.Distlock.ttl)
	if err != nil {
		return nil, distlockerror(err)
	}
	resp := &v1pb.DistlockLockResponse{}
	if resp.Lock, err = anypb.New(lock); err != nil {
		return nil, protoerror(err)
	}
	return resp, nil
}

func (srv *server) Unlock(
	ctx context.Context, req *v1pb.DistlockUnlockRequest) (
	*v1pb.DistlockUnlockResponse, error) {
	if req.Lock == nil {
		return nil, errInvalidArgument
	}
	var lock = &distlock.Lock{}
	if err := req.Lock.UnmarshalTo(lock); err != nil {
		return nil, protoerror(err)
	}
	if err := srv.distlock.Release(ctx, lock); err != nil {
		return nil, distlockerror(err)
	}
	return &v1pb.DistlockUnlockResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Database Service
////////////////////////////////////////////////////////////////////////////////
func (srv *server) Get(
	ctx context.Context, req *v1pb.DatabaseGetRequest) (
	_ *v1pb.DatabaseGetResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	// this operation must be done within 1s because of lock ttl
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// get lock
	resp := &v1pb.DatabaseGetResponse{}
	if req.LockOptions != nil {
		var lock *distlock.Lock
		if lock, err = srv.distlock.Obtain(
			ctx, lockKey(req), cfg.Distlock.ttl); err != nil {
			return nil, distlockerror(err)
		}
		defer func() {
			if err != nil {
				srv.distlock.Release(ctx, lock)
			}
		}()
		if resp.Lock, err = anypb.New(lock); err != nil {
			return nil, protoerror(err)
		}
	}
	// get data
	var s string
	if req.AddIfNotFound != nil {
		var b []byte
		if b, err = pb.Marshal(req.AddIfNotFound); err != nil {
			return nil, protoerror(err)
		}
		s, err = srv.remon.Get(ctx, dbKey(req), remon.AddIfNotFound(comm.B2S(b)))
	} else {
		s, err = srv.remon.Get(ctx, dbKey(req))
	}
	if err != nil {
		return nil, remonerror(err)
	}
	resp.Data = &anypb.Any{}
	if err = pb.Unmarshal(comm.S2B(s), resp.Data); err != nil {
		return nil, protoerror(err)
	}
	return resp, nil
}

func (srv *server) Set(
	ctx context.Context, req *v1pb.DatabaseSetRequest) (
	_ *v1pb.DatabaseSetResponse, err error) {
	// 在处理解锁之前检查请求参数，如果请求参数错误，就很难去猜测
	// 这个请求的真正意图要不要处理锁，所以还是不要动的好
	if !hasValidKey(req) || req.Data == nil {
		return nil, errInvalidArgument
	}
	// this operation must be done within 1s because of lock ttl
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// check lock and unlock options
	if req.Lock != nil {
		lock := &distlock.Lock{}
		if err = req.Lock.UnmarshalTo(lock); err != nil {
			return nil, protoerror(err)
		}
		if err = srv.distlock.Refresh(
			ctx, lock, cfg.Distlock.ttl); err != nil {
			return nil, distlockerror(err)
		}
		if req.UnlockOptions != nil {
			defer func() {
				if err == nil {
					err = srv.distlock.Release(ctx, lock)
				} else if req.UnlockOptions.EvenOnFailure {
					srv.distlock.Release(ctx, lock)
				}
			}()
		}
	}
	b, err := pb.Marshal(req.Data)
	if err != nil {
		return nil, protoerror(err)
	}
	if err = srv.remon.Set(ctx, dbKey(req), comm.B2S(b)); err != nil {
		return nil, remonerror(err)
	}
	return &v1pb.DatabaseSetResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Mailbox Service
////////////////////////////////////////////////////////////////////////////////
func (srv *server) List(
	ctx context.Context, req *v1pb.MailboxListRequest) (
	_ *v1pb.MailboxListResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	list, err := srv.mailing.List(ctx, dbKey(req))
	if err != nil {
		return nil, remonerror(err)
	}
	resp := &v1pb.MailboxListResponse{}
	for _, m := range list {
		resp.Mails = append(resp.Mails, &v1pb.Mail{
			Id: m.Id, Content: m.Content})
	}
	return resp, nil
}
func (srv *server) Push(
	ctx context.Context, req *v1pb.MailboxPushRequest) (
	_ *v1pb.MailboxPushResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	resp := &v1pb.MailboxPushResponse{}
	if resp.MailId, err = srv.mailing.Push(
		ctx, dbKey(req), req.Content,
		mailing.WithCapacity(req.Capacity)); err != nil {
		return nil, remonerror(err)
	}
	return resp, nil
}
func (srv *server) Pull(
	ctx context.Context, req *v1pb.MailboxPullRequest) (
	_ *v1pb.MailboxPullResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	resp := &v1pb.MailboxPullResponse{}
	if resp.PulledIds, err = srv.mailing.Pull(
		ctx, dbKey(req), req.Ids...); err != nil {
		return nil, remonerror(err)
	}
	return resp, nil
}
