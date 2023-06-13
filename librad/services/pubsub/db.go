package pubsub

import "github.com/ntons/redis"

type cfg struct {
	Redis string `json:"redis"`
}

var (
	cli redis.Client
)
