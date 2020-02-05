package backends

import (
	"context"
	"crypto/tls"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
	"github.com/prebid/prebid-cache/config"
)

type Redis struct {
	cfg    config.Redis
	client *redis.Client
}

func NewRedisBackend(cfg config.Redis) *Redis {
	constr := cfg.Host + ":" + strconv.Itoa(cfg.Port)

	options := &redis.Options{
		Addr:     constr,
		Password: cfg.Password,
		DB:       cfg.Db,
	}

	if cfg.Tls.Enabled {
		options = &redis.Options{
			Addr:     constr,
			Password: cfg.Password,
			DB:       cfg.Db,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: cfg.Tls.InsecureSkipVerify,
			},
		}
	}

	client := redis.NewClient(options)

	_, err := client.Ping().Result()

	if err != nil {
		log.Fatalf("Error creating Redis backend: %v", err)
	}

	log.Infof("Connected to Redis at %s:%d", cfg.Host, cfg.Port)

	return &Redis{
		cfg:    cfg,
		client: client,
	}
}

func (redis *Redis) Get(ctx context.Context, key string) (string, error) {
	res, err := redis.client.Get(key).Result()

	if err != nil {
		return "", err
	}

	return string(res), nil
}

func (redis *Redis) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = redis.cfg.Expiration * 60
	}
	err := redis.client.Set(key, value, time.Duration(ttlSeconds)*time.Second).Err()

	if err != nil {
		return err
	}

	return nil
}
