package backends

import (
	"context"
	"crypto/tls"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

type RedisDB interface {
	Get(key string) (string, error)
	Put(key string, value string, ttlSeconds int) (bool, error)
}

// RedisDBClient is a wrapper for the Redis client that implements
// the RedisDB interface
type RedisDBClient struct {
	client *redis.Client
}

// Get returns the value associated with the provided `key` parameter
func (db RedisDBClient) Get(key string) (string, error) {
	return db.client.Get(key).Result()
}

// Put will set 'key' to hold string 'value' if 'key' does not exist in the redis storage.
// When key already holds a value, no operation is performed. That's the reason this adapter
// uses the 'github.com/go-redis/redis's library SetNX. SetNX is short for "SET if Not eXists".
func (db RedisDBClient) Put(key string, value string, ttlSeconds int) (bool, error) {
	return db.client.SetNX(key, value, time.Duration(ttlSeconds)*time.Second).Result()
}

// Instantiates, and configures the Redis client, it also performs Get
// and Put operations and monitors results. Implements the Backend interface
type RedisBackend struct {
	cfg    config.Redis
	client RedisDB
}

func NewRedisBackend(cfg config.Redis) *RedisBackend {
	constr := cfg.Host + ":" + strconv.Itoa(cfg.Port)

	options := &redis.Options{
		Addr:     constr,
		Password: cfg.Password,
		DB:       cfg.Db,
	}

	if cfg.TLS.Enabled {
		options = &redis.Options{
			Addr:     constr,
			Password: cfg.Password,
			DB:       cfg.Db,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
			},
		}
	}

	redisClient := RedisDBClient{client: redis.NewClient(options)}

	_, err := redisClient.client.Ping().Result()

	if err != nil {
		log.Fatalf("Error creating Redis backend: %v", err)
	}

	log.Infof("Connected to Redis at %s:%d", cfg.Host, cfg.Port)

	return &RedisBackend{
		cfg:    cfg,
		client: redisClient,
	}
}

// Get calls the Redis client to return the value associated with the provided `key`
// parameter and interprets its response. A `Nil` error reply of the Redis client means
// the `key` does not exist.
func (b *RedisBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := b.client.Get(key)

	if err == redis.Nil {
		err = utils.KeyNotFoundError{}
	}

	return res, err
}

// Put writes the `value` under the provided `key` in the Redis storage server. Because the backend
// implementation of Put calls SetNX(item *Item), a `false` return value is interpreted as the data
// not being written because the `key` already holds a value, and a RecordExistsError is returned
func (b *RedisBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = b.cfg.Expiration * 60
	}

	success, err := b.client.Put(key, value, ttlSeconds)
	if !success || err == redis.Nil {
		return utils.RecordExistsError{}
	}
	return err
}
