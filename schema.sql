CREATE KEYSPACE prebid WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'};

CREATE TABLE prebid.cache (
    key text,
    value text,
    PRIMARY KEY (key)
);
