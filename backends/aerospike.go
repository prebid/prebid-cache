package backends

import (
	"context"
	"errors"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/PubMatic-OpenWrap/prebid-cache/metrics"
	"github.com/PubMatic-OpenWrap/prebid-cache/stats"
	"github.com/PubMatic-OpenWrap/prebid-cache/utils"
	as "github.com/aerospike/aerospike-client-go"
	as_types "github.com/aerospike/aerospike-client-go/types"
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
	namespace string
	client    AerospikeDB
	metrics   *metrics.Metrics
}

func NewAerospikeBackend(cfg config.Aerospike, metrics *metrics.Metrics) *AerospikeBackend {
	var hosts []*as.Host

	clientPolicy := as.NewClientPolicy()
	// cfg.User and cfg.Password are optional parameters
	// if left blank in the config, they will default to the empty
	// string and be ignored
	clientPolicy.User = cfg.User
	clientPolicy.Password = cfg.Password

	if len(cfg.Host) > 1 {
		hosts = append(hosts, as.NewHost(cfg.Host, cfg.Port))
		logger.Info("config.backend.aerospike.host is being deprecated in favor of config.backend.aerospike.hosts")
	}
	for _, host := range cfg.Hosts {
		hosts = append(hosts, as.NewHost(host, cfg.Port))
	}

	client, err := as.NewClientWithPolicyAndHost(clientPolicy, hosts...)
	if err != nil {
		stats.LogAerospikeErrorStats()
		logger.Fatal("Error creating Aerospike backend: %+v", err)
		panic("AerospikeBackend failure. This shouldn't happen.")
	}
	logger.Info("Connected to Aerospike host(s) %v on port %d", append(cfg.Hosts, cfg.Host), cfg.Port)

	return &AerospikeBackend{
		namespace: cfg.Namespace,
		client:    &AerospikeDBClient{client},
		metrics:   metrics,
	}
}

func classifyAerospikeError(err error) error {
	if err != nil {
		if aerr, ok := err.(as_types.AerospikeError); ok {
			if aerr.ResultCode() == as_types.KEY_NOT_FOUND_ERROR {
				return utils.KeyNotFoundError{}
			}
			if aerr.ResultCode() == as_types.KEY_EXISTS_ERROR {
				return utils.RecordExistsError{}
			}
		}
	}
	return err
}

func (a *AerospikeBackend) Get(ctx context.Context, key string) (string, error) {
	aerospikeStartTime := time.Now()
	asKey, err := a.client.NewUuidKey(a.namespace, key)
	if err != nil {
		return "", classifyAerospikeError(err)
	}
	rec, err := a.client.Get(asKey)
	if err != nil {
		return "", classifyAerospikeError(err)
	}
	if rec == nil {
		return "", errors.New("Nil record")
	}

	value, found := rec.Bins[binValue]
	if !found {
		return "", errors.New("No 'value' bucket found")
	}
	logger.Info("Time taken by Aerospike for get: %v", time.Now().Sub(aerospikeStartTime))

	str, isString := value.(string)
	if !isString {
		return "", errors.New("Unexpected non-string value found")
	}

	return str, nil
}

func (a *AerospikeBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	aerospikeStartTime := time.Now()

	asKey, err := a.client.NewUuidKey(a.namespace, key)
	if err != nil {
		return classifyAerospikeError(err)
	}

	bins := as.BinMap{binValue: value}
	policy := &as.WritePolicy{
		Expiration:         uint32(ttlSeconds),
		RecordExistsAction: as.CREATE_ONLY,
	}

	if err := a.client.Put(policy, asKey, bins); err != nil {
		return classifyAerospikeError(err)
	}

	logger.Info("Time taken by Aerospike for put: %v", time.Now().Sub(aerospikeStartTime))
	return nil
}
