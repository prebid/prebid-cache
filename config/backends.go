package config

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
	Redis     Redis       `mapstructure:"redis"`
	Ignite    Ignite      `mapstructure:"ignite"`
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
	case BackendIgnite:
		return cfg.Ignite.validateAndLog()
	case BackendMemory:
		return nil
	default:
		return fmt.Errorf(`invalid config.backend.type: %s. It must be "aerospike", "cassandra", "memcache", "redis",  "ignite", or "memory".`, cfg.Type)
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
	BackendIgnite    BackendType = "ignite"
)

type Aerospike struct {
	DefaultTTLSecs  int      `mapstructure:"default_ttl_seconds"`
	Host            string   `mapstructure:"host"`
	Hosts           []string `mapstructure:"hosts"`
	Port            int      `mapstructure:"port"`
	Namespace       string   `mapstructure:"namespace"`
	User            string   `mapstructure:"user"`
	Password        string   `mapstructure:"password"`
	MaxReadRetries  int      `mapstructure:"max_read_retries"`
	MaxWriteRetries int      `mapstructure:"max_write_retries"`
	// Please set this to a value lower than the `proto-fd-idle-ms` (converted
	// to seconds) value set in your Aerospike Server. This is to avoid having
	// race conditions where the server closes the connection but the client still
	// tries to use it. If set to a value less than or equal to 0, Aerospike
	// Client's default value will be used which is 55 seconds.
	ConnIdleTimeoutSecs int `mapstructure:"connection_idle_timeout_seconds"`
	// Specifies the size of the connection queue per node.
	ConnQueueSize int `mapstructure:"connection_queue_size"`
}

