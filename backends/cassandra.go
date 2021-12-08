package backends

import (
	"context"

	"github.com/gocql/gocql"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

type CassandraDB interface {
	Init() error
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error)
}

// CassandraDBClient is a wrapper for the Cassandra client 'gocql' that
// interacts with the Cassandra server and implements the CassandraDB interface
type CassandraDBClient struct {
	cfg     config.Cassandra
	cluster *gocql.ClusterConfig
	session *gocql.Session
}

// Get returns the value associated with the provided `key` parameter
func (c *CassandraDBClient) Get(ctx context.Context, key string) (string, error) {
	var res string

	err := c.session.Query(`SELECT value FROM cache WHERE key = ? LIMIT 1`, key).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&res)

	return res, err
}

// Put writes the `value` under the provided `key` in the Cassandra DB server
// only if it doesn't already exist. We make sure of this by adding the 'IF NOT EXISTS'
// clause to the 'INSERT' query
func (c *CassandraDBClient) Put(ctx context.Context, key string, value string, ttlSeconds int) (bool, error) {
	var insertedKey, insertedValue string

	return c.session.Query(`INSERT INTO cache (key, value) VALUES (?, ?) IF NOT EXISTS USING TTL ?`, key, value, ttlSeconds).
		WithContext(ctx).
		ScanCAS(&insertedKey, &insertedValue)
}

// Init initializes Cassandra cluster and session with the configuration
// loaded from environment variables or configuration files at startup
func (c *CassandraDBClient) Init() error {
	c.cluster = gocql.NewCluster(c.cfg.Hosts)
	c.cluster.Keyspace = c.cfg.Keyspace
	c.cluster.Consistency = gocql.LocalOne

	var err error
	c.session, err = c.cluster.CreateSession()

	return err
}

// CassandraBackend implements the Backend interface and get called from
// our Prebid Cache's endpoint handle functions.
type CassandraBackend struct {
	defaultTTL int
	client     CassandraDB
}

// NewCassandraBackend expects a valid config.Cassandra object
func NewCassandraBackend(cfg config.Cassandra) *CassandraBackend {
	backend := &CassandraBackend{
		cfg.DefaultTTL,
		&CassandraDBClient{cfg: cfg},
	}

	if err := backend.client.Init(); err != nil {
		log.Fatalf("Error creating Cassandra backend: %v", err)
		panic("Cassandra failure. This shouldn't happen.")
	}

	return backend
}

// Get makes the Cassandra client to retrieve the value that has been previously stored
// under 'key'. Returns KeyNotFoundError if no such key has ever been stored in the Cassandra
// database service
func (back *CassandraBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := back.client.Get(ctx, key)
	if err == gocql.ErrNotFound {
		err = utils.KeyNotFoundError{}
	}

	return res, err
}

// Put makes the Cassandra client to store `value` only if `key` doesn't
// exist in the storage already. If it does, no operation is performed and Put
// returns RecordExistsError
func (back *CassandraBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	applied, err := back.client.Put(ctx, key, value, ttlSeconds)
	if !applied {
		return utils.RecordExistsError{}
	}
	return err
}
