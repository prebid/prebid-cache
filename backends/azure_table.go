package backends

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"context"
	"sync"

	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const METHOD_GET string = "GET"
const METHOD_POST string = "POST"

type AzureValue struct {
	ID           string `json:"id"`
	Value        string `json:"value"`
	PartitionKey string `json:"uuid"`
}

type AzureTableBackend struct {
	Client *fasthttp.Client
	Key    string
	uri    string

	partitionKeyPool sync.Pool // Stores [8]byte instances where the first chars are [" and the last are "]
}

type AzureErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewAzureBackend(account string, key string) *AzureTableBackend {

	log.Debugf("New Azure Backend: Account %s Key %s", account, key)

	c := &AzureTableBackend{
		Key: key,
		// Consider an interface so we can mock. Take a look at how other clients get mocked.
		Client: &fasthttp.Client{
			MaxIdleConnDuration: 30 * time.Second,
			DialDualStack:       true,
			WriteTimeout:        15 * time.Second,
			ReadTimeout:         15 * time.Second,
		},
		// Probably fmt.Sprintf("https://%s.documents.azure.com/dbs/%s/colls/%s/docs", account, dbID, collectionID),
		uri: fmt.Sprintf("https://%s.documents.azure.com/%s", account, "dbs/prebidcache/colls/cache/docs"),

		partitionKeyPool: sync.Pool{
			New: func() interface{} {
				buffer := [8]byte{}
				buffer[0] = '['
				buffer[1] = '"'
				buffer[6] = '"'
				buffer[7] = ']'
				return &buffer
			},
		},
	}

	log.Infof("New Azure Client: %s", account)

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

func (c *AzureTableBackend) Send(ctx context.Context, req *fasthttp.Request, resp *fasthttp.Response, resourceType string, resourceId string) error {
	date := formattedRequestTime()
	req.Header.Add("x-ms-date", date)
	req.Header.Add("x-ms-version", "2018-12-31")
	req.Header.Add("Authorization", c.signReq(string(req.Header.Method()), resourceType, resourceId, date))

	deadline, ok := ctx.Deadline()
	var err error = nil
	if ok {
		err = c.Client.DoDeadline(req, resp, deadline)
	} else {
		err = c.Client.Do(req, resp)
	}

	return err
}

// Current function working
//func (c *AzureTableBackend) Get(ctx context.Context, key string) (string, error) {
//	/* validate get args */
//	if key == "" {
//		return "", fmt.Errorf("Invalid Key")
//	}
//
//	/* Create fasthttp request and response */
//	req := fasthttp.AcquireRequest()
//	defer fasthttp.ReleaseRequest(req)
//	var resp = fasthttp.AcquireResponse()
//	defer fasthttp.ReleaseResponse(resp)
//	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
//
//	/* Request headers */
//	req.Header.SetMethod(METHOD_GET)
//	req.SetRequestURI(fmt.Sprintf("%s/%s", c.uri, key))
//	req.SetBodyString("")
//	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(c.makePartitionKey(key)))
//
//	////err := c.Send(ctx, req, resp, "docs", resourceLink[1:])
//	//err := c.Send(ctx, req, resp, "docs", fmt.Sprintf("dbs/prebidcache/colls/cache/docs/%s", key))
//	req.Header.Add("x-ms-date", date)
//	req.Header.Add("x-ms-version", "2018-12-31")
//	//req.Header.Add("Authorization", c.signReq(string(req.Header.Method()), resourceType, resourceId, date))
//	signatureString := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
//		strings.ToLower("GET"),
//		"docs",
//		fmt.Sprintf("dbs/prebidcache/colls/cache/docs/%s", key),
//		strings.ToLower(date),
//	)
//
//	decodedKey, _ := base64.StdEncoding.DecodeString(c.Key)
//	sha256 := hmac.New(sha256.New, []byte(decodedKey))
//	sha256.Write([]byte(signatureString))
//
//	encodedSignature := base64.StdEncoding.EncodeToString(sha256.Sum(nil))
//	u := url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", encodedSignature))
//	req.Header.Add("Authorization", u)
//	log.Infof("%v \n", req)
//
//	/* Do request */
//	deadline, ok := ctx.Deadline()
//	var err error = nil
//	if ok {
//		err = c.Client.DoDeadline(req, resp, deadline)
//	} else {
//		err = c.Client.Do(req, resp)
//	}
//
//	if err != nil {
//		log.Debugf("Failed to make request")
//		return "", err
//	}
//
//	/* Build prebid-cache response */
//	av := AzureValue{}
//	err = json.Unmarshal(resp.Body(), &av)
//	if err != nil {
//		log.Debugf("Failed to decode request body into JSON")
//		return "", err
//	}
//
//	if av.Value == "" {
//		log.Debugf("Response had empty value: %v", av)
//		return "", utils.KeyNotFoundError{}
//	}
//
//	return av.Value, nil
//}

func (c *AzureTableBackend) Get(ctx context.Context, key string) (string, error) {
	// validate get args
	if err := validateGetArgs(key); err != nil {
		return "", err
	}

	// Make put request
	req := c.buildGetRequest(key)
	defer fasthttp.ReleaseRequest(req)
	var resp = fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Send request
	if err := c.sendRequest(ctx, req, resp); err != nil {
		return "", err
	}

	// Interpret response as success or error
	value, azureError := interpretAzureGetResponse(resp)
	if azureError != nil {
		return "", azureError
	}

	return value, nil
}

func validateGetArgs(key string) error {
	if key == "" {
		return fmt.Errorf("Invalid Key")
	}
	return nil
}

func (c *AzureTableBackend) buildGetRequest(key string) *fasthttp.Request {
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	req := fasthttp.AcquireRequest()

	// Set headers
	req.Header.SetMethod(METHOD_GET)
	req.SetRequestURI(fmt.Sprintf("%s/%s", c.uri, key))
	req.SetBodyString("")
	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(c.makePartitionKey(key)))
	req.Header.Add("x-ms-date", date)
	req.Header.Add("x-ms-version", "2018-12-31")
	req.Header.Add("Authorization", createEncodedSignature(c.Key, date, METHOD_GET, key))

	return req
}

