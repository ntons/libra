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
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra-go/api/v1"
	"github.com/ntons/libra/librad/comm"
)

func init() {
	comm.RegisterService("database", create)
}

type databaseServer struct {
	comm.UnimplementedServer
	v1.UnimplementedSyncServer
	v1.UnimplementedDatabaseServer
	v1.UnimplementedMailingServer

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
	db := &databaseServer{}
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
	db.remon = remon.New(r, m)
	// initalize distlock
	if ro, err = redis.ParseURL(cfg.Distlock.Redis); err != nil {
		return
	}
	db.distlock = distlock.New(redis.NewClient(ro))

	db.mailing = mailing.New(db.remon)

	return db, nil
}

func (db *databaseServer) RegisterGrpc(s *comm.GrpcServer) (err error) {
	v1.RegisterDatabaseServer(s, db)
	v1.RegisterSyncServer(s, db)
	v1.RegisterMailingServer(s, db)
	return
}

func (db *databaseServer) RegisterSchema(
	ctx context.Context, req *v1.DatabaseRegisterSchemaRequest) (
	res *v1.DatabaseRegisterSchemaResponse, err error) {
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

	res = &v1.DatabaseRegisterSchemaResponse{Schema: schema}
	return
}

type keyedRequest interface {
	GetKey() *v1.EntityKey
}

func dbKey(req keyedRequest) string {
	k := req.GetKey()
	return fmt.Sprintf("%s:%s:%s", k.AppId, k.Collection, k.Id)
}
func lockKey(req keyedRequest) string {
	return fmt.Sprintf("lock:{%s}", dbKey(req))
}

func hasValidKey(req keyedRequest) bool {
	k := req.GetKey()
	return k != nil && k.AppId != "" && k.Collection != "" && k.Id != ""
}

////////////////////////////////////////////////////////////////////////////////
// Sync Service
////////////////////////////////////////////////////////////////////////////////
func (db *databaseServer) Lock(
	ctx context.Context, req *v1.SyncLockRequest) (
	*v1.SyncLockResponse, error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	lock, err := db.distlock.Obtain(ctx, lockKey(req), cfg.Distlock.ttl)
	if err != nil {
		return nil, distlockerror(err)
	}
	resp := &v1.SyncLockResponse{}
	if resp.Lock, err = anypb.New(lock); err != nil {
		return nil, protoerror(err)
	}
	return resp, nil
}

func (db *databaseServer) Unlock(
	ctx context.Context, req *v1.SyncUnlockRequest) (
	*v1.SyncUnlockResponse, error) {
	if req.Lock == nil {
		return nil, errInvalidArgument
	}
	var lock = &distlock.Lock{}
	if err := req.Lock.UnmarshalTo(lock); err != nil {
		return nil, protoerror(err)
	}
	if err := db.distlock.Release(ctx, lock); err != nil {
		return nil, distlockerror(err)
	}
	return &v1.SyncUnlockResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Database Service
////////////////////////////////////////////////////////////////////////////////
func (db *databaseServer) Get(
	ctx context.Context, req *v1.DatabaseGetRequest) (
	_ *v1.DatabaseGetResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	// this operation must be done within 1s because of lock ttl
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// get lock
	resp := &v1.DatabaseGetResponse{}
	if req.LockOptions != nil {
		var lock *distlock.Lock
		if lock, err = db.distlock.Obtain(
			ctx, lockKey(req), cfg.Distlock.ttl); err != nil {
			return nil, distlockerror(err)
		}
		defer func() {
			if err != nil {
				db.distlock.Release(ctx, lock)
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
		s, err = db.remon.Get(ctx, dbKey(req), remon.AddIfNotFound(comm.B2S(b)))
	} else {
		s, err = db.remon.Get(ctx, dbKey(req))
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

func (db *databaseServer) Set(
	ctx context.Context, req *v1.DatabaseSetRequest) (
	_ *v1.DatabaseSetResponse, err error) {
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
		if err = db.distlock.Refresh(
			ctx, lock, cfg.Distlock.ttl); err != nil {
			return nil, distlockerror(err)
		}
		if req.UnlockOptions != nil {
			defer func() {
				if err == nil {
					err = db.distlock.Release(ctx, lock)
				} else if req.UnlockOptions.EvenOnFailure {
					db.distlock.Release(ctx, lock)
				}
			}()
		}
	}
	b, err := pb.Marshal(req.Data)
	if err != nil {
		return nil, protoerror(err)
	}
	if err = db.remon.Set(ctx, dbKey(req), comm.B2S(b)); err != nil {
		return nil, remonerror(err)
	}
	return &v1.DatabaseSetResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Mailing Service
////////////////////////////////////////////////////////////////////////////////
func (db *databaseServer) List(
	ctx context.Context, req *v1.MailingListRequest) (
	_ *v1.MailingListResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	list, err := db.mailing.List(ctx, dbKey(req))
	if err != nil {
		return nil, remonerror(err)
	}
	resp := &v1.MailingListResponse{}
	for _, m := range list {
		resp.Mails = append(resp.Mails, &v1.Mail{
			Id: m.Id, Content: m.Content})
	}
	return resp, nil
}
func (db *databaseServer) Push(
	ctx context.Context, req *v1.MailingPushRequest) (
	_ *v1.MailingPushResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	resp := &v1.MailingPushResponse{}
	if resp.MailId, err = db.mailing.Push(
		ctx, dbKey(req), req.Content,
		mailing.WithCapacity(req.Capacity)); err != nil {
		return nil, remonerror(err)
	}
	return resp, nil
}
func (db *databaseServer) Pull(
	ctx context.Context, req *v1.MailingPullRequest) (
	_ *v1.MailingPullResponse, err error) {
	if !hasValidKey(req) {
		return nil, errInvalidArgument
	}
	resp := &v1.MailingPullResponse{}
	if resp.PulledIds, err = db.mailing.Pull(
		ctx, dbKey(req), req.Ids...); err != nil {
		return nil, remonerror(err)
	}
	return resp, nil
}
