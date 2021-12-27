package config

import (
	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"fmt"
)

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
	Redis     Redis       `mapstructure:"redis"`
}

func (cfg *Backend) validateAndLog() error {
	logger.Info("config.backend.type: %s", cfg.Type)

	switch cfg.Type {
	case BackendAerospike:
		return cfg.Aerospike.validateAndLog()
	case BackendCassandra:
		return cfg.Cassandra.validateAndLog()
	case BackendMemcache:
		return cfg.Memcache.validateAndLog()
	case BackendRedis:
		return cfg.Redis.validateAndLog()
	case BackendMemory:
		return nil
	default:
		return fmt.Errorf(`invalid config.backend.type: %s. It must be "aerospike", "cassandra", "memcache", "redis", or "memory".`, cfg.Type)
	}
}

type BackendType string

const (
	BackendAerospike BackendType = "aerospike"
	BackendCassandra BackendType = "cassandra"
	BackendMemcache  BackendType = "memcache"
	BackendMemory    BackendType = "memory"
	BackendRedis     BackendType = "redis"
)

type Aerospike struct {
	DefaultTTL int      `mapstructure:"default_ttl_seconds"`
	Host       string   `mapstructure:"host"`
	Hosts      []string `mapstructure:"hosts"`
	Port       int      `mapstructure:"port"`
	Namespace  string   `mapstructure:"namespace"`
	User       string   `mapstructure:"user"`
	Password   string   `mapstructure:"password"`
}

func (cfg *Aerospike) validateAndLog() error {
	if len(cfg.Host) < 1 && len(cfg.Hosts) < 1 {
		return fmt.Errorf("Cannot connect to empty Aerospike host(s)")
	}

	if cfg.Port <= 0 {
		return fmt.Errorf("Cannot connect to Aerospike host at port %d", cfg.Port)
	}
	if cfg.DefaultTTL > 0 {
		logger.Info("config.backend.aerospike.default_ttl_seconds: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.DefaultTTL)
	}
	logger.Info("config.backend.aerospike.host: %s", cfg.Host)
	logger.Info("config.backend.aerospike.hosts: %v", cfg.Hosts)
	logger.Info("config.backend.aerospike.port: %d", cfg.Port)
	logger.Info("config.backend.aerospike.namespace: %s", cfg.Namespace)
	logger.Info("config.backend.aerospike.user: %s", cfg.User)

	return nil
}

type Cassandra struct {
	Hosts      string `mapstructure:"hosts"`
	Keyspace   string `mapstructure:"keyspace"`
	DefaultTTL int    `mapstructure:"default_ttl_seconds"`
}

func (cfg *Cassandra) validateAndLog() error {
	logger.Info("config.backend.cassandra.hosts: %s", cfg.Hosts)
	logger.Info("config.backend.cassandra.keyspace: %s", cfg.Keyspace)
	if cfg.DefaultTTL < 0 {
		// Goes back to default if we are provided a negative value
		cfg.DefaultTTL = 2400
	}
	logger.Info("config.backend.cassandra.default_ttl_seconds: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.DefaultTTL)

	return nil
}

type Memcache struct {
	ConfigHost          string   `mapstructure:"config_host"`
	PollIntervalSeconds int      `mapstructure:"poll_interval_seconds"`
	Hosts               []string `mapstructure:"hosts"`
}

func (cfg *Memcache) validateAndLog() error {
	if cfg.ConfigHost != "" {
		logger.Info("Memcache client will run in auto discovery mode")
		logger.Info("config.backend.memcache.config_host: %s", cfg.ConfigHost)
		logger.Info("config.backend.memcache.poll_interval_seconds: %d", cfg.PollIntervalSeconds)
	} else {
		logger.Info("config.backend.memcache.hosts: %v", cfg.Hosts)
	}
	return nil
}

type Redis struct {
	Host       string   `mapstructure:"host"`
	Port       int      `mapstructure:"port"`
	Password   string   `mapstructure:"password"`
	Db         int      `mapstructure:"db"`
	Expiration int      `mapstructure:"expiration"`
	TLS        RedisTLS `mapstructure:"tls"`
}

type RedisTLS struct {
	Enabled            bool `mapstructure:"enabled"`
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

func (cfg *Redis) validateAndLog() error {
	logger.Info("config.backend.redis.host: %s", cfg.Host)
	logger.Info("config.backend.redis.port: %d", cfg.Port)
	logger.Info("config.backend.redis.db: %d", cfg.Db)
	if cfg.Expiration > 0 {
		logger.Info("config.backend.redis.expiration: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.Expiration)
	}
	logger.Info("config.backend.redis.tls.enabled: %t", cfg.TLS.Enabled)
	logger.Info("config.backend.redis.tls.insecure_skip_verify: %t", cfg.TLS.InsecureSkipVerify)
	return nil
}
