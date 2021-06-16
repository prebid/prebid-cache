package backends

import (
	"context"
	"errors"

	as "github.com/aerospike/aerospike-client-go"
	as_types "github.com/aerospike/aerospike-client-go/types"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prebid/prebid-cache/utils"
)

// Mock Aerospike client that always throws an error
type errorProneAerospikeClient struct {
	errorThrowingFunction string
}

func NewErrorProneAerospikeClient(funcName string) *errorProneAerospikeClient {
	return &errorProneAerospikeClient{
		errorThrowingFunction: funcName,
	}
}

func (c *errorProneAerospikeClient) NewUuidKey(namespace string, key string) (*as.Key, error) {
	if c.errorThrowingFunction == "TEST_KEY_GEN_ERROR" {
		return nil, as_types.NewAerospikeError(as_types.NOT_AUTHENTICATED)
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Get(key *as.Key) (*as.Record, error) {
	if c.errorThrowingFunction == "TEST_GET_ERROR" {
		return nil, as_types.NewAerospikeError(as_types.KEY_NOT_FOUND_ERROR)
	} else if c.errorThrowingFunction == "TEST_NO_BUCKET_ERROR" {
		return &as.Record{Bins: as.BinMap{"AnyKey": "any_value"}}, nil
	} else if c.errorThrowingFunction == "TEST_NON_STRING_VALUE_ERROR" {
		return &as.Record{Bins: as.BinMap{binValue: 0.0}}, nil
	}
	return nil, nil
}

func (c *errorProneAerospikeClient) Put(policy *as.WritePolicy, key *as.Key, binMap as.BinMap) error {
	if c.errorThrowingFunction == "TEST_PUT_ERROR" {
		return as_types.NewAerospikeError(as_types.KEY_EXISTS_ERROR)
	}
	return nil
}

// Mock Aerospike client that does not throw errors
type goodAerospikeClient struct {
	records map[string]*as.Record
}

func NewGoodAerospikeClient() *goodAerospikeClient {
	return &goodAerospikeClient{
		records: map[string]*as.Record{
			"defaultKey": &as.Record{
				Bins: as.BinMap{binValue: "Default value"},
			},
		},
	}
}

func (c *goodAerospikeClient) Get(aeKey *as.Key) (*as.Record, error) {
	if aeKey != nil && aeKey.Value() != nil {

		key := aeKey.Value().String()

		if rec, found := c.records[key]; found {
			return rec, nil
		}
	}
	return nil, as_types.NewAerospikeError(as_types.KEY_NOT_FOUND_ERROR)
}

func (c *goodAerospikeClient) Put(policy *as.WritePolicy, aeKey *as.Key, binMap as.BinMap) error {
	if aeKey != nil && aeKey.Value() != nil {
		key := aeKey.Value().String()
		c.records[key] = &as.Record{
			Bins: binMap,
		}
		return nil
	}
	return as_types.NewAerospikeError(as_types.KEY_MISMATCH)
}

func (c *goodAerospikeClient) NewUuidKey(namespace string, key string) (*as.Key, error) {
	return as.NewKey(namespace, setName, key)
}

//------------------------------------------------------------------------

// Mock Cassandra client that always throws an error
type errorProneCassandraClient struct {
	errorToThrow error
}

func NewErrorProneCassandraClient(errorToThrow error) *errorProneCassandraClient {
	return &errorProneCassandraClient{errorToThrow}
}

func (ec *errorProneCassandraClient) Init() error {
	return errors.New("init error")
}

func (ec *errorProneCassandraClient) Get(ctx context.Context, key string) (string, error) {
	return "", ec.errorToThrow
}

func (ec *errorProneCassandraClient) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return ec.errorToThrow
}

// Mock Cassandra client client that does not throw errors
type goodCassandraClient struct {
	key   string
	value string
}

func NewGoodCassandraClient(key string, value string) *goodCassandraClient {
	return &goodCassandraClient{key, value}
}

func (gc *goodCassandraClient) Init() error {
	return nil
}

func (gc *goodCassandraClient) Get(ctx context.Context, key string) (string, error) {
	if key == gc.key {
		return gc.value, nil
	}
	return "", utils.KeyNotFoundError{}
}

func (gc *goodCassandraClient) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if gc.key != key {
		gc.key = key
	}
	gc.value = value

	return nil
}

//------------------------------------------------------------------------

// Mock Redis client that always throws an error
type errorProneRedisClient struct {
	errorToThrow error
}

func NewErrorProneRedisClient(errorToThrow error) *errorProneRedisClient {
	return &errorProneRedisClient{errorToThrow}
}

func (ec *errorProneRedisClient) Get(key string) (string, error) {
	return "", ec.errorToThrow
}

func (ec *errorProneRedisClient) Put(key string, value string, ttlSeconds int) error {
	return ec.errorToThrow
}

// Mock Redis client client that does not throw errors
type goodRedisClient struct {
	key   string
	value string
}

func NewGoodRedisClient(key string, value string) *goodRedisClient {
	return &goodRedisClient{key, value}
}

func (gc *goodRedisClient) Get(key string) (string, error) {
	if key == gc.key {
		return gc.value, nil
	}
	return "", utils.KeyNotFoundError{}
}

func (gc *goodRedisClient) Put(key string, value string, ttlSeconds int) error {
	if gc.key != key {
		gc.key = key
	}
	gc.value = value

	return nil
}

//------------------------------------------------------------------------

// Mock Memcache that always throws an error
type errorProneMemcache struct {
	errorToThrow error
}

func NewErrorProneMemcache(errorToThrow error) *errorProneMemcache {
	return &errorProneMemcache{errorToThrow}
}

func (ec *errorProneMemcache) Get(key string) (*memcache.Item, error) {
	return nil, ec.errorToThrow
}

func (ec *errorProneMemcache) Put(key string, value string, ttlSeconds int) error {
	return ec.errorToThrow
}

// Mock Memcache client that does not throw errors
type goodMemcache struct {
	key   string
	value string
}

func NewGoodMemcache(key string, value string) *goodMemcache {
	return &goodMemcache{key, value}
}

func (gc *goodMemcache) Get(key string) (*memcache.Item, error) {
	if key == gc.key {
		return &memcache.Item{Key: gc.key, Value: []byte(gc.value)}, nil
	}
	return nil, utils.KeyNotFoundError{}
}

func (gc *goodMemcache) Put(key string, value string, ttlSeconds int) error {
	if gc.key != key {
		gc.key = key
	}
	gc.value = value

	return nil
}
