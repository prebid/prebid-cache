package backends

import (
	"context"
	"crypto/tls"
	"reflect"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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
// the RedisDB interface. It can work with both standalone and cluster clients.
type RedisDBClient struct {
	client interface{} // Can be either *redis.Client or *redis.ClusterClient
}

// Get returns the value associated with the provided `key` parameter
func (db RedisDBClient) Get(ctx context.Context, key string) (string, error) {
	switch c := db.client.(type) {
	case *redis.Client:
		return c.Get(ctx, key).Result()
	case *redis.ClusterClient:
		return c.Get(ctx, key).Result()
	default:
		return "", utils.NewPBCError(utils.KEY_NOT_FOUND)
	}
}

// Put will set 'key' to hold string 'value' if 'key' does not exist in the redis storage.
// When key already holds a value, no operation is performed. That's the reason this adapter
// uses the 'github.com/go-redis/redis's library SetNX. SetNX is short for "SET if Not eXists".
func (db RedisDBClient) Put(ctx context.Context, key, value string, ttlSeconds int) (bool, error) {
	switch c := db.client.(type) {
	case *redis.Client:
		return c.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Result()
	case *redis.ClusterClient:
		return c.SetNX(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Result()
	default:
		return false, utils.NewPBCError(utils.KEY_NOT_FOUND)
	}
}

// RedisBackend when initialized will instantiate and configure the Redis client. It implements
// the Backend interface.
type RedisBackend struct {
	cfg    config.Redis
	client RedisDB
}

// NewRedisBackend initializes the redis client and pings to make sure connection was successful
func NewRedisBackend(cfg config.Redis, ctx context.Context) *RedisBackend {
	var redisClient RedisDBClient

	if cfg.Cluster.Enabled {
		// Cluster mode
		clusterOptions := &redis.ClusterOptions{
			Addrs:    cfg.Cluster.Hosts,
			Password: cfg.Password,
			// Note: DB selection is not supported in cluster mode
		}

		// Apply performance configuration for cluster
		applyClusterPerformanceOptions(clusterOptions, cfg)

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
		redisClient = RedisDBClient{client: clusterClient}
	} else {
		// Single-node mode
		constr := cfg.Host + ":" + strconv.Itoa(cfg.Port)

		options := &redis.Options{
			Addr:     constr,
			Password: cfg.Password,
			DB:       cfg.Db,
		}

		// Apply performance configuration for single-node
		applySingleNodePerformanceOptions(options, cfg)

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
		redisClient = RedisDBClient{client: singleClient}
	}

	return &RedisBackend{
		cfg:    cfg,
		client: redisClient,
	}
}

// applyPerformanceOptions applies performance tuning options to both single-node and cluster Redis clients
// This function uses reflection to set common fields on both redis.Options and redis.ClusterOptions
func applyPerformanceOptions(options interface{}, cfg config.Redis) {
	optionsValue := reflect.ValueOf(options).Elem()

	// Apply pool configuration
	if cfg.Pool != nil {
		if cfg.Pool.Size > 0 {
			if field := optionsValue.FieldByName("PoolSize"); field.IsValid() && field.CanSet() {
				field.SetInt(int64(cfg.Pool.Size))
			}
		}
		if cfg.Pool.Timeout > 0 {
			if field := optionsValue.FieldByName("PoolTimeout"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Pool.Timeout))
			}
		}
		if cfg.Pool.MinIdleConns > 0 {
			if field := optionsValue.FieldByName("MinIdleConns"); field.IsValid() && field.CanSet() {
				field.SetInt(int64(cfg.Pool.MinIdleConns))
			}
		}
		if cfg.Pool.MaxIdleConns > 0 {
			if field := optionsValue.FieldByName("MaxIdleConns"); field.IsValid() && field.CanSet() {
				field.SetInt(int64(cfg.Pool.MaxIdleConns))
			}
		}
		if cfg.Pool.ConnMaxIdleTime > 0 {
			if field := optionsValue.FieldByName("ConnMaxIdleTime"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Pool.ConnMaxIdleTime))
			}
		}
		if cfg.Pool.ConnMaxLifetime > 0 {
			if field := optionsValue.FieldByName("ConnMaxLifetime"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Pool.ConnMaxLifetime))
			}
		}
	}

	// Apply timeout configuration
	if cfg.Timeouts != nil {
		if cfg.Timeouts.DialTimeout > 0 {
			if field := optionsValue.FieldByName("DialTimeout"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Timeouts.DialTimeout))
			}
		}
		if cfg.Timeouts.ReadTimeout > 0 {
			if field := optionsValue.FieldByName("ReadTimeout"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Timeouts.ReadTimeout))
			}
		}
		if cfg.Timeouts.WriteTimeout > 0 {
			if field := optionsValue.FieldByName("WriteTimeout"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Timeouts.WriteTimeout))
			}
		}
	}

	// Apply retry configuration
	if cfg.Retry != nil {
		if cfg.Retry.MaxRetries >= 0 {
			if field := optionsValue.FieldByName("MaxRetries"); field.IsValid() && field.CanSet() {
				field.SetInt(int64(cfg.Retry.MaxRetries))
			}
		}
		if cfg.Retry.MinRetryBackoff > 0 {
			if field := optionsValue.FieldByName("MinRetryBackoff"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Retry.MinRetryBackoff))
			}
		}
		if cfg.Retry.MaxRetryBackoff > 0 {
			if field := optionsValue.FieldByName("MaxRetryBackoff"); field.IsValid() && field.CanSet() {
				field.Set(reflect.ValueOf(cfg.Retry.MaxRetryBackoff))
			}
		}
	}
}

// applySingleNodePerformanceOptions applies performance tuning options to single-node Redis client
func applySingleNodePerformanceOptions(options *redis.Options, cfg config.Redis) {
	applyPerformanceOptions(options, cfg)
}

// applyClusterPerformanceOptions applies performance tuning options to cluster Redis client
func applyClusterPerformanceOptions(options *redis.ClusterOptions, cfg config.Redis) {
	applyPerformanceOptions(options, cfg)
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
	if err != nil && err != redis.Nil {
		return err
	}
	if !success {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return nil
}
