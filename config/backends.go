package config

import log "github.com/Sirupsen/logrus"

type Backend struct {
	Type      BackendType `mapstructure:"type"`
	Aerospike Aerospike   `mapstructure:"aerospike"`
	Azure     Azure       `mapstructure:"azure"`
	Cassandra Cassandra   `mapstructure:"cassandra"`
	Memcache  Memcache    `mapstructure:"memcache"`
}

func (cfg *Backend) logValues() {
	log.Infof("config.backend.type: %s", cfg.Type)
	switch cfg.Type {
	case BackendAerospike:
		cfg.Aerospike.logValues()
	case BackendAzure:
		cfg.Azure.logValues()
	case BackendCassandra:
		cfg.Cassandra.logValues()
	case BackendMemcache:
		cfg.Memcache.logValues()
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

func (cfg *Aerospike) logValues() {
	log.Infof("config.backend.aerospike.host: %s", cfg.Host)
	log.Infof("config.backend.aerospike.port: %d", cfg.Port)
	log.Infof("config.backend.aerospike.namespace: %s", cfg.Namespace)
}

type Azure struct {
	Account string `mapstructure:"account"`
	Key     string `mapstructure:"key"`
}

func (cfg *Azure) logValues() {
	log.Infof("config.backend.azure.account: %s", cfg.Account)
	log.Infof("config.backend.azure.key: %s", cfg.Key)
}

type Cassandra struct {
	Hosts    string `mapstructure:"hosts"`
	Keyspace string `mapstructure:"keyspace"`
}

func (cfg *Cassandra) logValues() {
	log.Infof("config.backend.cassandra.hosts: %s", cfg.Hosts)
	log.Infof("config.backend.cassandra.keyspace: %s", cfg.Keyspace)
}

type Memcache struct {
	Hosts []string `mapstructure:"hosts"`
}

func (cfg *Memcache) logValues() {
	log.Infof("config.backend.memcache.hosts: %v", cfg.Hosts)
}
