package backends

import (
	"context"
	"errors"

	log "github.com/Sirupsen/logrus"
	as "github.com/aerospike/aerospike-client-go"
)

type AerospikeConfig struct {
	host      string
	port      int
	namespace string
}

type Aerospike struct {
	config *AerospikeConfig
	client *as.Client
}

func NewAerospikeBackend(config *AerospikeConfig) (*Aerospike, error) {
	a := &Aerospike{}
	a.config = config
	client, err := as.NewClient(config.host, config.port)
	if err != nil {
		return nil, err
	}
	log.Infof("Connected to Aerospike at %s:%d", config.host, config.port)

	a.client = client
	return a, nil
}

func (a *Aerospike) Get(ctx context.Context, key string) (string, error) {
	asKey, err := as.NewKey(a.config.namespace, "uuid", key)
	if err != nil {
		return "", err
	}
	rec, err := a.client.Get(nil, asKey, "value")
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", errors.New("client.Get returned a nil record. Is aerospike configured properly?")
	}
	return rec.Bins["value"].(string), nil
}

func (a *Aerospike) Put(ctx context.Context, key string, value string) error {
	asKey, err := as.NewKey(a.config.namespace, "uuid", key)
	if err != nil {
		return err
	}
	bins := as.BinMap{
		"value": value,
	}
	err = a.client.Put(nil, asKey, bins)
	if err != nil {
		return err
	}
	return nil
}
