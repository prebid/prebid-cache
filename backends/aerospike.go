package backends

import (
	"context"
	"errors"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	as "github.com/aerospike/aerospike-client-go"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/stats"
)

const setName = "uuid"
const binValue = "value"

type Aerospike struct {
	cfg    config.Aerospike
	client *as.Client
}

func NewAerospikeBackend(cfg config.Aerospike) *Aerospike {
	client, err := as.NewClient(cfg.Host, cfg.Port)
	if err != nil {
		stats.LogAerospikeErrorStats()
		logger.Fatal("Error creating Aerospike backend: %v", err)
		panic("Aerospike failure. This shouldn't happen.")
	}
	logger.Info("Connected to Aerospike at %s:%d", cfg.Host, cfg.Port)

	return &Aerospike{
		cfg:    cfg,
		client: client,
	}
}

func (a *Aerospike) Get(ctx context.Context, key string) (string, error) {
	aerospikeStartTime := time.Now().Nanosecond()
	asKey, err := as.NewKey(a.cfg.Namespace, setName, key)
	if err != nil {
		return "", err
	}
	rec, err := a.client.Get(nil, asKey, "value")
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", errors.New("client.Get returned a nil record. Is aerospike configured properly?")
	}
	aerospikeEndTime := time.Now().Nanosecond()
	aerospikeDiffTime := (aerospikeEndTime.Sub(aerospikeStartTime)) / 1000000
	logger.Info("Time taken by Aerospike for get: %v", aerospikeDiffTime)
	return rec.Bins[binValue].(string), nil
}

func (a *Aerospike) Put(ctx context.Context, key string, value string) error {
	aerospikeStartTime := time.Now().Nanosecond()
	asKey, err := as.NewKey(a.cfg.Namespace, setName, key)
	if err != nil {
		return err
	}
	bins := as.BinMap{
		binValue: value,
	}
	err = a.client.Put(nil, asKey, bins)
	if err != nil {
		return err
	}
	aerospikeEndTime := time.Now().Nanosecond()
	aerospikeDiffTime := (aerospikeEndTime.Sub(aerospikeStartTime)) / 1000000
	logger.Info("Time taken by Aerospike for put: %v", aerospikeDiffTime)
	return nil
}
