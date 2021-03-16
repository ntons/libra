package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type TimedRollingFileConfig struct {
	// output datastore
	Datastore string
	// timedrollingfile options
	MaxSize            int
	MaxBackups         int
	LocalTime          bool
	Compress           bool
	BackupTimeFormat   string
	FilenameTimeFormat string
}

type Config struct {
	Hosts    []string
	Password string
	DB       int
	// how many logs should be got from redis at one query
	BatchSize int64

	TimedRollingFile TimedRollingFileConfig
}

var cfg = &Config{}

func loadConfig(fp string) (err error) {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return
	}
	if err = yaml.Unmarshal(b, cfg); err != nil {
		return
	}
	if cfg.TimedRollingFile.Datastore == "" {
		cfg.TimedRollingFile.Datastore = "./datastore"
	}
	if cfg.TimedRollingFile.MaxSize == 0 {
		cfg.TimedRollingFile.MaxSize = 1024 // 1GB
	}
	if cfg.TimedRollingFile.MaxBackups == 0 {
		cfg.TimedRollingFile.MaxBackups = 10
	}
	if cfg.TimedRollingFile.BackupTimeFormat == "" {
		cfg.TimedRollingFile.BackupTimeFormat = "0405"
	}
	if cfg.TimedRollingFile.FilenameTimeFormat == "" {
		cfg.TimedRollingFile.FilenameTimeFormat = "2006010215.log"
	}
	return
}
