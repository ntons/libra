package registry

import (
	"context"

	admv1pb "github.com/ntons/libra-go/api/libra/admin/v1"
)

type appServer struct {
	admv1pb.UnimplementedAppServer
}

func newAppServer() *appServer {
	return &appServer{}
}

func (srv *appServer) List(
	context.Context, *admv1pb.AppListRequest) (
	*admv1pb.AppListResponse, error) {
	resp := &admv1pb.AppListResponse{}
	for _, a := range listApps() {
		resp.Apps = append(resp.Apps, &admv1pb.AppData{Id: a.Id})
	}
	return resp, nil
}

func (srv *appServer) Watch(
	req *admv1pb.AppListRequest, stream admv1pb.App_WatchServer) (err error) {
	watcher := make(chan []*xApp, 10)
	defer appWatcher.watch(watcher)()

	{
		// send the first reply
		resp := &admv1pb.AppListResponse{}
		for _, a := range listApps() {
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
			resp := &admv1pb.AppListResponse{}
			for _, a := range as {
				resp.Apps = append(resp.Apps, &admv1pb.AppData{Id: a.Id})
				stream.Send(resp)
			}
		}
	}
}
