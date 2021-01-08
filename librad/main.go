package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/flosch/pongo2"
	"github.com/ghodss/yaml"

	log "github.com/ntons/log-go"
	"go.uber.org/zap"
)

// build time variables
var (
	Version   string
	Built     string
	GitCommit string
	GoVersion string
	OSArch    string
)

func _main() (err error) {
	log.Infow("server is starting",
		"Version", Version,
		"Built", Built,
		"GitCommit", GitCommit,
		"GoVersion", GoVersion,
		"OSArch", OSArch)
	defer log.Info("server has stopped gracefully")

	rand.Seed(time.Now().UnixNano())

	clopts, err := parseCommandLineOptions()
	if err != nil {
		return fmt.Errorf("failed to parse command line options: %w", err)
	}

	cfg, err := loadConfig(clopts.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg.Log != nil {
		var logger *zap.Logger
		if logger, err = cfg.Log.Build(zap.AddCaller()); err != nil {
			return
		}
		log.SetZapLogger(logger)
	}

	if len(clopts.IncludeServices) > 0 {
		services := make(map[string]json.RawMessage)
		for _, name := range clopts.IncludeServices {
			services[name] = cfg.Services[name]
		}
		cfg.Services = services
	}
	if len(clopts.ExcludeServices) > 0 {
		for _, k := range clopts.ExcludeServices {
			delete(cfg.Services, k)
		}
	}

	done := make(chan struct{}, 1)
	defer func() { <-done }()

	go func() {
		defer func() { close(done) }()
		err = serve(cfg)
	}()
	defer shutdown()

	sig := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGPIPE)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	select {
	case <-sig: // terminating by signal
	case <-done: // terminating by server self
	}

	log.Infof("server is stopping")
	return
}
func main() {
	if err := _main(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

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

// command line options override some fileds of config
type CommandLineOptions struct {
	// [-c] configuration file path
	ConfigFilePath string
	// [-i] include service(s) even not in configuration
	IncludeServices Strings
	// [-e] exclude service(s) from configuration
	ExcludeServices Strings
}

func parseCommandLineOptions() (clopts *CommandLineOptions, err error) {
	clopts = &CommandLineOptions{}
	flag.StringVar(&clopts.ConfigFilePath, "c", "", "[C]onfig file path")
	flag.Var(&clopts.IncludeServices, "i", "[I]nclude service")
	flag.Var(&clopts.ExcludeServices, "e", "[E]xclude service")
	flag.Parse()
	return
}

// primary configuration
type Config struct {
	// serving address
	Bind string
	// unix domain socket used to forward gateway request to
	UnixDomainSock string
	// only grpc enabled
	GrpcOnly bool

	// modularized service configuration
	Services map[string]json.RawMessage
	// log configuration
	Log *zap.Config
}

func loadConfig(filePath string) (_ *Config, err error) {
	tpl, err := pongo2.FromFile(filePath)
	if err != nil {
		return
	}
	b, err := tpl.ExecuteBytes(nil)
	if err != nil {
		return
	}
	switch ext := filepath.Ext(filePath); ext {
	case ".json":
	case ".yml", ".yaml":
		if b, err = yaml.YAMLToJSON(b); err != nil {
			return
		}
	default:
		return nil, fmt.Errorf("unknown config file extension: %v", ext)
	}
	cfg := &Config{
		UnixDomainSock: fmt.Sprintf("/tmp/%s.sock", randString(10)),
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	}
	return cfg, nil
}

func randString(n int) string {
	const (
		S = "abcdefghijklmnopqrstuvwxyz"
		N = len(S)
	)
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(S[rand.Intn(N)])
	}
	return sb.String()
}
