package backends

import (
	"context"
	"errors"

	as "github.com/aerospike/aerospike-client-go/v6"
	as_types "github.com/aerospike/aerospike-client-go/v6/types"
	"github.com/go-redis/redis/v8"
	"github.com/google/gomemcache/memcache"
	"github.com/prebid/prebid-cache/utils"
)

// ------------------------------------------
// Aerospike client mocks
// ------------------------------------------
func NewMockAerospikeBackend(mockClient AerospikeDB) *AerospikeBackend {
	return &AerospikeBackend{client: mockClient}
}

type ErrorProneAerospikeClient struct {
	ServerError string
}

func (c *ErrorProneAerospikeClient) NewUUIDKey(namespace string, key string) (*as.Key, error) {
	if c.ServerError == "TEST_KEY_GEN_ERROR" {
		return nil, &as.AerospikeError{ResultCode: as_types.NOT_AUTHENTICATED}
	}
	return nil, nil
}

func (c *ErrorProneAerospikeClient) Get(key *as.Key) (*as.Record, error) {
	if c.ServerError == "TEST_GET_ERROR" {
		return nil, &as.AerospikeError{ResultCode: as_types.KEY_NOT_FOUND_ERROR}
	} else if c.ServerError == "TEST_NO_BUCKET_ERROR" {
		return &as.Record{Bins: as.BinMap{"AnyKey": "any_value"}}, nil
	} else if c.ServerError == "TEST_NON_STRING_VALUE_ERROR" {
		return &as.Record{Bins: as.BinMap{binValue: 0.0}}, nil
	}
	return nil, nil
}

func (c *ErrorProneAerospikeClient) Put(policy *as.WritePolicy, key *as.Key, binMap as.BinMap) error {
	if c.ServerError == "TEST_PUT_ERROR" {
		return &as.AerospikeError{ResultCode: as_types.KEY_EXISTS_ERROR}
	}
	return nil
}

// Aerospike client that does not throw errors
type GoodAerospikeClient struct {
	StoredData map[string]string
}

func (c *GoodAerospikeClient) Get(aeKey *as.Key) (*as.Record, error) {
	if aeKey != nil && aeKey.Value() != nil {
		key := aeKey.Value().String()

		if value, found := c.StoredData[key]; found {
			rec := &as.Record{
				Bins: as.BinMap{binValue: value},
			}
			return rec, nil
		}
	}
	return nil, &as.AerospikeError{ResultCode: as_types.KEY_NOT_FOUND_ERROR}
}

func (c *GoodAerospikeClient) Put(policy *as.WritePolicy, aeKey *as.Key, binMap as.BinMap) error {
	if aeKey != nil && aeKey.Value() != nil {
		key := aeKey.Value().String()
		if interfaceValue, found := binMap[binValue]; found {
			if str, asserted := interfaceValue.(string); asserted {
				c.StoredData[key] = str
			}
		}

		return nil
	}
	return &as.AerospikeError{ResultCode: as_types.KEY_MISMATCH}
}

func (c *GoodAerospikeClient) NewUUIDKey(namespace string, key string) (*as.Key, error) {
	return as.NewKey(namespace, setName, key)
}

// ------------------------------------------
// Cassandra client mocks
// ------------------------------------------
func NewMockCassandraBackend(ttl int, mockClient CassandraDB) *CassandraBackend {
	return &CassandraBackend{
		defaultTTL: ttl,
		client:     mockClient,
	}
}

type ErrorProneCassandraClient struct {
	Applied     bool
	ServerError error
}

func (ec *ErrorProneCassandraClient) Init() error {
	return errors.New("init error")
}

func (ec *ErrorProneCassandraClient) Get(ctx context.Context, key string) (string, error) {
	return "", ec.ServerError
}

func (ec *ErrorProneCassandraClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	return ec.Applied, ec.ServerError
}

// Cassandra client client that does not throw errors
type GoodCassandraClient struct {
	StoredData map[string]string
}

func (gc *GoodCassandraClient) Init() error {
	return nil
}

func (gc *GoodCassandraClient) Get(ctx context.Context, key string) (string, error) {
	if value, found := gc.StoredData[key]; found {
		return value, nil
	}
	return "", utils.NewPBCError(utils.KEY_NOT_FOUND)
}

func (gc *GoodCassandraClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	if _, found := gc.StoredData[key]; !found {
		gc.StoredData[key] = value
	}
	return true, nil
}

// ------------------------------------------
// Memcache client mocks
// ------------------------------------------
func NewMockMemcacheBackend(mockClient MemcacheDataStore) *MemcacheBackend {
	return &MemcacheBackend{
		memcache: mockClient,
	}
}

type ErrorProneMemcache struct {
	ServerError error
}

func (ec *ErrorProneMemcache) Get(key string) (*memcache.Item, error) {
	return nil, ec.ServerError
}

func (ec *ErrorProneMemcache) Put(key string, value string, ttlSeconds int) error {
	return ec.ServerError
}

// Memcache client that does not throw errors
type GoodMemcache struct {
	StoredData map[string]string
}

func (gm *GoodMemcache) Get(key string) (*memcache.Item, error) {
	if value, found := gm.StoredData[key]; found {
		return &memcache.Item{Key: key, Value: []byte(value)}, nil
	}
	return nil, utils.NewPBCError(utils.KEY_NOT_FOUND)
}

func (gm *GoodMemcache) Put(key string, value string, ttlSeconds int) error {
	if _, found := gm.StoredData[key]; !found {
		gm.StoredData[key] = value
	}
	return nil
}

// ------------------------------------------
// Redis client mocks
// ------------------------------------------
func NewMockRedisBackend(mockClient RedisDB) *RedisBackend {
	return &RedisBackend{
		client: mockClient,
	}
}

type ErrorProneRedisClient struct {
	Success     bool
	ServerError error
}

func (ec *ErrorProneRedisClient) Get(ctx context.Context, key string) (string, error) {
	return "", ec.ServerError
}

func (ec *ErrorProneRedisClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	return ec.Success, ec.ServerError
}

// GoodRedisClient does not throw errors
type GoodRedisClient struct {
	StoredData map[string]string
}

func (gr *GoodRedisClient) Get(ctx context.Context, key string) (string, error) {
	if value, found := gr.StoredData[key]; found {
		return value, nil
	}
	return "", utils.NewPBCError(utils.KEY_NOT_FOUND)
}

func (gr *GoodRedisClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	if _, found := gr.StoredData[key]; !found {
		gr.StoredData[key] = value
	}
	return true, redis.Nil
}

// ------------------------------------------
// Memory client mocks
// ------------------------------------------
func NewErrorResponseMemoryBackend() *ErrorProneMemoryClient {
	return &ErrorProneMemoryClient{}
}

type ErrorProneMemoryClient struct{}

func (ec *ErrorProneMemoryClient) Get(ctx context.Context, key string) (string, error) {
	return "", errors.New("Bakend error")
}

func (ec *ErrorProneMemoryClient) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return errors.New("Bakend error")
}

// Good memory client does not throw errors
func NewMemoryBackendWithValues(customData map[string]string) (*MemoryBackend, error) {
	backend := NewMemoryBackend()

	if len(customData) > 0 {
		for k, v := range customData {
			if err := backend.Put(context.Background(), k, v, 1); err != nil {
				return backend, err
			}
		}
	}
	return backend, nil
}
