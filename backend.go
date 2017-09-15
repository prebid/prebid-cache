package main

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

// Backend interface for storing data
type Backend interface {
	Put(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
}

func NewBackend(backendType string) Backend {
	switch backendType {
	case "cassandra":
		c := CassandraConfig{
			hosts:    viper.GetString("backend.cassandra.hosts"),
			keyspace: viper.GetString("backend.cassandra.keyspace"),
		}
		var backend, err = NewCassandraBackend(&c)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
		return backend
	case "memory":
		return NewMemoryBackend()
	case "azure":
		return NewAzureBackend(
			viper.GetString("backend.azure.account"),
			viper.GetString("backend.azure.key"))
	default:
		panic("Unknown backend")
	}
}
