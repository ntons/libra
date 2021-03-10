package syncer

import (
	"encoding/json"
	"sync"

	"github.com/ntons/libra/librad/internal/comm"
)

func init() {
	comm.RegisterService("syncer", create)
}

var (
	cfg = &struct {
		Tasks []*struct {
			Name  string
			Mongo string
			Redis []string
		}
	}{}
)

type server struct {
	tasks []*task
	quit  chan struct{}
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}

	tasks := make([]*task, 0, len(cfg.Tasks))
	for _, e := range cfg.Tasks {
		var t *task
		if t, err = dial(e.Name, e.Mongo, e.Redis); err != nil {
			return
		}
		tasks = append(tasks, t)
	}

	return &server{tasks: tasks, quit: make(chan struct{}, 1)}, nil
}

func (srv *server) Serve() {
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, t := range srv.tasks {
		wg.Add(1)
		go func(t *task) {
			defer wg.Done()
			t.Serve()
		}(t)
		defer t.Stop()
	}
	<-srv.quit
	return
}
func (srv *server) Stop() {
	select {
	case srv.quit <- struct{}{}:
	default:
	}
}
