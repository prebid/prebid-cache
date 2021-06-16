package backends

import (
	"context"

	"github.com/gocql/gocql"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

type CassandraDB interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string, ttlSeconds int) error
	Init() error
}

// CassandraDBClient is a wrapper for the Cassandra client that
// interacts with the Cassandra server and implements the CassandraDB client
type CassandraDBClient struct {
	cfg     config.Cassandra
	cluster *gocql.ClusterConfig
	session *gocql.Session
}

func (c *CassandraDBClient) Get(ctx context.Context, key string) (string, error) {
	var res string

	err := c.session.Query(`SELECT value FROM cache WHERE key = ? LIMIT 1`, key).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&res)

	return res, err
}

func (c *CassandraDBClient) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return c.session.Query(`INSERT INTO cache (key, value) VALUES (?, ?) USING TTL ?`, key, value, ttlSeconds).
		WithContext(ctx).
		Exec()
}

// Init initializes cassandra cluster and session with the configuration
// loaded from environment variables or configuration files at startup
func (c *CassandraDBClient) Init() error {
	c.cluster = gocql.NewCluster(c.cfg.Hosts)
	c.cluster.Keyspace = c.cfg.Keyspace
	c.cluster.Consistency = gocql.LocalOne

	var err error
	c.session, err = c.cluster.CreateSession()

	return err
}

//------------------------------------------------------------------------

// CassandraBackend implements the Backend interface
type CassandraBackend struct {
	defaultTTL int
	client     CassandraDB
}

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

func (back *CassandraBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := back.client.Get(ctx, key)
	if err == gocql.ErrNotFound {
		err = utils.KeyNotFoundError{}
	}

	return res, err
}

func (back *CassandraBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = back.defaultTTL
	}
	return back.client.Put(ctx, key, value, ttlSeconds)
}