func (cfg *Aerospike) validateAndLog() error {
	if len(cfg.Host) < 1 && len(cfg.Hosts) < 1 {
		return fmt.Errorf("Cannot connect to empty Aerospike host(s)")
	}

	if cfg.Port <= 0 {
		return fmt.Errorf("Cannot connect to Aerospike host at port %d", cfg.Port)
	}

	log.Infof("config.backend.aerospike.host: %s", cfg.Host)
	log.Infof("config.backend.aerospike.hosts: %v", cfg.Hosts)
	log.Infof("config.backend.aerospike.port: %d", cfg.Port)
	log.Infof("config.backend.aerospike.namespace: %s", cfg.Namespace)
	log.Infof("config.backend.aerospike.user: %s", cfg.User)

	if cfg.DefaultTTLSecs > 0 {
		log.Infof("config.backend.aerospike.default_ttl_seconds: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.DefaultTTLSecs)
	}

	if cfg.ConnIdleTimeoutSecs > 0 {
		log.Infof("config.backend.aerospike.connection_idle_timeout_seconds: %d.", cfg.ConnIdleTimeoutSecs)
	}

	if cfg.MaxReadRetries < 2 {
		log.Infof("config.backend.aerospike.max_read_retries value will default to 2")
		cfg.MaxReadRetries = 2
	} else if cfg.MaxReadRetries > 2 {
		log.Infof("config.backend.aerospike.max_read_retries: %d.", cfg.MaxReadRetries)
	}

	if cfg.MaxWriteRetries < 0 {
		log.Infof("config.backend.aerospike.max_write_retries value cannot be negative and will default to 0")
		cfg.MaxWriteRetries = 0
	} else if cfg.MaxWriteRetries > 0 {
		log.Infof("config.backend.aerospike.max_write_retries: %d.", cfg.MaxWriteRetries)
	}

	if cfg.ConnQueueSize > 0 {
		log.Infof("config.backend.aerospike.connection_queue_size: %d", cfg.ConnQueueSize)
	} else {
		log.Infof("config.backend.aerospike.connection_queue_size value will default to 256")
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
	Host              string         `mapstructure:"host"`
	Port              int            `mapstructure:"port"`
	Password          string         `mapstructure:"password"`
	Db                int            `mapstructure:"db"`
	ExpirationMinutes int            `mapstructure:"expiration"`
	TLS               RedisTLS       `mapstructure:"tls"`
	Cluster           RedisCluster   `mapstructure:"cluster"`
	Pool              *RedisPool     `mapstructure:"pool"`
	Timeouts          *RedisTimeouts `mapstructure:"timeouts"`
	Retry             *RedisRetry    `mapstructure:"retry"`
}

type RedisPool struct {
	Size            int           `mapstructure:"size"`
	Timeout         time.Duration `mapstructure:"timeout"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisTimeouts struct {
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type RedisRetry struct {
	MaxRetries      int           `mapstructure:"max_retries"`
	MinRetryBackoff time.Duration `mapstructure:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff"`
}

type RedisCluster struct {
	Enabled bool     `mapstructure:"enabled"`
	Hosts   []string `mapstructure:"hosts"`
}

type RedisTLS struct {
	Enabled            bool `mapstructure:"enabled"`
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

func (cfg *Redis) validateAndLog() error {
	if cfg.Cluster.Enabled {
		// Cluster mode validation and logging
		if len(cfg.Cluster.Hosts) == 0 {
			return errors.New("Redis cluster is enabled but no hosts are provided")
		}
		log.Infof("config.backend.redis.cluster.enabled: %t", cfg.Cluster.Enabled)
		log.Infof("config.backend.redis.cluster.hosts: %v", cfg.Cluster.Hosts)
		if cfg.Db != 0 {
			log.Warnf("config.backend.redis.db: %d. Note that database selection is not supported in Redis cluster mode and will be ignored", cfg.Db)
		}
	} else {
		// Single-node mode validation and logging
		log.Infof("config.backend.redis.host: %s", cfg.Host)
		log.Infof("config.backend.redis.port: %d", cfg.Port)
		log.Infof("config.backend.redis.db: %d", cfg.Db)
	}

	// Common configuration logging
	if cfg.ExpirationMinutes > 0 {
		log.Infof("config.backend.redis.expiration: %d. Note that this configuration option is being deprecated in favor of config.request_limits.max_ttl_seconds", cfg.ExpirationMinutes)
	}
	log.Infof("config.backend.redis.tls.enabled: %t", cfg.TLS.Enabled)
	log.Infof("config.backend.redis.tls.insecure_skip_verify: %t", cfg.TLS.InsecureSkipVerify)

	// Validate and log pool configuration
	if cfg.Pool != nil {
		if cfg.Pool.Size < 0 {
			return errors.New("Redis pool size cannot be negative")
		}
		if cfg.Pool.MinIdleConns < 0 {
			return errors.New("Redis pool min_idle_conns cannot be negative")
		}
		if cfg.Pool.MaxIdleConns < 0 {
			return errors.New("Redis pool max_idle_conns cannot be negative")
		}
		if cfg.Pool.MinIdleConns > 0 && cfg.Pool.MaxIdleConns > 0 && cfg.Pool.MinIdleConns > cfg.Pool.MaxIdleConns {
			return errors.New("Redis pool min_idle_conns cannot be greater than max_idle_conns")
		}
		if cfg.Pool.Timeout < 0 {
			return errors.New("Redis pool timeout cannot be negative")
		}
		if cfg.Pool.ConnMaxIdleTime < 0 {
			return errors.New("Redis pool conn_max_idle_time cannot be negative")
		}
		if cfg.Pool.ConnMaxLifetime < 0 {
			return errors.New("Redis pool conn_max_lifetime cannot be negative")
		}

		log.Infof("config.backend.redis.pool.size: %d", cfg.Pool.Size)
		log.Infof("config.backend.redis.pool.timeout: %v", cfg.Pool.Timeout)
		log.Infof("config.backend.redis.pool.min_idle_conns: %d", cfg.Pool.MinIdleConns)
		log.Infof("config.backend.redis.pool.max_idle_conns: %d", cfg.Pool.MaxIdleConns)
		log.Infof("config.backend.redis.pool.conn_max_idle_time: %v", cfg.Pool.ConnMaxIdleTime)
		log.Infof("config.backend.redis.pool.conn_max_lifetime: %v", cfg.Pool.ConnMaxLifetime)
	}

	// Validate and log timeout configuration
	if cfg.Timeouts != nil {
		if cfg.Timeouts.DialTimeout < 0 {
			return errors.New("Redis dial_timeout cannot be negative")
		}
		if cfg.Timeouts.ReadTimeout < 0 {
			return errors.New("Redis read_timeout cannot be negative")
		}
		if cfg.Timeouts.WriteTimeout < 0 {
			return errors.New("Redis write_timeout cannot be negative")
		}

		log.Infof("config.backend.redis.timeouts.dial_timeout: %v", cfg.Timeouts.DialTimeout)
		log.Infof("config.backend.redis.timeouts.read_timeout: %v", cfg.Timeouts.ReadTimeout)
		log.Infof("config.backend.redis.timeouts.write_timeout: %v", cfg.Timeouts.WriteTimeout)
	}

	// Validate and log retry configuration
	if cfg.Retry != nil {
		if cfg.Retry.MaxRetries < 0 {
			return errors.New("Redis max_retries cannot be negative")
		}
		if cfg.Retry.MinRetryBackoff < 0 {
			return errors.New("Redis min_retry_backoff cannot be negative")
		}
		if cfg.Retry.MaxRetryBackoff < 0 {
			return errors.New("Redis max_retry_backoff cannot be negative")
		}
		if cfg.Retry.MinRetryBackoff > 0 && cfg.Retry.MaxRetryBackoff > 0 && cfg.Retry.MinRetryBackoff > cfg.Retry.MaxRetryBackoff {
			return errors.New("Redis min_retry_backoff cannot be greater than max_retry_backoff")
		}

		log.Infof("config.backend.redis.retry.max_retries: %d", cfg.Retry.MaxRetries)
		log.Infof("config.backend.redis.retry.min_retry_backoff: %v", cfg.Retry.MinRetryBackoff)
		log.Infof("config.backend.redis.retry.max_retry_backoff: %v", cfg.Retry.MaxRetryBackoff)
	}

	return nil
}

type Ignite struct {
	Scheme string `mapstructure:"scheme"`
	Host   string `mapstructure:"host"`
	Port   int    `mapstructure:"port"`
	// If VerifyCert is set to true, Prebid Cache verifies the SSL certificate on the Ignite server
	VerifyCert bool              `mapstructure:"secure"`
	Headers    map[string]string `mapstructure:"headers"`
	Cache      IgniteCache       `mapstructure:"cache"`
}

type IgniteCache struct {
	Name          string `mapstructure:"name"`
	CreateOnStart bool   `mapstructure:"create_on_start"`
}

func (cfg *Ignite) validateAndLog() error {
	if len(cfg.Scheme) == 0 {
		return errors.New("Cannot connect to Ignite: empty config.ignite.scheme")
	}
	if len(cfg.Host) == 0 {
		return errors.New("Cannot connect to Ignite: empty config.ignite.host")
	}
	if len(cfg.Cache.Name) == 0 {
		return errors.New("Cannot write nor read from Ignite: empty config.ignite.cachename")
	}
	log.Infof("config.backend.ignite.scheme: %s", cfg.Scheme)
	log.Infof("config.backend.ignite.host: %s", cfg.Host)
	log.Infof("config.backend.ignite.port: %d", cfg.Port)
	log.Infof("config.backend.ignite.cache.create_on_start: %t", cfg.VerifyCert)
	log.Infof("config.backend.ignite.cache.create_on_start: %v", cfg.Headers)
	log.Infof("config.backend.ignite.cache.name: %s", cfg.Cache.Name)
	log.Infof("config.backend.ignite.cache.create_on_start: %t", cfg.Cache.CreateOnStart)

	return nil
}
