package config

import (
	log "github.com/Sirupsen/logrus"
)

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Azure     Azure       `mapstructure:"azure"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
	Redis     Redis       `mapstructure:"redis"`
}

func (cfg *Backend) validateAndLog() {
	log.Infof("config.backend.type: %s", cfg.Type)
	switch cfg.Type {
	case BackendAerospike:
		cfg.Aerospike.validateAndLog()
	case BackendAzure:
		cfg.Azure.validateAndLog()
	case BackendCassandra:
		cfg.Cassandra.validateAndLog()
	case BackendMemcache:
		cfg.Memcache.validateAndLog()
	case BackendRedis:
		cfg.Redis.validateAndLog()
	case BackendMemory:
	default:
		log.Fatalf(`invalid config.backend.type: %s. It must be "aerospike", "azure", "cassandra", "memcache", "redis", or "memory".`, cfg.Type)
	}
}

type BackendType string

const (
	BackendAerospike BackendType = "aerospike"
	BackendAzure     BackendType = "azure"
	BackendCassandra BackendType = "cassandra"
	BackendMemcache  BackendType = "memcache"
	BackendMemory    BackendType = "memory"
	BackendRedis     BackendType = "redis"
)

type Aerospike struct {
	DefaultTTL int    `mapstructure:"default_ttl_seconds"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Namespace  string `mapstructure:"namespace"`
}

func (cfg *Aerospike) validateAndLog() {
	log.Infof("config.backend.aerospike.default_ttl_seconds: %d", cfg.DefaultTTL)
	log.Infof("config.backend.aerospike.host: %s", cfg.Host)
	log.Infof("config.backend.aerospike.port: %d", cfg.Port)
	log.Infof("config.backend.aerospike.namespace: %s", cfg.Namespace)
}

type Azure struct {
	Account string `mapstructure:"account"`
	Key     string `mapstructure:"key"`
}

func (cfg *Azure) validateAndLog() {
	log.Infof("config.backend.azure.account: %s", cfg.Account)
	log.Infof("config.backend.azure.key: %s", cfg.Key)
}

type Cassandra struct {
	Hosts    string `mapstructure:"hosts"`
	Keyspace string `mapstructure:"keyspace"`
}

func (cfg *Cassandra) validateAndLog() {
	log.Infof("config.backend.cassandra.hosts: %s", cfg.Hosts)
	log.Infof("config.backend.cassandra.keyspace: %s", cfg.Keyspace)
}

type Memcache struct {
	Hosts []string `mapstructure:"hosts"`
}

func (cfg *Memcache) validateAndLog() {
	log.Infof("config.backend.memcache.hosts: %v", cfg.Hosts)
}

type Redis struct {
	Host       string   `mapstructure:"host"`
	Port       int      `mapstructure:"port"`
	Password   string   `mapstructure:"password"`
	Db         int      `mapstructure:"db"`
	Expiration int      `mapstructure:"expiration"`
	Tls        RedisTLS `mapstructure:"tls"`
}

type RedisTLS struct {
	Enabled            bool `mapstructure:"enabled"`
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

func (cfg *Redis) validateAndLog() {
	log.Infof("config.backend.redis.host: %s", cfg.Host)
	log.Infof("config.backend.redis.port: %d", cfg.Port)
	log.Infof("config.backend.redis.db: %d", cfg.Db)
	log.Infof("config.backend.redis.expiration: %d", cfg.Expiration)
	log.Infof("config.backend.redis.tls.enabled: %t", cfg.Tls.Enabled)
	log.Infof("config.backend.redis.tls.insecure_skip_verify: %t", cfg.Tls.InsecureSkipVerify)
}
