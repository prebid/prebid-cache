package backends

import (
	"context"
	"errors"

	as "github.com/aerospike/aerospike-client-go"
	as_types "github.com/aerospike/aerospike-client-go/types"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	log "github.com/sirupsen/logrus"
)

const setName = "uuid"
const binValue = "value"

// Wrapper for the Aerospike client
type AerospikeDB interface {
	NewUuidKey(namespace string, key string) (*as.Key, error)
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

func (db *AerospikeDBClient) NewUuidKey(namespace string, key string) (*as.Key, error) {
	return as.NewKey(namespace, setName, key)
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
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, key)
	if err != nil {
		return "", formatAerospikeError(err)
	}
	rec, err := a.client.Get(asKey)
	if err != nil {
		return "", formatAerospikeError(err)
	}
	if rec == nil {
		return "", formatAerospikeError(errors.New("Nil record"))
	}
	a.metrics.RecordExtraTTLSeconds(float64(rec.Expiration))

	value, found := rec.Bins[binValue]
	if !found {
		return "", formatAerospikeError(errors.New("No 'value' bucket found"))
	}

	str, isString := value.(string)
	if !isString {
		return "", formatAerospikeError(errors.New("Unexpected non-string value found"))
	}

	return str, nil
}

func (a *AerospikeBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	asKey, err := a.client.NewUuidKey(a.cfg.Namespace, key)
	if err != nil {
		return formatAerospikeError(err)
	}

	if ttlSeconds == 0 {
		ttlSeconds = a.cfg.DefaultTTL
	}

	bins := as.BinMap{binValue: value}
	policy := &as.WritePolicy{Expiration: uint32(ttlSeconds)}

	if err := a.client.Put(policy, asKey, bins); err != nil {
		return formatAerospikeError(err)
	}

	return nil
}

func formatAerospikeError(err error) error {
	if err != nil {
		msg := "Aerospike "

		if aerr, ok := err.(as_types.AerospikeError); ok {
			if aerr.ResultCode() == as_types.KEY_NOT_FOUND_ERROR {
				return KeyNotFoundError{msg}
			}
		}
		return errors.New(msg + err.Error())
	}
	return err
}
