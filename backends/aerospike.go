package backends

import (
	"context"
	"errors"
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	log "github.com/sirupsen/logrus"
)

const binValue = "value"

// Wrapper for the Aerospike client
type AerospikeDB interface {
	NewUuidKey(namespace string, set string, key string) (*as.Key, error)
	Get(key *as.Key) (*as.Record, error)
	Put(policy *as.WritePolicy, key *as.Key, binMap as.BinMap) error
}

type AerospikeDBClient struct {
	client *as.Client
}

func (db AerospikeDBClient) Get(key *as.Key) (*as.Record, error) {
	return db.client.Get(nil, key, binValue)
}

func (db AerospikeDBClient) Put(policy *as.WritePolicy, key *as.Key, binMap as.BinMap) error {
	return db.client.Put(policy, key, binMap)
}

func (db *AerospikeDBClient) NewUuidKey(namespace string, set string, key string) (*as.Key, error) {
	return as.NewKey(namespace, set, key)
}

// Instantiates, and configures the Aerospike client, it also performs Get and Put operations and monitors results
type AerospikeBackend struct {
	cfg     config.Aerospike
	client  AerospikeDB
	metrics *metrics.Metrics
}

func NewAerospikeBackend(cfg config.Aerospike, metrics *metrics.Metrics) *AerospikeBackend {
	client, err := as.NewClient(cfg.Host, cfg.Port)
	if err != nil {
		log.Fatalf("%v", formatAerospikeError(err).Error())
		panic("AerospikeBackend failure. This shouldn't happen.")
	}
	log.Infof("Connected to Aerospike at %s:%d", cfg.Host, cfg.Port)

	return &AerospikeBackend{
		cfg:     cfg,
		client:  &AerospikeDBClient{client},
		metrics: metrics,
	}
}

func (a *AerospikeBackend) Get(ctx context.Context, key string) (string, error) {
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, a.cfg.Set, key)
	if err != nil {
		return "", formatAerospikeError(err, "GET")
	}
	rec, err := a.client.Get(asKey)
	if err != nil {
		return "", formatAerospikeError(err, "GET")
	}
	if rec == nil {
		return "", formatAerospikeError(errors.New("Nil record"), "GET")
	}
	a.metrics.RecordExtraTTLSeconds(float64(rec.Expiration))

	value, found := rec.Bins[binValue]
	if !found {
		return "", formatAerospikeError(errors.New("No 'value' bucket found"), "GET")
	}

	str, isString := value.(string)
	if !isString {
		return "", formatAerospikeError(errors.New("Unexpected non-string value found"), "GET")
	}

	return str, nil
}

func (a *AerospikeBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, a.cfg.Set, key)
	if err != nil {
		return formatAerospikeError(err, "PUT")
	}

	if ttlSeconds == 0 {
		ttlSeconds = a.cfg.DefaultTTL
	}

	bins := as.BinMap{binValue: value}
	policy := &as.WritePolicy{Expiration: uint32(ttlSeconds)}

	if err := a.client.Put(policy, asKey, bins); err != nil {
		return formatAerospikeError(err, "PUT")
	}

	return nil
}

func formatAerospikeError(err error, caller ...string) error {
	if err != nil {
		msg := "Aerospike"
		for _, str := range caller {
			if len(str) > 0 {
				msg = fmt.Sprintf("%s %s", msg, str)
			}
		}
		return fmt.Errorf("%s: %s", msg, err.Error())
	}
	return err
}
