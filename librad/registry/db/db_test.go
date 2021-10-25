package db

import (
	"context"
	"testing"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	logcfg "github.com/ntons/log-go/config"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testApp = &App{
	Id:  "testapp",
	Key: 1024,
}

func testInit() (err error) {
	logcfg.DefaultZapJsonConfig.Use()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const uri = "mongodb://mongo0:27010,mongo1:27011,mongo2:27013/test?replicaSet=rs0"
	cli, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return
	}
	if err = cli.Connect(ctx); err != nil {
		return
	}
	// purge collection
	if err = cli.Database("testapp").Drop(ctx); err != nil {
		return
	}
	mdb = cli
	return
}

func TestBindAcctIdToUser(t *testing.T) {
	if err := testInit(); err != nil {
		t.Fatalf("failed to dial to database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := loginUser(
		ctx,
		testApp,
		"127.0.0.1",
		[]string{"acct1", "acct2"},
		&v1pb.UserLoginOptions{},
	); err != errUserNotFound {
		t.Fatalf("expect user not found, but got: %v", err)
	}

	u1, err := loginUser(
		ctx,
		testApp,
		"127.0.0.1",
		[]string{"acct1", "acct2"},
		&v1pb.UserLoginOptions{
			AutoCreate: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to login user: %v", err)
	}

	u2, err := loginUser(
		ctx,
		testApp,
		"127.0.0.1",
		[]string{"acct2", "acct3"},
		&v1pb.UserLoginOptions{
			AutoCreate: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to login user: %v", err)
	}

	if u1.Id != u2.Id {
		t.Fatalf("user id mismatch: %v, %v", u1.Id, u2.Id)
	}
	if len(u2.AcctIds) != 3 {
		t.Fatalf("user acct ids: %v", u2.AcctIds)
	}

	u3, err := loginUser(
		ctx,
		testApp,
		"127.0.0.1",
		[]string{"acct4", "acct5"},
		&v1pb.UserLoginOptions{
			AutoCreate: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to login user: %v", err)
	}

	if _, err = bindAcctIdToUser(
		ctx,
		testApp.Id,
		u3.Id,
		[]string{"acct3", "acct6"},
		&v1pb.UserBindOptions{},
	); err != errAcctAlreadyExists {
		t.Fatalf("expect acct already exists, but got: %v", err)
	}

	if acctIds, err := bindAcctIdToUser(
		ctx,
		testApp.Id,
		u3.Id,
		[]string{"acct5", "acct7"},
		&v1pb.UserBindOptions{},
	); err != nil {
		t.Fatalf("failed to bind acct to user: %v", err)
	} else if len(acctIds) != 3 {
		t.Fatalf("expected acct ids count 3, but got: %v", len(acctIds))
	}

	if acctIds, err := bindAcctIdToUser(
		ctx,
		testApp.Id,
		u3.Id,
		[]string{"acct3", "acct6"},
		&v1pb.UserBindOptions{
			AutoTransfer: true,
		},
	); err != nil {
		t.Fatalf("failed to bind acct to user: %v", err)
	} else if len(acctIds) != 5 {
		t.Fatalf("expected acct ids count 5, but got: %v", len(acctIds))
	}
}
