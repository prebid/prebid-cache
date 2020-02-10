package backends

import (
	"context"

	"github.com/gocql/gocql"
	"github.com/prebid/prebid-cache/config"
	log "github.com/sirupsen/logrus"
)

// Cassandra Object use to implement backend interface
type Cassandra struct {
	cluster *gocql.ClusterConfig
	session *gocql.Session
}

// NewCassandraBackend create a new cassandra backend
func NewCassandraBackend(cfg config.Cassandra) *Cassandra {
	var err error

	c := &Cassandra{}
	c.cluster = gocql.NewCluster(cfg.Hosts)
	c.cluster.Keyspace = cfg.Keyspace
	c.cluster.Consistency = gocql.LocalOne

	c.session, err = c.cluster.CreateSession()
	if err != nil {
		log.Fatalf("Error creating Cassandra backend: %v", err)
		panic("Cassandra failure. This shouldn't happen.")
	}

	return c
}

func (c *Cassandra) Get(ctx context.Context, key string) (string, error) {
	var res string
	err := c.session.Query(`SELECT value FROM cache WHERE key = ? LIMIT 1`, key).
		WithContext(ctx).
		Consistency(gocql.One).
		Scan(&res)

	return res, err
}

func (c *Cassandra) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = 2400
	}
	err := c.session.Query(`INSERT INTO cache (key, value) VALUES (?, ?) USING TTL ?`, key, value, ttlSeconds).
		WithContext(ctx).
		Exec()

	return err
}
