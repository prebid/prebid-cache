package backends

import (
	"context"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/gocql/gocql"
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
		logger.Fatal("Error creating Cassandra backend: %v", err)
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

func (c *Cassandra) Put(ctx context.Context, key string, value string) error {
	err := c.session.Query(`INSERT INTO cache (key, value) VALUES (?, ?) USING TTL 2400`, key, value).
		WithContext(ctx).
		Exec()

	return err
}
