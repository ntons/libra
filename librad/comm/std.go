package comm

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/flosch/pongo2"
	"github.com/ghodss/yaml"
	log "github.com/ntons/log-go"
	"go.uber.org/zap"
)

type Strings []string

func (ss *Strings) String() string {
	sb := strings.Builder{}
	for i, s := range *ss {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(s)
	}
	return sb.String()
}
func (ss *Strings) Set(v string) (err error) {
	i := sort.SearchStrings(*ss, v)
	if i == len(*ss) || (*ss)[i] != v {
		*ss = append(*ss, "")
		copy((*ss)[i+1:], (*ss)[i:])
		(*ss)[i] = v
	}
	return
}

// command line options
type CommandLineOptions struct {
	// [-c] configuration file path
	ConfigFilePath string
	// [-i] include service(s) even not in configuration
	IncludeServices Strings
	// [-e] exclude service(s) from configuration
	ExcludeServices Strings
}

func (clopts *CommandLineOptions) SetFlag() {
	flag.StringVar(&clopts.ConfigFilePath, "c", "", "config file path")
	flag.Var(&clopts.IncludeServices, "i", "include service")
	flag.Var(&clopts.ExcludeServices, "e", "exclude service")
}

// standard configuration
type Config struct {
	// serving address
	Bind string
	// modularized service configuration
	Services map[string]json.RawMessage
	// log configuration
	Log *zap.Config
}

func loadConfig(opts *CommandLineOptions) (_ *Config, err error) {
	tpl, err := pongo2.FromFile(opts.ConfigFilePath)
	if err != nil {
		return
	}
	b, err := tpl.ExecuteBytes(nil)
	if err != nil {
		return
	}
	switch ext := filepath.Ext(opts.ConfigFilePath); ext {
	case ".json":
	case ".yml", ".yaml":
		if b, err = yaml.YAMLToJSON(b); err != nil {
			return
		}
	default:
		return nil, fmt.Errorf("unknown config file extension: %v", ext)
	}
	cfg := &Config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	// include or exclude services
	if len(opts.IncludeServices) > 0 {
		services := make(map[string]json.RawMessage)
		for _, name := range opts.IncludeServices {
			services[name] = cfg.Services[name]
		}
		cfg.Services = services
	}
	if len(opts.ExcludeServices) > 0 {
		for _, k := range opts.ExcludeServices {
			delete(cfg.Services, k)
		}
	}
	// initialize log
	if cfg.Log != nil {
		var logger *zap.Logger
		if logger, err = cfg.Log.Build(zap.AddCaller()); err != nil {
			return
		}
		log.SetZapLogger(logger)
	}
	return cfg, nil
}

// standard modularized service entry
func Serve(opts ...ServeOption) (err error) {
	clopts := &CommandLineOptions{}
	clopts.SetFlag()
	flag.Parse()

	cfg, err := loadConfig(clopts)
	if err != nil {
		return
	}

	lis, err := net.Listen("tcp", cfg.Bind)
	if err != nil {
		return
	}
	defer lis.Close()

	var wg sync.WaitGroup
	defer wg.Wait()

	srv := New()
	defer srv.Shutdown()

	for name, b := range cfg.Services {
		var svc Service
		if svc, err = createService(name, b); err != nil {
			return
		}
		srv.RegisterService(name, svc)
		log.Infof("service %s registered", name)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.Serve(lis, opts...)
	}()

	srv.WaitForTerm()
	log.Infof("terminated")
	return
}
