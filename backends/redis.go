package backends

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
	"github.com/prebid/prebid-cache/config"
	"strconv"
)

type Redis struct {
	cfg    config.Redis
	client *redis.Client
}

func NewRedisBackend(cfg config.Redis) *Redis {
	constr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     constr,
		Password: cfg.Password,
		DB:       cfg.Db,
	})

	_, err := client.Ping().Result()

	if err != nil {
		log.Fatalf("Error creating Redis backend: %v", err)
		panic("Failed Connecting to Redis.")
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

func (redis *Redis) Put(ctx context.Context, key string, value string) error {
	err := redis.client.Set(key, value, 0).Err()

	if err != nil {
		return err
	}

	return nil
}
