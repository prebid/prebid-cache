package config

import log "github.com/Sirupsen/logrus"

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Azure     Azure       `mapstructure:"azure"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
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
	case BackendMemory:
	default:
		log.Fatalf(`invalid config.backend.type: %s. It must be "aerospike", "azure", "cassandra", "memcache", or "memory".`, cfg.Type)
	}
}

type BackendType string

const (
	BackendAerospike BackendType = "aerospike"
	BackendAzure     BackendType = "azure"
	BackendCassandra BackendType = "cassandra"
	BackendMemcache  BackendType = "memcache"
	BackendMemory    BackendType = "memory"
)

type Aerospike struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
}

func (cfg *Aerospike) validateAndLog() {
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