//func (c *AzureTableBackend) Get(ctx context.Context, key string) (string, error) {
//
//	if key == "" {
//		return "", fmt.Errorf("Invalid Key")
//	}
//
//	// Full key for the stupid gets
//	//resourceLink := fmt.Sprintf("/dbs/prebidcache/colls/cache/docs/%s", key)
//	req := fasthttp.AcquireRequest()
//	defer fasthttp.ReleaseRequest(req)
//	var resp = fasthttp.AcquireResponse()
//	defer fasthttp.ReleaseResponse(resp)
//
//	req.Header.SetMethod("GET")
//	//req.SetRequestURI(fmt.Sprintf("%s%s", c.URI, resourceLink))
//	req.SetRequestURI(fmt.Sprintf("%s/%s", c.uri, key))
//	req.SetBodyString("")
//
//	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(c.makePartitionKey(key)))
//	//err := c.Send(ctx, req, resp, "docs", resourceLink[1:])
//	err := c.Send(ctx, req, resp, "docs", fmt.Sprintf("dbs/prebidcache/colls/cache/docs/%s", key))
//	if err != nil {
//		log.Debugf("Failed to make request")
//		return "", err
//	}
//
//	av := AzureValue{}
//	err = json.Unmarshal(resp.Body(), &av)
//	if err != nil {
//		log.Debugf("Failed to decode request body into JSON")
//		return "", err
//	}
//
//	if av.Value == "" {
//		log.Debugf("Response had empty value: %v", av)
//		return "", fmt.Errorf("Key not found")
//	}
//
//	return av.Value, nil
//}

