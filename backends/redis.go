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

// RedisBackend when initialized will instantiate and configure the Redis client. It implements
// the Backend interface.
type RedisBackend struct {
	cfg    config.Redis
	client redis.Cmdable // This interface is implemented by both redis.Client and redis.ClusterClient
}

// NewRedisBackend initializes the redis client and pings to make sure connection was successful
func NewRedisBackend(cfg config.Redis, ctx context.Context) *RedisBackend {
	var client redis.Cmdable

	if cfg.Cluster.Enabled {
		// Cluster mode
		clusterOptions := &redis.ClusterOptions{
			Addrs:    cfg.Cluster.Hosts,
			Password: cfg.Password,
			// Note: DB selection is not supported in cluster mode
		}

		if cfg.TLS.Enabled {
			clusterOptions.TLSConfig = &tls.Config{
				InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
			}
		}

		clusterClient := redis.NewClusterClient(clusterOptions)

		// Test cluster connection
		_, err := clusterClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Error creating Redis cluster backend: %v", err)
			panic("RedisBackend failure. This shouldn't happen.")
		}

		log.Infof("Connected to Redis cluster with hosts: %v", cfg.Cluster.Hosts)
		client = clusterClient
	} else {
		// Single-node mode
		constr := cfg.Host + ":" + strconv.Itoa(cfg.Port)

		options := &redis.Options{
			Addr:     constr,
			Password: cfg.Password,
			DB:       cfg.Db,
		}

		if cfg.TLS.Enabled {
			options.TLSConfig = &tls.Config{
				InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
			}
		}

		singleClient := redis.NewClient(options)

		// Test single-node connection
		_, err := singleClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Error creating Redis backend: %v", err)
			panic("RedisBackend failure. This shouldn't happen.")
		}

		log.Infof("Connected to Redis at %s:%d", cfg.Host, cfg.Port)
		client = singleClient
	}

	return &RedisBackend{
		cfg:    cfg,
		client: client,
	}
}

// Get calls the Redis client to return the value associated with the provided `key`
// parameter and interprets its response. A `Nil` error reply of the Redis client means
// the `key` does not exist.
func (b *RedisBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := b.client.Get(ctx, key).Result()

	if err == redis.Nil {
		err = utils.NewPBCError(utils.KEY_NOT_FOUND)
	}

	return res, err
}

// Put writes the `value` under the provided `key` in the Redis storage server. Because the backend
// implementation of Put calls SetNX(item *Item), a `false` return value is interpreted as the data
// not being written because the `key` already holds a value, and a RecordExistsError is returned
func (b *RedisBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	success, err := b.client.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if !success {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return nil
}
