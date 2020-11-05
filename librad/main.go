package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc/metadata"

	"github.com/ntons/libra/librad/comm"
	// register services
	_ "github.com/ntons/libra/librad/database"
	_ "github.com/ntons/libra/librad/gateway"
	_ "github.com/ntons/libra/librad/portal"
	_ "github.com/ntons/libra/librad/ranking"
)

// build time variables
var (
	Version   string
	Built     string
	GitCommit string
	GoVersion string
	OSArch    string
)

func main() {
	fmt.Println("Version:    ", Version)
	fmt.Println("Built:      ", Built)
	fmt.Println("Git Commit: ", GitCommit)
	fmt.Println("Go Version: ", GoVersion)
	fmt.Println("OSArch:     ", OSArch)

	rand.Seed(time.Now().UnixNano())

	const (
		xLibra  = "x-libra-"
		xCookie = "x-cookie-x-libra-"
	)

	if err := comm.Serve(comm.WithGrpcGatewayServeMuxOption(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			return key, strings.HasPrefix(strings.ToLower(key), xLibra)
		}),
		runtime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
			return key, strings.HasPrefix(strings.ToLower(key), xLibra)
		}),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			md := make(metadata.MD)
			for _, cookie := range req.Cookies() {
				if strings.HasPrefix(strings.ToLower(cookie.Name), xLibra) {
					md.Set(cookie.Name, cookie.Value)
				}
			}
			return md
		}),
		runtime.WithForwardResponseOption(func(ctx context.Context, w http.ResponseWriter, _ proto.Message) (_ error) {
			md, ok := runtime.ServerMetadataFromContext(ctx)
			if !ok {
				return
			}
			for key, vals := range md.HeaderMD {
				if strings.HasPrefix(strings.ToLower(key), xCookie) {
					http.SetCookie(w, &http.Cookie{
						Name:  key[len(xCookie)-len(xLibra):],
						Value: vals[0],
					})
				}
			}
			return
		}),
	)); err != nil {
		log.Errorf("failed to serve: %v", err)
		os.Exit(1)
	}
}
