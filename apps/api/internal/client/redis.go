package client

import (
	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
