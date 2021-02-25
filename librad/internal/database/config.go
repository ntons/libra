package database

import (
	"fmt"
	"time"
)

type ReMonConfig struct {
	Redis []string `json:"redis"`
	Mongo string   `json:"mongo"`
}

func (cfg *ReMonConfig) parse() (err error) {
	return
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
	if cfg.ttl < time.Second {
		return fmt.Errorf("ttl could not be less than 1s")
	}
	return
}

type config struct {
	Database ReMonConfig    `json:"database"`
	MailBox  ReMonConfig    `json:"mailbox"`
	Distlock DistlockConfig `json:"distlock"`
}

func (cfg *config) parse() (err error) {
	if err = cfg.Distlock.parse(); err != nil {
		return
	}
	return
}

var cfg = &config{}
