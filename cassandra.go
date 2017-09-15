package main

import (
	"context"
	"github.com/gocql/gocql"
)

// CassandraConfig is used to configure the cluster
type CassandraConfig struct {
	hosts    string
	keyspace string
}

// Cassandra Object use to implement backend interface
type Cassandra struct {
	config  *CassandraConfig
	cluster *gocql.ClusterConfig
	session *gocql.Session
}

// NewCassandraBackend create a new cassandra backend
func NewCassandraBackend(config *CassandraConfig) (*Cassandra, error) {
	var err error

	c := &Cassandra{}
	c.config = config
	c.cluster = gocql.NewCluster(c.config.hosts)
	c.cluster.Keyspace = c.config.keyspace
	c.cluster.Consistency = gocql.LocalOne

	c.session, err = c.cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return c, nil
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
