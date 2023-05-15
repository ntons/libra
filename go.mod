module github.com/ntons/libra

go 1.18

require (
	github.com/cncf/xds/go v0.0.0-20211011173535-cb28da3451f1
	github.com/envoyproxy/go-control-plane v0.9.10-0.20210907150352-cf90f659a021
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang/protobuf v1.5.2
	github.com/ntons/distlock v0.2.0
	github.com/ntons/grpc-compressor v0.1.1
	github.com/ntons/libra-go v0.0.0-20230515080153-b5455353502f
	github.com/ntons/log-go v0.1.0
	github.com/ntons/redchart v0.1.8
	github.com/ntons/redis v0.1.4
	github.com/ntons/redmq v0.0.0-20220222065331-070944d0f346
	github.com/ntons/remon v0.1.8-0.20230515074222-9887038abcf0
	github.com/ntons/tongo/httputil v0.0.0-20210926235700-c0c0e6e56ff5
	github.com/ntons/tongo/sign v0.0.0-20201009033551-29ad62f045c5
	github.com/onemoreteam/httpframework v0.1.2-0.20220301023911-259e38a8a715
	github.com/pierrec/lz4/v4 v4.1.3
	github.com/sigurn/crc16 v0.0.0-20160107003519-da416fad5162
	github.com/tencentyun/cos-go-sdk-v5 v0.7.41
	github.com/vmihailenco/msgpack/v4 v4.3.12
	go.mongodb.org/mongo-driver v1.5.3
	google.golang.org/genproto v0.0.0-20220118154757-00ab72f36ad5
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)

require (
	github.com/aws/aws-sdk-go v1.34.28 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/flosch/pongo2/v4 v4.0.2 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.9.5 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/onemoreteam/yaml v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sigurn/utils v0.0.0-20190728110027-e1fefb11a144 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/vmihailenco/tagparser v0.1.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

//replace github.com/ntons/libra-go => ../libra-go
//replace github.com/ntons/remon => ../remon
//replace github.com/ntons/distlock => ../distlock
//replace github.com/ntons/ranking => ../ranking
//replace github.com/onemoreteam/httpframework => ../../onemoreteam/httpframework
