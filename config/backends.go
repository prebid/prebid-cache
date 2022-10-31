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
	DefaultTTL      int      `mapstructure:"default_ttl_seconds"`
	Host            string   `mapstructure:"host"`
	Hosts           []string `mapstructure:"hosts"`
	Port            int      `mapstructure:"port"`
	Namespace       string   `mapstructure:"namespace"`
	User            string   `mapstructure:"user"`
	Password        string   `mapstructure:"password"`
	MaxReadRetries  int      `mapstructure:"max_read_retries"`
	MaxWriteRetries int      `mapstructure:"max_write_retries"`
}

func (cfg *Aerospike) validateAndLog() error {
	if len(cfg.Host) < 1 && len(cfg.Hosts) < 1 {
		return fmt.Errorf("Cannot connect to empty Aerospike host(s)")
	}

	if cfg.Port <= 0 {
		return fmt.Errorf("Cannot connect to Aerospike host at port %d", cfg.Port)
	}
	if cfg.DefaultTTL > 0 {
		log.Infof("config.backend.aerospike.default_ttl_seconds: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.DefaultTTL)
	}
	log.Infof("config.backend.aerospike.host: %s", cfg.Host)
	log.Infof("config.backend.aerospike.hosts: %v", cfg.Hosts)
	log.Infof("config.backend.aerospike.port: %d", cfg.Port)
	log.Infof("config.backend.aerospike.namespace: %s", cfg.Namespace)
	log.Infof("config.backend.aerospike.user: %s", cfg.User)

	if cfg.MaxReadRetries < 2 {
		log.Infof("config.backend.aerospike.max_read_retries: %d. Values less than two will default to two", cfg.MaxReadRetries)
		cfg.MaxReadRetries = 2
	}
	if cfg.MaxWriteRetries < 0 {
		log.Infof("config.backend.aerospike.max_write_retries: %d. Value cannot be negative and will default to 0", cfg.MaxWriteRetries)
		cfg.MaxWriteRetries = 0
	} else if cfg.MaxWriteRetries > 0 {
		log.Warnf("config.backend.aerospike.max_write_retries: %d. Database writes that are not idempotent may be performed multiple times when retried", cfg.MaxWriteRetries)
	}

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
	if cfg.DefaultTTL < 0 {
		// Goes back to default if we are provided a negative value
		cfg.DefaultTTL = 2400
	}
	log.Infof("config.backend.cassandra.default_ttl_seconds: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.DefaultTTL)

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
	Host              string   `mapstructure:"host"`
	Port              int      `mapstructure:"port"`
	Password          string   `mapstructure:"password"`
	Db                int      `mapstructure:"db"`
	ExpirationMinutes int      `mapstructure:"expiration"`
	TLS               RedisTLS `mapstructure:"tls"`
}

type RedisTLS struct {
	Enabled            bool `mapstructure:"enabled"`
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

func (cfg *Redis) validateAndLog() error {
	log.Infof("config.backend.redis.host: %s", cfg.Host)
	log.Infof("config.backend.redis.port: %d", cfg.Port)
	log.Infof("config.backend.redis.db: %d", cfg.Db)
	if cfg.ExpirationMinutes > 0 {
		log.Infof("config.backend.redis.expiration: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.ExpirationMinutes)
	}
	log.Infof("config.backend.redis.tls.enabled: %t", cfg.TLS.Enabled)
	log.Infof("config.backend.redis.tls.insecure_skip_verify: %t", cfg.TLS.InsecureSkipVerify)
	return nil
}
