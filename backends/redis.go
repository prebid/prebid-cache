package backends

import (
	"context"
	"crypto/tls"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

// RedisDB is an interface that helps us communicate with an instance of a
// Redis database. Its implementation is intended to use the "github.com/go-redis/redis"
// client
type RedisDB interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error)
}

// RedisDBClient is a wrapper for the Redis client that implements
// the RedisDB interface
type RedisDBClient struct {
	client *redis.Client
}

// Get returns the value associated with the provided `key` parameter
func (db RedisDBClient) Get(ctx context.Context, key string) (string, error) {
	return db.client.Get(ctx, key).Result()
}

// Put will set 'key' to hold string 'value' if 'key' does not exist in the redis storage.
// When key already holds a value, no operation is performed. That's the reason this adapter
// uses the 'github.com/go-redis/redis's library SetNX. SetNX is short for "SET if Not eXists".
func (db RedisDBClient) Put(ctx context.Context, key, value string, ttlSeconds int) (bool, error) {
	return db.client.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Result()
}

// RedisBackend when initialized will instantiate and configure the Redis client. It implements
// the Backend interface.
type RedisBackend struct {
	cfg    config.Redis
	client RedisDB
}

// NewRedisBackend initializes the redis client and pings to make sure connection was successful
func NewRedisBackend(cfg config.Redis, ctx context.Context) *RedisBackend {
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

	_, err := redisClient.client.Ping(ctx).Result()

	if err != nil {
		log.Fatalf("Error creating Redis backend: %v", err)
		panic("RedisBackend failure. This shouldn't happen.")
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
	res, err := b.client.Get(ctx, key)

	if err == redis.Nil {
		err = utils.NewPBCError(utils.KEY_NOT_FOUND)
	}

	return res, err
}

// Put writes the `value` under the provided `key` in the Redis storage server. Because the backend
// implementation of Put calls SetNX(item *Item), a `false` return value is interpreted as the data
// not being written because the `key` already holds a value, and a RecordExistsError is returned
func (b *RedisBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	success, err := b.client.Put(ctx, key, value, ttlSeconds)
	if err == redis.Nil && !success {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return err
}
