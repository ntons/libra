package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	log "github.com/ntons/log-go"
	"go.uber.org/zap"

	"github.com/ntons/libra/librad/comm"
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
	if err = comm.LoadConfig(clopts.ConfigFilePath); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if comm.Config.Log != nil {
		var logger *zap.Logger
		if logger, err = comm.Config.Log.Build(zap.AddCaller()); err != nil {
			return
		}
		log.SetZapLogger(logger)
	}

	if len(clopts.IncludeServices) > 0 {
		services := make(map[string]json.RawMessage)
		for _, name := range clopts.IncludeServices {
			services[name] = comm.Config.Services[name]
		}
		comm.Config.Services = services
	}
	if len(clopts.ExcludeServices) > 0 {
		for _, name := range clopts.ExcludeServices {
			delete(comm.Config.Services, name)
		}
	}

	done := make(chan struct{}, 1)
	defer func() { <-done }()

	go func() {
		defer func() { close(done) }()
		err = serve()
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
