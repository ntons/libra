module github.com/ntons/libra

go 1.15

require (
	github.com/cncf/xds/go v0.0.0-20210312221358-fbca930ec8ed
	github.com/envoyproxy/go-control-plane v0.9.9-0.20210512163311-63b5d3c536b0
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang/protobuf v1.5.2
	github.com/ntons/distlock v0.2.0
	github.com/ntons/grpc-compressor/lz4 v0.0.0-20210305100006-06d7d07e537e
	github.com/ntons/libra-go v0.0.0-20211209073443-4d374f9b1234
	github.com/ntons/log-go v0.0.0-20210804015646-3ca0ced163e5
	github.com/ntons/ranking v0.1.7-0.20210308073015-fcb506a578cb
	github.com/ntons/remon v0.1.6
	github.com/ntons/tongo/httputil v0.0.0-20210926235700-c0c0e6e56ff5
	github.com/ntons/tongo/redis v0.0.0-20211108034221-312ac468634b
	github.com/ntons/tongo/sign v0.0.0-20201009033551-29ad62f045c5
	github.com/onemoreteam/httpframework v0.0.0-20211112074923-c9a5c5f9c7ce
	github.com/pierrec/lz4/v4 v4.1.3
	github.com/sigurn/crc16 v0.0.0-20160107003519-da416fad5162
	github.com/sigurn/utils v0.0.0-20190728110027-e1fefb11a144 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.12
	go.mongodb.org/mongo-driver v1.5.3
	go.uber.org/multierr v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20210617175327-b9e0b3197ced
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.26.0
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)

//replace github.com/ntons/libra-go => ../libra-go
//replace github.com/ntons/remon => ../remon
//replace github.com/ntons/distlock => ../distlock
//replace github.com/ntons/ranking => ../ranking
//replace github.com/onemoreteam/httpframework => ../../onemoreteam/httpframework
