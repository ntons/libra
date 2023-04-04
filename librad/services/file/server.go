package file

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
	"github.com/tencentyun/cos-go-sdk-v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() { modularity.Register(&fileServer{}) }

type fileServer struct {
	modularity.Skeleton
	v1pb.UnimplementedFileServiceServer

	*cos.Client

	rootPath string
}

func (fileServer) Name() string { return "file" }

func (srv *fileServer) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
	}
	var cfg config
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}

	u, err := url.Parse(cfg.Url)
	if err != nil {
		return
	}

	srv.Client = cos.NewClient(
		&cos.BaseURL{
			BucketURL: u,
		},
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  cfg.SecretId,
				SecretKey: cfg.SecretKey,
			},
		})

	srv.rootPath = u.Path

	server.RegisterService(&v1pb.FileService_ServiceDesc, srv)
	return
}

func (srv *fileServer) Get(
	ctx context.Context, req *v1pb.FileGetRequest) (
	_ *v1pb.FileGetResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	//opt := &cos.ObjectGetOptions{
	//	ResponseContentType: "text/html",
	//	Range:               "bytes=0-3", // 通过 range 下载0~3字节的数据
	//}
	resp, err := srv.Object.Get(
		ctx,
		filepath.Join(srv.rootPath, trusted.AppId, req.Path),
		nil)
	if err != nil {
		log.Warnf("failed to get file from cos: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get file from storage")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("failed to read file from cos: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get file from storage")
	}

	return &v1pb.FileGetResponse{
		File: &v1pb.FileData{
			Path:    req.Path,
			Content: content,
		},
	}, nil
}

func (srv *fileServer) Put(
	ctx context.Context, req *v1pb.FilePutRequest) (
	_ *v1pb.FilePutResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	//opt := &cos.ObjectPutOptions{
	//	ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
	//		ContentType: "text/html",
	//	},
	//	ACLHeaderOptions: &cos.ACLHeaderOptions{
	//		// 如果不是必要操作，建议上传文件时不要给单个文件设置权限，避免达到限制。若不设置默认继承桶的权限。
	//		XCosACL: "private",
	//	},
	//}
	if _, err = srv.Object.Put(
		ctx,
		filepath.Join(srv.rootPath, trusted.AppId, req.File.Path),
		bytes.NewReader(req.File.Content),
		nil,
	); err != nil {
		log.Warnf("failed to put file to cos: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to put file to storage")
	}

	return &v1pb.FilePutResponse{}, nil
}

func (srv *fileServer) Del(
	ctx context.Context, req *v1pb.FileDelRequest) (
	_ *v1pb.FileDelResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	if _, err = srv.Object.Delete(
		ctx,
		filepath.Join(srv.rootPath, trusted.AppId, req.Path),
	); err != nil {
		panic(err)
	}
	return nil, status.Errorf(codes.Unimplemented, "method Del not implemented")
}
