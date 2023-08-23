package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

// IgniteDB is an interface that helps us communicate with an Apache Ignite storage service
type IgniteDB interface {
	Get(req *http.Request) (string, bool, error)
	Put(ctx context.Context, req *http.Request) (bool, error)
}

// IgniteClient implements IgniteDB interface and communicates with the Apache Ignite storage
// via its REST API as documented in https://ignite.apache.org/docs/2.11.1/restapi#rest-api-reference
type IgniteClient struct {
	client *http.Client
}

type getResponse struct {
	Error    string `json:"error"`
	Response string `json:"response"`
	Status   int    `json:"successStatus"`
}

// Get implements the IgniteDB interface Get method and makes the Ignite storage client retrieve
// the value that has been previously stored under 'key' if its TTL is still current. We can tell
// when a key is not faound when Ignite doesn't return an error, nor a 'Status' different than zero, but
// the 'Response' field is empty. Get can also return Ignite server-side errors
func (ig *IgniteClient) Get(req *http.Request) (string, error) {
	// Send request to Ignite storage service
	httpResp, err := ig.client.Do(req)
	if err != nil {
		return "", err
	}
	if httpResp.Body == nil {
		return "", errors.New("Ignite error: weceived empty httpResp.Body")
	}
	defer httpResp.Body.Close()

	requestBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", err
	}

	// Unmarshall response
	igniteResponse := getResponse{}
	if err := json.Unmarshal(requestBody, &igniteResponse); err != nil {
		return "", err
	}

	// Validate response
	if len(igniteResponse.Error) > 0 {
		return "", utils.NewPBCError(utils.GET_INTERNAL_SERVER, igniteResponse.Error)
	} else if igniteResponse.Status > 0 {
		return "", utils.NewPBCError(utils.GET_INTERNAL_SERVER, "Ignite response.Status not zero")
	} else if len(igniteResponse.Response) == 0 { // both igniteResponse.Status == 0 && len(igniteResponse.Error) == 0
		return "", utils.NewPBCError(utils.KEY_NOT_FOUND)
	}

	return igniteResponse.Response, nil
}

type putResponse struct {
	Error    string `json:"error"`
	Response bool   `json:"response"`
	Status   int    `json:"successStatus"`
}

// Put implements the IgniteDB interface Put method and comunicates with the Ignite storage service
// Returns an Ignite storage internal server error, if any.
func (ig *IgniteClient) Put(req *http.Request) (bool, error) {
	// Send request to Ignite storage service
	httpResp, err := ig.client.Do(req)
	defer httpResp.Body.Close()
	if err != nil {
		return false, err
	}
	requestBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return false, err
	}

	// Unmarshall response
	igniteResponse := putResponse{}
	if err := json.Unmarshal(requestBody, &igniteResponse); err != nil {
		return false, err
	}

	// Validate response
	if igniteResponse.Status > 0 || len(igniteResponse.Error) > 0 {
		return false, utils.NewPBCError(utils.PUT_INTERNAL_SERVER, igniteResponse.Error)
	}

	return igniteResponse.Response, nil
}

// IgniteBackend implements the Backend interface
type IgniteBackend struct {
	client    *IgniteClient
	serverURL *url.URL
}

// NewIgniteBackend expects a valid config.IgniteBackend object
func NewIgniteBackend(cfg config.Ignite) *IgniteBackend {

	url, err := url.Parse(fmt.Sprintf("%s://%s:%d/ignite?cacheName=%s", cfg.Scheme, cfg.Host, cfg.Port, cfg.CacheName))
	if err != nil {
		log.Fatalf("Error creating Ignite backend: %s", err.Error())
		panic("IgniteBackend failure. This shouldn't happen.")
	}

	return &IgniteBackend{
		client: &IgniteClient{
			client: http.DefaultClient,
		},
		serverURL: url,
	}
}

// Get creates an Get http.Request with the URL query values needed to perform a "get" operation
// on an Ignite storage instance using the REST API
func (back *IgniteBackend) Get(ctx context.Context, key string) (string, error) {

	urlCopy := *back.serverURL

	q := urlCopy.Query()
	q.Set("cmd", "get")
	q.Set("key", key)

	urlCopy.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlCopy.String(), nil)
	if err != nil {
		return "", err
	}

	return back.client.Get(httpReq)
}

// Put creates an Get http.Request with the URL query values needed to perform a "putifabs" command
// in order to store `value` only if `key` doesn't exist in the storage already. Returns
// RecordExistsError or whatever PUT_INTERNAL_SERVER error we might find on the storage side
func (back *IgniteBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	urlCopy := *(&back.serverURL)
	q := urlCopy.Query()
	q.Set("cmd", "putifabs")
	q.Set("key", key)
	q.Set("val", value)
	q.Set("exp", fmt.Sprintf("%d", ttlSeconds*1000))

	urlCopy.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlCopy.String(), nil)
	if err != nil {
		return err
	}

	applied, err := back.client.Put(httpReq)
	if !applied {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return err
}
