package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
	Redis     Redis       `mapstructure:"redis"`
}

func (cfg *Backend) validateAndLog() error {

	log.Infof("config.backend.type: %s", cfg.Type)
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
	return nil
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
	log.Infof("config.backend.aerospike.default_ttl_seconds: %d", cfg.DefaultTTL)
	log.Infof("config.backend.aerospike.host: %s", cfg.Host)
	log.Infof("config.backend.aerospike.hosts: %v", cfg.Hosts)
	log.Infof("config.backend.aerospike.port: %d", cfg.Port)
	log.Infof("config.backend.aerospike.namespace: %s", cfg.Namespace)
	log.Infof("config.backend.aerospike.user: %s", cfg.User)

	return nil
}

type Cassandra struct {
	Hosts      string `mapstructure:"hosts"`
	Keyspace   string `mapstructure:"keyspace"`
	DefaultTTL int    `mapstructure:"default_ttl_seconds"`
}

func (cfg *Cassandra) validateAndLog() error {
	log.Infof("config.backend.cassandra.hosts: %s", cfg.Hosts)
	log.Infof("config.backend.cassandra.keyspace: %s", cfg.Keyspace)
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 2400
	}
	log.Infof("config.backend.cassandra.default_ttl_seconds: %d", cfg.DefaultTTL)

	return nil
}

type Memcache struct {
	ConfigHost          string   `mapstructure:"config_host"`
	PollIntervalSeconds int      `mapstructure:"poll_interval_seconds"`
	Hosts               []string `mapstructure:"hosts"`
}

func (cfg *Memcache) validateAndLog() error {
	if cfg.ConfigHost != "" {
		log.Infof("Memcache client will run in auto discovery mode")
		log.Infof("config.backend.memcache.config_host: %s", cfg.ConfigHost)
		log.Infof("config.backend.memcache.poll_interval_seconds: %d", cfg.PollIntervalSeconds)
	} else {
		log.Infof("config.backend.memcache.hosts: %v", cfg.Hosts)
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
	log.Infof("config.backend.redis.host: %s", cfg.Host)
	log.Infof("config.backend.redis.port: %d", cfg.Port)
	log.Infof("config.backend.redis.db: %d", cfg.Db)
	log.Infof("config.backend.redis.expiration: %d", cfg.Expiration)
	log.Infof("config.backend.redis.tls.enabled: %t", cfg.TLS.Enabled)
	log.Infof("config.backend.redis.tls.insecure_skip_verify: %t", cfg.TLS.InsecureSkipVerify)
	return nil
}
