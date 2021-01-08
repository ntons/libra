package comm

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	svcFactoryMu sync.Mutex
	svcFactories = make(map[string]ServiceFactory)
)

type ServiceFactory func(cfg json.RawMessage) (Service, error)

func RegisterService(name string, factory ServiceFactory) {
	svcFactoryMu.Lock()
	defer svcFactoryMu.Unlock()
	svcFactories[name] = factory
}

func CreateService(name string, cfg json.RawMessage) (Service, error) {
	svcFactoryMu.Lock()
	defer svcFactoryMu.Unlock()
	if factory, ok := svcFactories[name]; !ok {
		return nil, fmt.Errorf("unregistered service: %s", name)
	} else {
		return factory(cfg)
	}
}
