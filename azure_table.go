package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
)

type AzureValue struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type AzureTableBackend struct {
	Client  *http.Client
	Account string
	Key     string
	URI     string
}

func NewAzureBackend(account string, key string) *AzureTableBackend {

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
		Key:     key,
		Client: &http.Client{
			//TODO add to configMap
			Transport: tr,
		},
		URI: fmt.Sprintf("https://%s.documents.azure.com", account),
	}

	log.Info("New Azure Client", account)

	return c
}

func (c *AzureTableBackend) signReq(verb, resourceType, resourceLink, date string) string {

	strToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		resourceType,
		resourceLink,
		strings.ToLower(date),
	)

	decodedKey, _ := base64.StdEncoding.DecodeString(c.Key)
	sha256 := hmac.New(sha256.New, []byte(decodedKey))
	sha256.Write([]byte(strToSign))

	signature := base64.StdEncoding.EncodeToString(sha256.Sum(nil))
	u := url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", signature))

	return u
}

func formattedRequestTime() string {
	t := time.Now().UTC()
	return t.Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

func (c *AzureTableBackend) Send(ctx context.Context, req *http.Request, resourceType string, resourceId string) (*http.Response, error) {
	date := formattedRequestTime()
	req.Header.Add("x-ms-date", date)
	req.Header.Add("x-ms-version", "2017-01-19")
	req.Header.Add("Authorization", c.signReq(req.Method, resourceType, resourceId, date))

	ctx = httptrace.WithClientTrace(ctx, newHttpTracer())

	resp, err := ctxhttp.Do(ctx, c.Client, req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *AzureTableBackend) Do(ctx context.Context, method string, resourceLink string, resourceType string, resourceId string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.URI, resourceLink), body)
	if err != nil {
		return nil, err
	}
	return c.Send(ctx, req, resourceType, resourceId)
}

func (c *AzureTableBackend) Get(ctx context.Context, key string) (string, error) {

	if key == "" {
		return "", fmt.Errorf("Invalid Key")
	}

	// Full key for the stupid gets
	resourceLink := fmt.Sprintf("/dbs/prebidcache/colls/cache/docs/%s", key)
	resp, err := c.Do(ctx, "GET", resourceLink, "docs", resourceLink[1:], nil)
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

	av := AzureValue{
		ID:    key,
		Value: value,
	}

	b, err := json.Marshal(&av)
	if err != nil {
		return err
	}

	resourceLink := "/dbs/prebidcache/colls/cache/docs"
	resp, err := c.Do(ctx, "POST", resourceLink, "docs", "dbs/prebidcache/colls/cache", bytes.NewBuffer(b))
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
