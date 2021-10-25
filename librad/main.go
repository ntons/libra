package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	_ "github.com/ntons/grpc-compressor/lz4" // register lz4 compressor
	log "github.com/ntons/log-go"
	logcfg "github.com/ntons/log-go/config"
	"github.com/onemoreteam/httpframework"
	"github.com/onemoreteam/httpframework/config"
	"github.com/onemoreteam/httpframework/modularity"
	_ "google.golang.org/grpc/encoding/gzip"

	// load modules
	_ "github.com/ntons/libra/librad/auth"
	_ "github.com/ntons/libra/librad/database"
	_ "github.com/ntons/libra/librad/ranking"
	_ "github.com/ntons/libra/librad/registry"
	_ "github.com/ntons/libra/librad/registry/db"
	_ "github.com/ntons/libra/librad/syncer"
	_ "github.com/onemoreteam/httpframework/modularity/log"
	_ "github.com/onemoreteam/httpframework/modularity/server"
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
	rand.Seed(time.Now().UnixNano())

	logcfg.DefaultZapJsonConfig.Use()

	if err := _main(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func _main() (err error) {
	clopts, err := parseCommandLineOptions()
	if err != nil {
		return fmt.Errorf("failed to parse command line options: %w", err)
	}
	log.Infow("server is starting",
		"Version", Version,
		"Built", Built,
		"GitCommit", GitCommit,
		"GoVersion", GoVersion,
		"OSArch", OSArch)
	if clopts.ShowVersionAndExit {
		return
	}

	if len(clopts.IncludeServices) > 0 {
		modularity.DeregisterAllExcept(clopts.IncludeServices...)
	}
	if len(clopts.ExcludeServices) > 0 {
		modularity.Deregister(clopts.ExcludeServices...)
	}

	jb, err := config.BytesFromFile(clopts.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err = modularity.Initialize(jb); err != nil {
		return fmt.Errorf("failed to initalize modules: %w", err)
	}

	defer log.Info("server has stopped gracefully")

	done := make(chan error, 1)
	go func() {
		defer func() { close(done) }()
		modularity.Serve()
	}()

	httpframework.IgnoreSignal(syscall.SIGPIPE)
	select {
	case <-httpframework.WatchSignal(syscall.SIGTERM, syscall.SIGINT).Chan():
		modularity.Shutdown()
		select {
		case <-httpframework.WatchSignal(syscall.SIGTERM, syscall.SIGINT).Chan():
		case <-done:
		}
	case err = <-done:
	}

	modularity.Finalize()

	return
}

////////////////////////////////////////////////////////////////////////////////
type xStrings []string

func (ss *xStrings) String() string {
	sb := strings.Builder{}
	for i, s := range *ss {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(s)
	}
	return sb.String()
}
func (ss *xStrings) Set(v string) (err error) {
	i := sort.SearchStrings(*ss, v)
	if i == len(*ss) || (*ss)[i] != v {
		*ss = append(*ss, "")
		copy((*ss)[i+1:], (*ss)[i:])
		(*ss)[i] = v
	}
	return
}

// command line options override some fileds of config
type xCommandLineOptions struct {
	// [-c] configuration file path
	ConfigFilePath string
	// [-i] include service(s) even not in configuration
	IncludeServices xStrings
	// [-e] exclude service(s) from configuration
	ExcludeServices xStrings
	// [-v] show version and exit
	ShowVersionAndExit bool
}

func parseCommandLineOptions() (clopts *xCommandLineOptions, err error) {
	clopts = &xCommandLineOptions{}
	flag.StringVar(&clopts.ConfigFilePath, "c", "", "[C]onfig file path")
	flag.Var(&clopts.IncludeServices, "i", "[I]nclude service")
	flag.Var(&clopts.ExcludeServices, "e", "[E]xclude service")
	flag.BoolVar(&clopts.ShowVersionAndExit, "v", false, "Show [v]ersion and exit")
	flag.Parse()
	return
}