//func (c *AzureTableBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
//
//	if key == "" {
//		return fmt.Errorf("Invalid Key")
//	}
//
//	if value == "" {
//		return fmt.Errorf("Invalid Value")
//	}
//	partitionKey := c.makePartitionKey(key)
//	log.Debugf("POST partition key %s", partitionKey)
//	av := AzureValue{
//		ID:           key,
//		Value:        value,
//		PartitionKey: partitionKey,
//	}
//
//	b, err := json.Marshal(&av)
//	if err != nil {
//		return err
//	}
//
//	//resourceLink := "/dbs/prebidcache/colls/cache/docs"
//
//	req := fasthttp.AcquireRequest()
//	defer fasthttp.ReleaseRequest(req)
//	var resp = fasthttp.AcquireResponse()
//	defer fasthttp.ReleaseResponse(resp)
//
//	req.Header.SetMethod("POST")
//	req.SetRequestURI(c.uri)
//	req.SetBody(b)
//
//	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(partitionKey))
//	req.Header.Add("x-ms-documentdb-is-upsert", "false")
//	if err != nil {
//		return err
//	}
//	err = c.Send(ctx, req, resp, "docs", "dbs/prebidcache/colls/cache")
//	return err
//}

func (c *AzureTableBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	// validate put args
	if err := validatePutArgs(key, value); err != nil {
		return err
	}

	// Build put request to be sent
	req, err := c.buildPutRequest(key, value)
	defer fasthttp.ReleaseRequest(req)
	if err != nil {
		return err
	}

	// Send request
	var resp = fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := c.sendRequest(ctx, req, resp); err != nil {
		return err
	}

	// Interpret response as success or error
	if azureError := interpretAzurePutResponse(resp); azureError != nil {
		return azureError
	}

	return nil
}

// interpretAzureGetResponse checks the response object to verify whether the
// GET request was successful or not
func interpretAzureGetResponse(resp *fasthttp.Response) (string, error) {
	var rv string = ""
	var re error = nil

	if resp == nil {
		return "", errors.New(http.StatusText(http.StatusInternalServerError))
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		av := AzureValue{}
		if err := json.Unmarshal(resp.Body(), &av); err != nil {
			return "", errors.New("Failed to decode request body into JSON")
		}
		rv = av.Value
	case http.StatusNotFound:
		re = utils.KeyNotFoundError{}
	default:
		re = errors.New(http.StatusText(resp.StatusCode()))
	}

	return rv, re
}

func unmarshallAzureErrorResponse(respBody []byte) (*AzureErrorDesc, error) {
	if len(respBody) == 0 {
		return nil, errors.New("Azure CosmoDB Response: Empty response body")
	}

	// Unmarshall body
	ae := AzureErrorResponse{}
	var jsonMsg string
	if err := json.Unmarshal(respBody, &ae); err != nil {
		return nil, errors.New("Azure CosmoDB Response: Could not unmarshal")
	}

	// Check the error message field is populated
	if len(ae.Message) == 0 {
		return nil, errors.New("Azure CosmoDB Response: Message field is empty")
	}

	// Check if ae.Message contains an "Errors" field
	v := strings.Split(ae.Message, "\n")
	if len(v[0]) == 0 || !strings.Contains(ae.Message, "Errors") {
		return nil, errors.New("Azure CosmoDB Response: Couldn't find 'Errors' field")
	}

	// Strip the JSON object the error message field
	jsonObjStart := strings.IndexByte(v[0], '{')
	jsonObjEnd := strings.IndexByte(v[0], '}')
	if jsonObjStart == -1 || jsonObjEnd == -1 {
		return nil, errors.New("Azure CosmoDB Response: Couldn't find JSON object inside the message response")
	}
	jsonMsg = v[0][jsonObjStart : jsonObjEnd+1]

	errMsg := AzureErrorDesc{}
	if err := json.Unmarshal(json.RawMessage(jsonMsg), &errMsg); err != nil {
		return nil, errors.New("Azure CosmoDB Response: Could not unmarshal message field value")
	}

	if len(errMsg.Errors) == 0 {
		return nil, errors.New("Azure CosmoDB Response: Empty 'Errors' field inside the 'message' value")
	}

	return &errMsg, nil
}

