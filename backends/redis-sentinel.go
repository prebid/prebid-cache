package backends

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

// RedisSentinelDB is an interface that helps us communicate with an instance of a
// Redis Sentinel database. Its implementation is intended to use the "github.com/redis/go-redis"
// client
type RedisSentinelDB interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error)
}

// RedisSentinelDBClient is a wrapper for the Redis client that implements the RedisSentinelDB interface
type RedisSentinelDBClient struct {
	client *redis.Client
}

// Get returns the value associated with the provided `key` parameter
func (db RedisSentinelDBClient) Get(ctx context.Context, key string) (string, error) {
	return db.client.Get(ctx, key).Result()
}

// Put will set 'key' to hold string 'value' if 'key' does not exist in the redis storage.
// When key already holds a value, no operation is performed. That's the reason this adapter
// uses the 'github.com/go-redis/redis's library SetNX. SetNX is short for "SET if Not eXists".
func (db RedisSentinelDBClient) Put(ctx context.Context, key, value string, ttlSeconds int) (bool, error) {
	return db.client.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Result()
}

// RedisSentinelBackend when initialized will instantiate and configure the Redis client. It implements
// the Backend interface.
type RedisSentinelBackend struct {
	cfg    config.RedisSentinel
	client RedisDB
}

// NewRedisSentinelBackend initializes the Redis Sentinel client and pings to make sure connection was successful
func NewRedisSentinelBackend(cfg config.RedisSentinel, ctx context.Context) *RedisSentinelBackend {
	options := &redis.FailoverOptions{
		MasterName:    cfg.MasterName,
		SentinelAddrs: cfg.SentinelAddrs,
		Password:      cfg.Password,
		DB:            cfg.Db,
	}

	if cfg.TLS.Enabled {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: cfg.TLS.InsecureSkipVerify}
	}

	client := RedisSentinelDBClient{client: redis.NewFailoverClient(options)}

	_, err := client.client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error creating Redis Sentinel backend: %v", err)
	}
	log.Infof("Connected to Redis Sentinels at %v", cfg.SentinelAddrs)

	return &RedisSentinelBackend{
		cfg:    cfg,
		client: client,
	}
}

// Get calls the Redis Sentinel client to return the value associated with the provided `key`
// parameter and interprets its response. A `Nil` error reply of the Redis client means
// the `key` does not exist.
func (b *RedisSentinelBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := b.client.Get(ctx, key)
	if err == redis.Nil {
		err = utils.NewPBCError(utils.KEY_NOT_FOUND)
	}

	return res, err
}

// Put writes the `value` under the provided `key` in the Redis Sentinel storage server. Because the backend
// implementation of Put calls SetNX(item *Item), a `false` return value is interpreted as the data
// not being written because the `key` already holds a value, and a RecordExistsError is returned
func (b *RedisSentinelBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	success, err := b.client.Put(ctx, key, value, ttlSeconds)
	if err != nil && err != redis.Nil {
		return err
	}
	if !success {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}

	return nil
}
