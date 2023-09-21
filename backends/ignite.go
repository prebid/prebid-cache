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

// requestDoer is an interface that sends the request to the server
type requestDoer interface {
	//DoRequest(*url.URL)

	CreateCache(url *url.URL, cacheName string) error
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

type putResponse struct {
	Error    string `json:"error"`
	Response bool   `json:"response"`
	Status   int    `json:"successStatus"`
}

// CreateCache uses the Apache Ignite REST API "getorcreate" command to create a cache
func (ig *IgniteClient) CreateCache(url *url.URL, cacheName string) error {
	urlCopy := *url
	q := urlCopy.Query()
	q.Set("cmd", "getorcreate")
	q.Set("cacheName", cacheName)
	urlCopy.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(context.Background(), "GET", urlCopy.String(), nil)
	if err != nil {
		return err
	}

	httpResp, err := ig.client.Do(httpReq)
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

	if len(cfg.Scheme) == 0 || len(cfg.Host) == 0 || cfg.Port == 0 || len(cfg.Cache.Name) == 0 {
		errMsg := "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name"
		log.Fatalf(errMsg)
		panic(errMsg)
	}
	completeHost := fmt.Sprintf("%s://%s:%d/ignite", cfg.Scheme, cfg.Host, cfg.Port)

	url, err := url.Parse(fmt.Sprintf("%s?cacheName=%s", completeHost, cfg.Cache.Name))
	if err != nil {
		errMsg := fmt.Sprintf("Error creating Ignite backend: error parsing Ignite host URL %s", err.Error())
		log.Fatalf(errMsg)
		panic(errMsg)
	}

	igb := &IgniteBackend{serverURL: url}
	if cfg.Secure {
		igb.client = &IgniteClient{
			client: http.DefaultClient,
		}
	} else {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		igb.client = &IgniteClient{
			client: &http.Client{Transport: tr},
		}
	}

	if cfg.Cache.CreateOnStart {
		if err := igb.client.CreateCache(url, cfg.Cache.Name); err != nil {
			errMsg := fmt.Sprintf("Error creating Ignite backend: %s", err.Error())
			log.Fatalf(errMsg)
			panic(errMsg)
		}
	}
	log.Infof("Prebid Cache will write to Ignite cache name: %s", cfg.Cache.Name)

	igb.cacheName = cfg.Cache.Name

	return igb
}

// Get implements the Backend interface. Makes the Ignite storage client retrieve the value that has
// been previously stored under 'key' if its TTL is still current. We can tell when a key is not found
// when Ignite doesn't return an error, nor a 'Status' different than zero, but the 'Response' field is
// empty. Get can also return Ignite server-side errors
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

	//return back.client.Get(httpReq)
	httpResp, httpErr := back.client.client.Do(httpReq)
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

// Put implements the Backend interface to comunicates with the Ignite storage service to perform
// a "putifabs" command in order to store the "value" parameter only if the "key" doesn't exist in
// the storage already. Returns RecordExistsError or whatever PUT_INTERNAL_SERVER error we might
// find in the storage side
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

	//applied, err := back.client.Put(httpReq)
	//<---->
	//func (ig *IgniteClient) Put(req *http.Request) (bool, error) {
	// Send request to Ignite storage service
	httpResp, httpErr := back.client.client.Do(httpReq)
	if httpErr != nil {
		return httpErr
	}

	if httpResp.StatusCode != http.StatusOK {
		httpErr = fmt.Errorf("Ignite error. Unexpected status code: %d", httpResp.StatusCode)
	}

	if httpResp.Body == nil {
		errMsg := "Received empty httpResp.Body"
		if httpErr == nil {
			return fmt.Errorf("Ignite error. %s", errMsg)
		}
		return fmt.Errorf("%s; %s", httpErr.Error(), errMsg)
	}
	defer httpResp.Body.Close()

	responseBody, ioErr := io.ReadAll(httpResp.Body)
	if ioErr != nil {
		errMsg := fmt.Sprintf("IO reader error: %s", ioErr)
		if httpErr == nil {
			return fmt.Errorf("Ignite error. %s", errMsg)
		}
		return fmt.Errorf("%s; %s", httpErr.Error(), errMsg)
	}

	// Unmarshall response
	igniteResponse := putResponse{}
	if err := json.Unmarshal(responseBody, &igniteResponse); err != nil {
		return err
	}

	// Validate response
	if igniteResponse.Status > 0 || len(igniteResponse.Error) > 0 {
		return utils.NewPBCError(utils.PUT_INTERNAL_SERVER, igniteResponse.Error)
	}

	if !igniteResponse.Response {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}

	return nil
}
