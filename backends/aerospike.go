package backends

import (
	"context"
	as "github.com/aerospike/aerospike-client-go"
	"github.com/golang/snappy"
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
	a.client = client
	return a, nil
}

func (a *Aerospike) Get(ctx context.Context, key string) (string, error) {
	asKey, err := as.NewKey(a.config.namespace, "uuid", key)
	if err != nil {
		return "", err
	}
	rec, err := a.client.Get(nil, asKey)
	if err != nil {
		return "", err
	}
	val, err := snappy.Decode(nil, rec.Bins["value"].([]byte))
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (a *Aerospike) Put(ctx context.Context, key string, value string) error {
	asKey, err := as.NewKey(a.config.namespace, "uuid", key)
	if err != nil {
		return err
	}
	val := snappy.Encode(nil, []byte(value))
	bins := as.BinMap{
		"value": val,
	}
	err = a.client.Put(nil, asKey, bins)
	if err != nil {
		return err
	}
	return nil
}
