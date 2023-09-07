package backends

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	Put(req *http.Request) (bool, error)
	CreateCache() error
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
	httpResp, httpErr := ig.client.Do(req)
	if httpErr != nil {
		return "", httpErr
	}

	if httpResp.StatusCode != http.StatusOK {
		httpErr = fmt.Errorf("Ignite error. Unexpected status code: %d", httpResp.StatusCode)
	}

	if httpResp.Body == nil {
		errMsg := "Received empty httpResp.Body"
		if httpErr == nil {
			return "", fmt.Errorf("Ignite error. %s", errMsg)
		}
		return "", fmt.Errorf("%s; %s", httpErr.Error(), errMsg)
	}
	defer httpResp.Body.Close()

	responseBody, ioErr := io.ReadAll(httpResp.Body)
	if ioErr != nil {
		errMsg := fmt.Sprintf("IO reader error: %s", ioErr)
		if httpErr == nil {
			return "", fmt.Errorf("Ignite error. %s", errMsg)
		}
		return "", fmt.Errorf("%s; %s", httpErr.Error(), errMsg)
	}

	// Unmarshall response
	igniteResponse := getResponse{}
	if unmarshalErr := json.Unmarshal(responseBody, &igniteResponse); unmarshalErr != nil {
		errMsg := fmt.Sprintf("Unmarshal response error: %s; Response body: %s", unmarshalErr.Error(), string(responseBody))
		if httpErr == nil {
			return "", fmt.Errorf("Ignite error. %s", errMsg)
		}
		return "", fmt.Errorf("%s; %s", httpErr.Error(), errMsg)
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

// createCache uses the Apache Ignite REST API "getorcreate" command to create a cache
func createCache(igniteHost, cacheName string) error {
	url, err := url.Parse(fmt.Sprintf("%s?cmd=getorcreate&cacheName=%s", igniteHost, cacheName))
	if err != nil {
		return err
	}

	httpResp, err := http.Get(url.String())
	if err != nil {
		return err
	}

	if httpResp.Body == nil {
		return fmt.Errorf("Received empty httpResp.Body when trying to create cache %s", cacheName)
	}
	defer httpResp.Body.Close()

	var statusCodeErrMsg string
	if httpResp.StatusCode != http.StatusOK {
		statusCodeErrMsg = fmt.Sprintf("http status %d when creating cache %s", httpResp.StatusCode, cacheName)
	}

	requestBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		if len(statusCodeErrMsg) > 0 {
			return fmt.Errorf("%s %s", statusCodeErrMsg, err.Error())
		}
		return fmt.Errorf("%s", err.Error())
	}

	igniteResponse := getResponse{}
	if err := json.Unmarshal(requestBody, &igniteResponse); err != nil {
		if len(statusCodeErrMsg) > 0 {
			return fmt.Errorf("%s %s", statusCodeErrMsg, err.Error())
		}
		return fmt.Errorf("%s", err.Error())
	}
	if igniteResponse.Status > 0 {
		if len(igniteResponse.Error) > 0 {
			return fmt.Errorf("%s", igniteResponse.Error)
		}
		return fmt.Errorf("successStatus does not equal 0 %v", igniteResponse)
	}

	return nil
}

// IgniteBackend implements the Backend interface
type IgniteBackend struct {
	client    *IgniteClient
	serverURL *url.URL
	cacheName string
}

// NewIgniteBackend expects a valid config.IgniteBackend object
func NewIgniteBackend(cfg config.Ignite) *IgniteBackend {

	completeHost := fmt.Sprintf("%s://%s:%d/ignite", cfg.Scheme, cfg.Host, cfg.Port)

	if cfg.Cache.CreateOnStart {
		if err := createCache(completeHost, cfg.Cache.Name); err != nil {
			log.Fatalf("Error creating Ignite backend: %s", err.Error())
		}
	}
	log.Infof("Prebid Cache will write to Ignite cache name: %s", cfg.Cache.Name)

	url, err := url.Parse(fmt.Sprintf("%s://%s:%d/ignite?cacheName=%s", cfg.Scheme, cfg.Host, cfg.Port, cfg.Cache.Name))
	if err != nil {
		errMsg := fmt.Sprintf("Error creating Ignite backend: error parsing Ignite host URL %s", err.Error())
		log.Fatalf(errMsg)
		panic(errMsg)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &IgniteBackend{
		client: &IgniteClient{
			client: &http.Client{Transport: tr},
		},
		serverURL: url,
		cacheName: "myCache",
	}
}

// Get creates an Get http.Request with the URL query values needed to perform a "get" operation
// on an Ignite storage instance using the REST API
func (back *IgniteBackend) Get(ctx context.Context, key string) (string, error) {

	urlCopy := *back.serverURL

	q := urlCopy.Query()
	q.Set("cmd", "get")
	q.Set("cacheName", back.cacheName)
	q.Set("key", key)

	urlCopy.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlCopy.String(), nil)
	if err != nil {
		return "", err
	}

	headers := http.Header{}
	headers.Add("Host", "prebid.adnxs.com")
	httpReq.Header = headers

	return back.client.Get(httpReq)
}

// Put creates an Get http.Request with the URL query values needed to perform a "putifabs" command
// in order to store `value` only if `key` doesn't exist in the storage already. Returns
// RecordExistsError or whatever PUT_INTERNAL_SERVER error we might find on the storage side
func (back *IgniteBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	urlCopy := *(&back.serverURL)
	q := urlCopy.Query()
	q.Set("cmd", "putifabs")
	q.Set("cacheName", back.cacheName)
	q.Set("key", key)
	q.Set("val", value)
	q.Set("exp", fmt.Sprintf("%d", ttlSeconds*1000))

	urlCopy.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlCopy.String(), nil)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Add("Host", "prebid.adnxs.com")
	httpReq.Header = headers

	applied, err := back.client.Put(httpReq)
	if !applied {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return err
}
