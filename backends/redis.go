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

func (db RedisDBClient) Get(key string) (string, error) {
	return db.client.Get(key).Result()
}

func (db RedisDBClient) Put(key string, value string, ttlSeconds int) (bool, error) {
	return db.client.SetNX(key, value, time.Duration(ttlSeconds)*time.Second).Result()
}

//------------------------------------------------------------------------------

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

	redisClient := RedisDBClient{redis.NewClient(options)}

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

func (back *RedisBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := back.client.Get(key)

	if err == redis.Nil {
		err = utils.KeyNotFoundError{}
	}

	return res, err
}

func (back *RedisBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = back.cfg.Expiration * 60
	}

	success, err := back.client.Put(key, value, ttlSeconds)
	if err == nil && !success {
		return utils.RecordExistsError{}
	}
	return err
}
