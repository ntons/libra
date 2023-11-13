package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/common/util"
	"github.com/ntons/log-go"
	"github.com/ntons/redis"
	"github.com/ntons/redlock"
	"github.com/ntons/redmon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/anypb"
)

const distlockTypeUrl = "https://github.com/ntons/distlock"

type server struct {
	v1pb.UnimplementedDistlockServer
	v1pb.UnimplementedDatabaseServer
	v1pb.UnimplementedMailboxServer

	db *redmon.Client  // database
	mb *redmon.Client  // mailbox
	dl *redlock.Client // distlock
}

func createServer(jb json.RawMessage) (*server, error) {
	dialMongo := func(
		ctx context.Context, url string) (_ *mongo.Client, err error) {
		mdb, err := mongo.NewClient(options.Client().ApplyURI(url))
		if err != nil {
			return
		}
		if err = mdb.Connect(ctx); err != nil {
			return
		}
		return mdb, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := json.Unmarshal(jb, cfg); err != nil {
		return nil, err
	} else if err = cfg.parse(); err != nil {
		return nil, err
	}

	log.Debugf("database.cfg: %#v", cfg)

	srv := &server{}

	if rdb, err := redis.Dial(
		ctx, cfg.Database.Redis, redis.WithPingTest()); err != nil {
		return nil, err
	} else if mdb, err := dialMongo(ctx, cfg.Database.Mongo); err != nil {
		return nil, err
	} else {
		srv.db = redmon.NewClient(rdb, mdb)
	}

	if rdb, err := redis.Dial(
		ctx, cfg.MailBox.Redis, redis.WithPingTest()); err != nil {
		return nil, err
	} else if mdb, err := dialMongo(ctx, cfg.MailBox.Mongo); err != nil {
		return nil, err
	} else {
		srv.mb = redmon.NewClient(rdb, mdb)
	}

	if rdb, err := redis.Dial(
		ctx, cfg.Distlock.Redis, redis.WithPingTest()); err != nil {
		return nil, err
	} else {
		srv.dl = redlock.New(rdb)
	}

	return srv, nil
}

// //////////////////////////////////////////////////////////////////////////////
// Key mapping strategy
// //////////////////////////////////////////////////////////////////////////////
type keyedRequest interface {
	GetKey() *v1pb.EntryKey
}

func isValidStr(s string, min, max int) bool {
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

func getAppId(ctx context.Context) (appId string, err error) {
	if trusted := L.RequireAuthBySecret(ctx); trusted == nil {
		return "", errUnauthenticated
	} else if !isValidStr(trusted.AppId, 1, 32) {
		return "", errInvalidArgument
	} else {
		return trusted.AppId, nil
	}
}

func getUniqKey(appId string, key *v1pb.EntryKey) (_ string, err error) {
	if key == nil || !isValidStr(key.Kind, 1, 32) || !isValidStr(key.Id, 1, 64) {
		return "", errInvalidArgument
	}
	// default redmon key mapping strategy split key by ':'
	// into (database, collection, _id)
	return fmt.Sprintf("%s:%s:%s", appId, key.Kind, key.Id), nil
}

func getAppIdAndUniqKey(ctx context.Context, key *v1pb.EntryKey) (_, _ string, err error) {
	appId, err := getAppId(ctx)
	if err != nil {
		return
	}
	uniqKey, err := getUniqKey(appId, key)
	if err != nil {
		return
	}
	return appId, uniqKey, nil
}

// //////////////////////////////////////////////////////////////////////////////
// Distlock Service
// //////////////////////////////////////////////////////////////////////////////
func (srv *server) lock(
	ctx context.Context, key string, opts *v1pb.DistlockLockOptions) (
	*anypb.Any, error) {
	ttl := cfg.Distlock.ttl
	if opts != nil && opts.TimeoutMilliseconds > 0 {
		ttl = time.Duration(opts.TimeoutMilliseconds) * time.Millisecond
	}
	if ttl > 10*time.Minute {
		// 保险起见自动超时不能超过10分钟，以防异常情况下无法解锁
		return nil, errTimeoutTooLong
	}
	lock, err := srv.dl.Obtain(ctx, key, ttl,
		redlock.WithRetryStrategy(redlock.LimitRetry(
			// 32 + 32 + 32 + 32 + 32 + 64 + 128 + 256 + 512 + 512 = 1632
			redlock.ExponentialBackoff(
				32*time.Millisecond,
				512*time.Millisecond), 10)))
	if err != nil {
		return nil, fromRedlockError(err)
	}
	return &anypb.Any{
		TypeUrl: distlockTypeUrl,
		Value:   util.StringToBytes(lock.GetToken()),
	}, nil
}

func (srv *server) unlock(
	ctx context.Context, key string, token *anypb.Any) error {
	if token == nil || token.TypeUrl != distlockTypeUrl {
		return errInvalidArgument
	}
	lock := redlock.NewLock(key, util.BytesToString(token.Value))
	if err := srv.dl.Release(ctx, lock); err != nil {
		return fromRedlockError(err)
	}
	return nil
}

func (srv *server) ensureLock(
	ctx context.Context, key string, token *anypb.Any) error {
	if token == nil || token.TypeUrl != distlockTypeUrl {
		return errInvalidArgument
	}
	lock := redlock.NewLock(key, util.BytesToString(token.Value))
	ttl := cfg.Distlock.ttl
	if t, ok := ctx.Deadline(); ok {
		ttl = t.Sub(time.Now())
	}
	if err := srv.dl.Ensure(ctx, lock, ttl); err != nil {
		return fromRedlockError(err)
	}
	return nil
}

func (srv *server) Lock(
	ctx context.Context, req *v1pb.DistlockLockRequest) (
	_ *v1pb.DistlockLockResponse, err error) {
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
	if err != nil {
		return
	}
	token, err := srv.lock(ctx, key, req.LockOptions)
	if err != nil {
		return
	}
	return &v1pb.DistlockLockResponse{LockToken: token}, nil
}

func (srv *server) Unlock(
	ctx context.Context, req *v1pb.DistlockUnlockRequest) (
	_ *v1pb.DistlockUnlockResponse, err error) {
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
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
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
	if err != nil {
		return
	}
	resp := &v1pb.DatabaseGetResponse{Data: &anypb.Any{}}
	// get lock
	if req.LockOptions != nil {
		// this operation must be done within ttl
		if resp.LockToken, err = srv.lock(
			ctx, key, req.LockOptions); err != nil {
			return
		}
		defer func() {
			if err != nil {
				srv.unlock(ctx, key, resp.LockToken)
			}
		}()
	}
	// get data
	var opts []redmon.GetOption
	if req.AddIfNotFound != nil {
		if buf, err := encodeMessage(req.AddIfNotFound); err != nil {
			return nil, fromProtoError(err)
		} else {
			opts = append(opts, redmon.AddIfNotExists(buf))
		}
	}
	if rev, buf, err := srv.db.Get(ctx, key, opts...); err != nil {
		return nil, fromRedmonError(err)
	} else if err = decodeMessage(buf, resp.Data); err != nil {
		return nil, fromProtoError(err)
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
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
	if err != nil {
		return
	}
	if req.Data == nil {
		return nil, errInvalidArgument
	}
	// check lock and unlock options
	if req.LockToken != nil {
		// this operation must be done within ttl
		if err = srv.ensureLock(ctx, key, req.LockToken); err != nil {
			log.Warnf("set: failed to ensure lock: %s", err)
			return
		}
		if req.UnlockOptions != nil {
			defer func() {
				if err == nil {
					if err = srv.unlock(ctx, key, req.LockToken); err != nil {
						log.Warnf("set: failed to unlock: %s", err)
					}
				} else if req.UnlockOptions.EvenOnFailure {
					if err := srv.unlock(ctx, key, req.LockToken); err != nil {
						log.Warnf("set: failed to unlock: %s", err)
					}
				}
			}()
		}
	}
	buf, err := encodeMessage(req.Data)
	if err != nil {
		log.Warnf("set: failed to encode message: %s", err)
		return nil, fromProtoError(err)
	}
	resp := &v1pb.DatabaseSetResponse{}
	if resp.Revision, err = srv.db.Set(ctx, key, buf); err != nil {
		log.Warnf("set: failed set to db: %s", err)
		return nil, fromRedmonError(err)
	}
	return resp, nil
}

// //////////////////////////////////////////////////////////////////////////////
// Mailbox Service
// //////////////////////////////////////////////////////////////////////////////
func (srv *server) List(
	ctx context.Context, req *v1pb.MailboxListRequest) (
	_ *v1pb.MailboxListResponse, err error) {
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
	if err != nil {
		return
	}
	a, err := srv.mb.List(ctx, key)
	if err != nil {
		if err == redmon.ErrNotExists && req.Options.GetRegardNotFoundAsEmpty() {
			a, err = nil, nil
		} else {
			return nil, fromRedmonError(err)
		}
	}

	// 重要性逆序，时间正序
	sort.Slice(a, func(i, j int) bool {
		vi, vj := a[i].GetImportance(), a[j].GetImportance()
		return vi > vj || vi == vj && a[i].Id < a[j].Id
	})

	resp := &v1pb.MailboxListResponse{}
	for _, e := range a {
		m := &v1pb.Mail{
			Id:   fmt.Sprintf("%x", e.Id),
			Data: &anypb.Any{},
		}
		if err = decodeMessage(e.Val, m.Data); err != nil {
			return nil, fromProtoError(err)
		}
		resp.Mails = append(resp.Mails, m)
		if req.Count > 0 && uint32(len(resp.Mails)) >= req.Count {
			break
		}
	}
	return resp, nil
}

func (srv *server) Pull(
	ctx context.Context, req *v1pb.MailboxPullRequest) (
	_ *v1pb.MailboxPullResponse, err error) {
	_, key, err := getAppIdAndUniqKey(ctx, req.Key)
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
		return nil, fromRedmonError(err)
	}
	resp := &v1pb.MailboxPullResponse{
		PulledIds: make([]string, len(ids)),
	}
	for i, id := range ids {
		resp.PulledIds[i] = fmt.Sprintf("%x", id)
	}
	return resp, nil
}

func (srv *server) Send(
	ctx context.Context, req *v1pb.MailboxSendRequest) (
	_ *v1pb.MailboxSendResponse, err error) {
	appId, err := getAppId(ctx)
	if err != nil {
		return
	}

	for _, e := range req.Envelopes {
		buf, err := encodeMessage(e.Data)
		if err != nil {
			return nil, fromProtoError(err)
		}

		opts := make([]redmon.PushOption, 0, 2)
		if e.Capacity > 0 {
			opts = append(opts, redmon.WithCapacity(uint16(e.Capacity)))
		}
		if e.Overridable {
			opts = append(opts, redmon.WithRing())
		}
		if e.Importance > 255 {
			opts = append(opts, redmon.WithImportance(255))
		} else if e.Importance > 0 {
			opts = append(opts, redmon.WithImportance(uint8(e.Importance)))
		}

		// we send these mails as many as possible, then return the first error
		for _, key := range e.Keys {
			var (
				_key string
				_err error
			)
			if _key, _err = getUniqKey(appId, key); err != nil {
				if err == nil {
					err = _err
				}
				continue
			}
			if _, _err = srv.mb.Push(ctx, _key, buf, opts...); err != nil {
				if _err != redmon.ErrMailBoxFull {
					return nil, fromRedmonError(err)
				}
				if err == nil {
					err = _err
				}
				continue
			}
		}
	}
	return &v1pb.MailboxSendResponse{}, nil
}
