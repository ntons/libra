module github.com/ntons/libra

go 1.15

require (
	github.com/envoyproxy/go-control-plane v0.9.9-0.20201210154907-fd9021fe5dad
	github.com/flosch/pongo2 v0.0.0-20200913210552-0d938eb266f3
	github.com/ghodss/yaml v1.0.0
	github.com/go-redis/redis/v8 v8.3.3
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/ntons/distlock v0.1.1-0.20210304003815-737c837072cf
	github.com/ntons/grpc-compressor/lz4 v0.0.0-20210305100006-06d7d07e537e
	github.com/ntons/libra-go v0.0.0-20210305022616-e09e8fc02e08
	github.com/ntons/log-go v0.0.0-20200924092648-d9caee8882d8
	github.com/ntons/ranking v0.1.7-0.20210308073015-fcb506a578cb
	github.com/ntons/remon v0.1.3-0.20210305043354-079fbe06b9fb
	github.com/ntons/tongo/sign v0.0.0-20201009033551-29ad62f045c5
	github.com/pierrec/lz4/v4 v4.1.3
	github.com/sigurn/crc16 v0.0.0-20160107003519-da416fad5162
	github.com/sigurn/utils v0.0.0-20190728110027-e1fefb11a144 // indirect
	go.mongodb.org/mongo-driver v1.4.3
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102
	google.golang.org/genproto v0.0.0-20201104152603-2e45c02ce95c
	google.golang.org/grpc v1.35.0
	google.golang.org/protobuf v1.25.0
)

//replace github.com/ntons/libra-go => ../libra-go
//replace github.com/ntons/remon => ../remon
//replace github.com/ntons/distlock => ../distlock
//replace github.com/ntons/ranking => ../ranking