type AzureErrorDesc struct {
	Errors []string `json:"Errors"`
}

// interpretAzurePutResponse checks the response object to verify whether the
// POST request was successful or not
func interpretAzurePutResponse(resp *fasthttp.Response) error {

	if resp == nil {
		return errors.New(http.StatusText(http.StatusInternalServerError))
	}

	if resp.StatusCode() != http.StatusCreated {

		if resp.StatusCode() == http.StatusConflict {
			return utils.RecordExistsError{}
		}
		ae, err := unmarshallAzureErrorResponse(resp.Body())
		if err != nil {
			return err
		}
		return errors.New(string(ae.Errors[0]))
	}

	// Successfully posted element in Azure Cosmo DB Document storage
	return nil
}

func (c *AzureTableBackend) makePartitionKey(objectKey string) string {
	end := len(objectKey)
	if end > 4 {
		end = 4
	}
	return objectKey[0:end]
}

func (c *AzureTableBackend) wrapForHeader(partitionKey string) string {
	buffer := c.partitionKeyPool.Get().(*[8]byte)
	defer c.partitionKeyPool.Put(buffer)
	copy((*buffer)[2:6], partitionKey)
	return string((*buffer)[:])
}

func validatePutArgs(key string, value string) error {
	if key == "" {
		return errors.New("Invalid Key")
	}

	if value == "" {
		return errors.New("Invalid Value")
	}

	return nil
}

func (c *AzureTableBackend) buildPutRequest(key string, value string) (*fasthttp.Request, error) {
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	partitionKey := c.makePartitionKey(key)
	log.Debugf("POST partition key %s", partitionKey)

	// create element to put
	b, err := newPutValue(key, value, partitionKey)
	if err != nil {
		return nil, err
	}

	// Allocate request
	req := fasthttp.AcquireRequest()

	// write request headers
	req.SetRequestURI(c.uri)
	req.SetBody(b)
	req.Header.SetMethod(METHOD_POST)
	req.Header.Add("x-ms-documentdb-partitionkey", c.wrapForHeader(partitionKey))
	req.Header.Add("x-ms-documentdb-is-upsert", "false")
	req.Header.Add("x-ms-date", date)
	req.Header.Add("x-ms-version", "2018-12-31")
	req.Header.Add("Authorization", createEncodedSignature(c.Key, date, METHOD_POST, key))

	return req, nil
}

func newPutValue(key string, value string, partitionKey string) ([]byte, error) {
	av := AzureValue{
		ID:           key,
		Value:        value,
		PartitionKey: partitionKey,
	}

	return json.Marshal(&av)
}

func createEncodedSignature(azureAuthorizationKey, date, requestMethod, elemKey string) string {
	var resourceLink string

	switch requestMethod {
	case METHOD_GET:
		resourceLink = fmt.Sprintf("dbs/prebidcache/colls/cache/docs/%s", elemKey)
	case METHOD_POST:
		resourceLink = "dbs/prebidcache/colls/cache"
	}

	signatureString := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(requestMethod),
		"docs",
		resourceLink,
		strings.ToLower(date),
	)

	decodedKey, _ := base64.StdEncoding.DecodeString(azureAuthorizationKey)
	sha256 := hmac.New(sha256.New, []byte(decodedKey))
	sha256.Write([]byte(signatureString))

	encodedSignature := base64.StdEncoding.EncodeToString(sha256.Sum(nil))

	return url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", encodedSignature))
}

func (c *AzureTableBackend) sendRequest(ctx context.Context, req *fasthttp.Request, resp *fasthttp.Response) error {
	if deadline, ok := ctx.Deadline(); ok {
		return c.Client.DoDeadline(req, resp, deadline)
	}
	return c.Client.Do(req, resp)
}
