package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type TimedRollingFileConfig struct {
	// output datastore
	Datastore string
	// timedrollingfile options
	MaxSize            string
	maxSize            int // MiB
	MaxBackups         int
	LocalTime          bool
	Compress           bool
	BackupTimeFormat   string
	FilenameTimeFormat string
}

func (cfg *TimedRollingFileConfig) Parse() error {
	if cfg.Datastore == "" {
		cfg.Datastore = "./datastore"
	}
	if cfg.MaxSize == "" {
		cfg.maxSize = 1024
	} else if v, err := ParseSize(cfg.MaxSize); err != nil {
		return err
	} else if cfg.maxSize = v / 1024 / 1024; cfg.maxSize == 0 {
		cfg.maxSize = 1
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 10
	}
	if cfg.BackupTimeFormat == "" {
		cfg.BackupTimeFormat = "0405"
	}
	if cfg.FilenameTimeFormat == "" {
		cfg.FilenameTimeFormat = "2006010215.log"
	}
	return nil
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
	if err = cfg.TimedRollingFile.Parse(); err != nil {
		return
	}
	return
}
