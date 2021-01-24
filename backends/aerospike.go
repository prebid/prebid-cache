package backends

import (
	"context"
	"errors"
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
	ase "github.com/aerospike/aerospike-client-go/types"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	log "github.com/sirupsen/logrus"
)

const SET_NAME = "uuid"
const BIN_VALUE = "value"

type AerospikeDBClient interface {
	NewUuidKey(namespace string, key string) (*as.Key, error)
	Get(key *as.Key) (*as.Record, error)
	Put(key *as.Key, value string, ttlSeconds int) error
}

type Aerospike struct {
	client *as.Client
}

func (db Aerospike) Get(key *as.Key) (*as.Record, error) {
	rec, err := db.client.Get(nil, key, BIN_VALUE)
	if err != nil {
		return nil, printAerospikeError(err, "GET")
	}
	return rec, nil
}

func (db Aerospike) Put(key *as.Key, value string, ttlSeconds int) error {
	bins := as.BinMap{BIN_VALUE: value}
	policy := &as.WritePolicy{Expiration: uint32(ttlSeconds)}

	err := db.client.Put(policy, key, bins)

	return printAerospikeError(err, "PUT")
}

func (db *Aerospike) NewUuidKey(namespace string, key string) (*as.Key, error) {
	asKey, err := as.NewKey(namespace, SET_NAME, key)
	if err != nil {
		return nil, printAerospikeError(err, "NEW_KEY")
	}
	return asKey, nil
}

type AerospikeBackend struct {
	cfg     config.Aerospike
	client  AerospikeDBClient
	metrics *metrics.Metrics
}

func NewAerospikeBackend(cfg config.Aerospike, metrics *metrics.Metrics) *AerospikeBackend {
	if cfg.Host == "" {
		log.Fatalf("Cannot connect to empty Aerospike host")
	}
	if cfg.Port <= 0 {
		log.Fatalf("Cannot connect to Aerospike host at port %d", cfg.Port)
	}
	client, err := as.NewClient(cfg.Host, cfg.Port)
	if err != nil {
		log.Fatalf("Error creating Aerospike backend: %v", printAerospikeError(err, "NewAerospikeBackend"))
		panic("AerospikeBackend failure. This shouldn't happen.")
	}
	log.Infof("Connected to Aerospike at %s:%d", cfg.Host, cfg.Port)

	return &AerospikeBackend{
		cfg:     cfg,
		client:  &Aerospike{client},
		metrics: metrics,
	}
}

func (a *AerospikeBackend) Get(ctx context.Context, key string) (string, error) {
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, key)
	if err != nil {
		return "", err
	}
	rec, err := a.client.Get(asKey)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", errors.New("Aerospike GET. Nil record")
	}
	a.metrics.RecordExtraTTLSeconds(float64(rec.Expiration))

	value, found := rec.Bins[BIN_VALUE]
	if !found {
		return "", errors.New("Aerospike GET. No 'value' bucket found")
	}

	str, isString := value.(string)
	if !isString {
		return "", errors.New("Aerospike GET. Retrieved value is not a string")
	}

	return str, nil
}

func (a *AerospikeBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, SET_NAME)
	if err != nil {
		return err
	}
	if ttlSeconds == 0 {
		ttlSeconds = a.cfg.DefaultTTL
	}
	return a.client.Put(asKey, value, ttlSeconds)
}

func printAerospikeError(err error, caller string) error {
	if err != nil {
		if aerr, ok := err.(ase.AerospikeError); ok {
			return fmt.Errorf("%s Aerospike error: %s. Code: %d", caller, aerr.Error(), aerr.ResultCode())
		}
	}
	return err
}
