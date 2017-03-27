package broker

import (
	"flag"
	"log"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/heroku/busl/util"
)

var (
	redisURL           = flag.String("redisUrl", os.Getenv("REDIS_URL"), "URL of the redis server")
	redisServer        *url.URL
	redisPool          *pool
	redisKeyExpire     = 60 // redis uses seconds for EXPIRE
	redisChannelExpire = redisKeyExpire * 60
)

type pool struct {
	*redis.Pool
	c int64
}

func (p *pool) Get() Conn {
	n := atomic.AddInt64(&p.c, 1)
	util.SampleWithData("redis.connections", n, "at=acquire")
	return Conn{p.Pool.Get(), p}
}

type Conn struct {
	redis.Conn
	p *pool
}

func (c Conn) Close() error {
	n := atomic.AddInt64(&c.p.c, -1)
	util.SampleWithData("redis.connections", n, "at=release")
	return c.Conn.Close()
}

func init() {
	flag.Parse()
	redisServer, _ = url.Parse(*redisURL)
	redisPool = newPool(redisServer)

	conn := redisPool.Get()
	defer conn.Close()
}

func newPool(server *url.URL) *pool {
	cleanServerURL := *server
	cleanServerURL.User = nil
	log.Printf("connecting to redis: %s", cleanServerURL.String())
	p := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 4 * time.Minute,
		Dial: func() (c redis.Conn, err error) {
			c, err = redis.Dial("tcp", server.Host)
			if err != nil {
				return
			}

			if server.User == nil {
				return
			}

			pw, pwset := server.User.Password()
			if !pwset {
				return
			}

			if _, err = c.Do("AUTH", pw); err != nil {
				c.Close()
				return
			}
			return
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &pool{p, 0}
}

type channel string

func (c channel) id() string {
	return string(c) + ":id"
}

func (c channel) wildcardID() string {
	return string(c) + ":*"
}

func (c channel) doneID() string {
	return string(c) + ":done"
}

func (c channel) killID() string {
	return string(c) + ":kill"
}

// RedisRegistrar is a channel storing data on redis
type RedisRegistrar struct{}

// NewRedisRegistrar creates a new registrar instance
func NewRedisRegistrar() *RedisRegistrar {
	registrar := &RedisRegistrar{}

	return registrar
}

// Register registers the new channel
func (rr *RedisRegistrar) Register(channelName string) (err error) {
	conn := redisPool.Get()
	defer conn.Close()

	channel := channel(channelName)

	_, err = conn.Do("SETEX", channel.id(), redisChannelExpire, make([]byte, 0))
	if err != nil {
		util.CountWithData("RedisRegistrar.Register.error", 1, "error=%s", err)
		return
	}
	return
}

// IsRegistered checks whether a channel name is registered
func (rr *RedisRegistrar) IsRegistered(channelName string) (registered bool, err error) {
	conn := redisPool.Get()
	defer conn.Close()

	channel := channel(channelName)

	exists, err := redis.Bool(conn.Do("EXISTS", channel.id()))
	if err != nil {
		util.CountWithData("RedisRegistrar.IsRegistered.error", 1, "error=%s", err)
		return false, err
	}

	return exists, nil
}

// Get returns a key value
func Get(key string) ([]byte, error) {
	conn := redisPool.Get()
	defer conn.Close()

	channel := channel(key)
	return redis.Bytes(conn.Do("GET", channel.id()))
}
