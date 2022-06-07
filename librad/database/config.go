package database

import (
	"fmt"
	"time"
)

type DatabaseConfig struct {
	Redis       string `json:"redis"`
	Mongo       string `json:"mongo"`
	MaxDataSize int    `json:"maxDataSize"`
}

func (cfg *DatabaseConfig) parse() (err error) {
	if cfg.MaxDataSize <= 0 {
		cfg.MaxDataSize = 256 * 1024
	}
	return
}

type MailBoxConfig struct {
	Redis string `json:"redis"`
	Mongo string `json:"mongo"`
}

type DistlockConfig struct {
	Redis   string `json:"redis"`
	TTLExpr string `json:"ttl"`
	// parsed values
	ttl time.Duration
}

func (cfg *DistlockConfig) parse() (err error) {
	if cfg.ttl, err = time.ParseDuration(cfg.TTLExpr); err != nil {
		return fmt.Errorf("malformed ttl value")
	}
	if cfg.ttl < 2*time.Second {
		return fmt.Errorf("ttl could not be less than 2s")
	}
	return
}

type config struct {
	Database DatabaseConfig `json:"database"`
	MailBox  MailBoxConfig  `json:"mailbox"`
	Distlock DistlockConfig `json:"distlock"`
}

func (cfg *config) parse() (err error) {
	if err = cfg.Database.parse(); err != nil {
		return
	}
	if err = cfg.Distlock.parse(); err != nil {
		return
	}
	return
}

var cfg = &config{}
