package registry

import (
	"context"

	"github.com/ntons/libra/librad/registry/db"

	admv1pb "github.com/ntons/libra-go/api/libra/admin/v1"
)

type appAdminServer struct {
	admv1pb.UnimplementedAppAdminServer
}

func newAppServer() *appAdminServer {
	return &appAdminServer{}
}

func (srv *appAdminServer) List(
	context.Context, *admv1pb.AppAdminListRequest) (
	*admv1pb.AppAdminListResponse, error) {
	resp := &admv1pb.AppAdminListResponse{}
	for _, a := range db.ListApps() {
		resp.Apps = append(resp.Apps, &admv1pb.AppData{Id: a.Id})
	}
	return resp, nil
}

func (srv *appAdminServer) Watch(
	req *admv1pb.AppAdminListRequest,
	stream admv1pb.AppAdmin_WatchServer) (err error) {
	watcher := make(chan []*db.App, 10)
	defer db.WatchApps(watcher)()

	{
		// send the first reply
		resp := &admv1pb.AppAdminListResponse{}
		for _, a := range db.ListApps() {
			resp.Apps = append(resp.Apps, &admv1pb.AppData{Id: a.Id})
			stream.Send(resp)
		}
	}

	for {
		// watching app list change
		select {
		case <-stream.Context().Done():
			return
		case as := <-watcher:
			resp := &admv1pb.AppAdminListResponse{}
			for _, a := range as {
				resp.Apps = append(resp.Apps, &admv1pb.AppData{Id: a.Id})
				stream.Send(resp)
			}
		}
	}
}
