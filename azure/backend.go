package azure

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"context"
	"crypto/tls"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context/ctxhttp"
	"net/http/httptrace"
	"sync"
)

type AzureValue struct {
	ID           string `json:"id"`
	Value        string `json:"value"`
	PartitionKey string `json:"partition"`
}

type AzureTableBackend struct {
	Client  *http.Client
	Account string
	URI     string

	auth             *authorization
	partitionKeyPool sync.Pool // Stores [8]byte instances where the first chars are [" and the last are "]
}

func NewBackend(account string, key string) *AzureTableBackend {

	log.Debugf("New Azure Backend: Account %s Key %s", account, key)
	tr := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     60 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
	}

	c := &AzureTableBackend{
		Account: account,
		Client: &http.Client{
			//TODO add to configMap
			Transport: tr,
		},
		URI: fmt.Sprintf("https://%s.documents.azure.com", account),

		auth:     newAuthorization(key),
		partitionKeyPool: sync.Pool{
			New: func() interface{} {
				buffer := [8]byte{}
				buffer[0] = '['
				buffer[1] = '"'
				buffer[6] = '"'
				buffer[7] = ']'
				return buffer
			},
		},
	}

	log.Infof("New Azure Client: %s", account)

	return c
}

func formattedRequestTime() string {
	t := time.Now().UTC()
	return t.Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

func (c *AzureTableBackend) Send(ctx context.Context, req *http.Request, resourceType string, resourceId string) (*http.Response, error) {
	date := formattedRequestTime()
	req.Header.Add("x-ms-date", date)
	req.Header.Add("x-ms-version", "2017-01-19")
	req.Header.Add("Authorization", c.auth.sign(req.Method, resourceType, resourceId, date))

	ctx = httptrace.WithClientTrace(ctx, newHttpTracer())

	resp, err := ctxhttp.Do(ctx, c.Client, req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *AzureTableBackend) Get(ctx context.Context, key string) (string, error) {

	if key == "" {
		return "", fmt.Errorf("Invalid Key")
	}

	// Full key for the stupid gets
	resourceLink := fmt.Sprintf("/dbs/prebidcache/colls/cache/docs/%s", key)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.URI, resourceLink), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(c.makePartitionKey(key)))
	resp, err := c.Send(ctx, req, "docs", resourceLink[1:])
	if err != nil {
		log.Debugf("Failed to make request")
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("Failed to read the request body")
		return "", err
	}

	av := AzureValue{}
	err = json.Unmarshal(body, &av)
	if err != nil {
		log.Debugf("Failed to decode request body into JSON")
		return "", err
	}

	if av.Value == "" {
		log.Debugf("Response had empty value: %v", av)
		return "", fmt.Errorf("Key not found")
	}

	return av.Value, nil
}

func (c *AzureTableBackend) Put(ctx context.Context, key string, value string) error {

	if key == "" {
		return fmt.Errorf("Invalid Key")
	}

	if value == "" {
		return fmt.Errorf("Invalid Value")
	}
	partitionKey := c.makePartitionKey(key)
	log.Debugf("POST partition key %s", partitionKey)
	av := AzureValue{
		ID:           key,
		Value:        value,
		PartitionKey: partitionKey,
	}

	b, err := json.Marshal(&av)
	if err != nil {
		return err
	}

	resourceLink := "/dbs/prebidcache/colls/cache/docs"

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.URI, resourceLink), bytes.NewBuffer(b))
	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(partitionKey))
	if err != nil {
		return err
	}
	resp, err := c.Send(ctx, req, "docs", "dbs/prebidcache/colls/cache")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the whole body so that the Transport knows it's safe to reuse the connection.
	// See the docs on http.Response.Body
	ioutil.ReadAll(resp.Body)
	return nil
}

func newHttpTracer() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		PutIdleConn: func(err error) {
			if err != nil {
				log.Infof("Failed adding idle connection to the pool: %v", err.Error())
			}
		},

		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				log.Warnf("Failed to connect. Network: %s, Addr: %s, err: %v", network, addr, err)
			}
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			if info.Err != nil {
				log.Warnf("Failed DNS lookup: %v", info.Err)
			}
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err != nil {
				log.Warnf("Failed TLS Handshake: %v", err)
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err != nil {
				log.Warnf("Failed to write request: %v", info.Err)
			}
		},
	}
}

func (c *AzureTableBackend) makePartitionKey(objectKey string) string {
	return objectKey[0:4]
}

func (c *AzureTableBackend) wrapForHeader(partitionKey string) string {
	buffer := c.partitionKeyPool.Get().([8]byte)
	defer c.partitionKeyPool.Put(buffer)
	copy(buffer[2:6], partitionKey)
	return string(buffer[:])
}
