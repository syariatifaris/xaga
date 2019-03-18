package xaga

import (
	"errors"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	Pool *redis.Pool
	Conn redis.Conn
}

func NewRedis(url string) (*Redis, error) {
	if url == "" {
		return nil, errors.New("empty string url")
	}
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", url)
		},
	}
	if pool == nil {
		return nil, fmt.Errorf("unable to create redis pool for %s", url)
	}
	conn := pool.Get()
	if conn == nil {
		return nil, fmt.Errorf("unable to get redis connection for %s", url)
	}
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return nil, err
	}
	return &Redis{
		Pool: pool,
		Conn: conn,
	}, nil
}
