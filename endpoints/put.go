package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
)

// PutHandler serves "POST /cache" requests.
type PutHandler struct {
	backend backends.Backend
	cfg     putHandlerConfig
	memory  syncPools
	metrics *metrics.Metrics
}

type putHandlerConfig struct {
	maxNumValues int
	allowKeys    bool
}

type syncPools struct {
	requestPool     sync.Pool
	putResponsePool sync.Pool
}

// NewPutHandler returns the handle function for the "/cache" endpoint when it receives a POST request
func NewPutHandler(storage backends.Backend, metrics *metrics.Metrics, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	putHandler := &PutHandler{}

	// Assign storage client to put endpoint
	putHandler.backend = storage

	// pass metrics engine
	putHandler.metrics = metrics

	// Pass configuration values
	putHandler.cfg = putHandlerConfig{
		maxNumValues: maxNumValues,
		allowKeys:    allowKeys,
	}

	// Instantiate thread-safe memory pools
	putHandler.memory = syncPools{
		requestPool: sync.Pool{
			New: func() interface{} {
				return &putRequest{}
			},
		},
		putResponsePool: sync.Pool{
			New: func() interface{} {
				return &PutResponse{}
			},
		},
	}

	return putHandler.handle
}

// parseRequest unmarshals the incoming put request into a thread-safe memory pool. If
// the incoming request could not be unmarshalled or if the request comes with more
// elements to put than the maximum allowed in Prebid Cache's configuration, the
// corresponding error is returned
func (e *PutHandler) parseRequest(r *http.Request) (*putRequest, error) {
	if r == nil {
		return nil, utils.NewPBCError(utils.PUT_BAD_REQUEST)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, utils.NewPBCError(utils.PUT_BAD_REQUEST)
	}
	defer r.Body.Close()

	// Allocate a PutRequest object in thread-safe memory
	put := e.memory.requestPool.Get().(*putRequest)
	put.Puts = make([]putObject, 0)

	if err := json.Unmarshal(body, put); err != nil {
		// place memory back in sync pool
		e.memory.requestPool.Put(put)
		return nil, utils.NewPBCError(utils.PUT_BAD_REQUEST, string(body))
	}

	if len(put.Puts) > e.cfg.maxNumValues {
		// place memory back in sync pool
		e.memory.requestPool.Put(put)
		return nil, utils.NewPBCError(utils.PUT_MAX_NUM_VALUES, fmt.Sprintf("More keys than allowed: %d", e.cfg.maxNumValues))
	}

	return put, nil
}

// parsePutObject returns an error if the putObject comes with an invalid field
// and formats the string according to its type:
//   - XML content gets unmarshaled in order to un-escape it and then gets
//     prepended by its type
//   - JSON content gets prepended by its type
// No other formats are supported.
func parsePutObject(p putObject) (string, error) {
	var toCache string

	// Make sure there's data to store
	if len(p.Value) == 0 {
		return "", utils.NewPBCError(utils.MISSING_VALUE)
	}

	// Make sure a non-negative time-to-live quantity was provided
	if p.TTLSeconds < 0 {
		return "", utils.NewPBCError(utils.NEGATIVE_TTL, fmt.Sprintf("ttlseconds must not be negative %d.", p.TTLSeconds))
	}

	// Make sure data type is specified
	//if len(p.Type) == 0 {
	//	return "", utils.NewPBCError(utils.MISSING_TYPE)
	//}

	// Limit the type of data to XML or JSON
	if p.Type == backends.XML_PREFIX {
		if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
			return "", utils.NewPBCError(utils.MALFORMED_XML, fmt.Sprintf("XML messages must have a String value. Found %v", p.Value))
		}

		// Be careful about the cross-script escaping issues here. JSON requires quotation marks to be escaped,
		// for example... so we'll need to un-escape it before we consider it to be XML content.
		var interpreted string
		if err := json.Unmarshal(p.Value, &interpreted); err != nil {
			return "", utils.NewPBCError(utils.MALFORMED_XML, fmt.Sprintf("Error unmarshalling XML value: %v", p.Value))
		}

		toCache = p.Type + interpreted
	} else if p.Type == backends.JSON_PREFIX {
		toCache = p.Type + string(p.Value)
	} else {
		return "", utils.NewPBCError(utils.UNSUPPORTED_DATA_TO_STORE, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type))
	}

	return toCache, nil
}

func classifyBackendError(err error, index int) error {
	if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
		return utils.NewPBCError(utils.BAD_PAYLOAD_SIZE, fmt.Sprintf("POST /cache element %d exceeded max size: %v", index, err.Error()))
	}

	switch err {
	case context.DeadlineExceeded:
		return utils.NewPBCError(utils.PUT_DEADLINE_EXCEEDED)
	default:
		return utils.NewPBCError(utils.PUT_INTERNAL_SERVER, err.Error())
	}

	return nil
}

func logBackendError(err error) {
	logrus.Error("POST /cache Error while writing to the back-end: ", err)

	if pbcErr, isPBCErr := err.(utils.PBCError); isPBCErr && pbcErr.StatusCode == utils.PUT_DEADLINE_EXCEEDED {
		logrus.Error("POST /cache timed out:", err)
	} else {
		logrus.Error("POST /cache had an unexpected error:", err)
	}
}

// handle is the handler function that gets assigned to the POST method of the `/cache` endpoint
func (e *PutHandler) handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	e.metrics.RecordPutTotal()

	start := time.Now()

	bytes, err := e.processPutRequest(r)
	if err != nil {
		// At least one of the elements in the incoming request could not be stored
		// write the http error and log corresponding metrics
		var statusCode int
		if pbcErr, isPBCErr := err.(utils.PBCError); isPBCErr {
			statusCode = pbcErr.StatusCode
			if statusCode >= 400 && statusCode < 500 {
				e.metrics.RecordPutBadRequest()
			} else {
				e.metrics.RecordPutError()
			}
		} else {
			// All errors returned by e.processPutRequest(r) should be utils.PBCErrors
			// if not, consider it an interval server error with a http.StatusInternalServerError
			// status code and accounted under RecordPutError()
			statusCode = http.StatusInternalServerError
			e.metrics.RecordPutError()
		}

		http.Error(w, err.Error(), statusCode)
		return
	}

	// successfully stored all elements in storage service or database, write http
	// response and record duration metrics
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
	e.metrics.RecordPutDuration(time.Since(start))
}

// processPutRequest parses, unmarshals, and validates the incoming request; then calls the back-end Put()
// implementation on every element of the "puts" array. This function exits after all elements in the
// "puts" array have been stored in the back-end, or after the first error is found
func (e *PutHandler) processPutRequest(r *http.Request) ([]byte, error) {
	// Parse and validate incoming request
	putRequest, err := e.parseRequest(r)
	if err != nil {
		return nil, err
	}
	defer e.memory.requestPool.Put(putRequest)

	// Allocate a PutResponse object in thread-safe memory
	putResponse := e.memory.putResponsePool.Get().(*PutResponse)
	putResponse.Responses = make([]putResponseObject, len(putRequest.Puts))
	defer e.memory.putResponsePool.Put(putResponse)

	// Send elements to storage service or database
	if pcErr := e.putElements(putRequest, putResponse); pcErr != nil {
		return nil, pcErr
	}

	// Marshal Prebid Cache's response
	bytes, err := json.Marshal(putResponse)
	if err != nil {
		return nil, utils.NewPBCError(utils.MARSHAL_RESPONSE)
	}

	return bytes, nil
}

// putElements calls put(po *putObject, wg *sync.WaitGroup) in parallel and if any of those calls generates an error, logs the
// first one in the order its corresponding putObject came inside the []PutRequest.Puts array
//
// TODO: For those storage clients that support storing multiple elements in a single call, build a batch and send them together
// TODO: Allow Prebid Cache to provide error details in an "errors" field in the response
func (e *PutHandler) putElements(put *putRequest, resps *PutResponse) error {
	// Call Put() implementation of storage back-end in parrallel
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(put.Puts))

	for i := 0; i < len(put.Puts); i++ {
		go e.put(&put.Puts[i], &resps.Responses[i], i, &waitGroup)
	}
	waitGroup.Wait()

	// Log the first element found and return it
	for _, resp := range resps.Responses {
		if resp.err != nil {
			logBackendError(resp.err)
			return resp.err
		}
	}

	return nil
}

// put parses the putObject, validates it and calls the back-end storage Put() function this Prebid Cache instance
// is using. Returns a putResponseObject storing either the corresponding UUID's data was stored under, or an error
// if any.
func (e *PutHandler) put(po *putObject, resp *putResponseObject, index int, wg *sync.WaitGroup) {
	defer wg.Done()

	toCache, err := parsePutObject(*po)
	if err != nil {
		resp.err = err
		return
	}

	// Only allow setting a provided key if configured (and ensure a key is provided).
	if e.cfg.allowKeys && len(po.Key) > 0 {
		// put object comes with custom key, which we are allowed to use
		resp.UUID = po.Key
		e.metrics.RecordPutKeyProvided()
	} else {
		// Either put object doesn't come with a custom key or Prebid Cache is configured
		// to not use custom keys. Generate a random UUID
		if resp.UUID, err = utils.GenerateRandomID(); err != nil {
			resp.UUID = ""
			resp.err = utils.NewPBCError(utils.PUT_INTERNAL_SERVER, "Error generating version 4 UUID")
			return
		}
	}

	// If we have a blank UUID, don't store anything.
	// Eventually we may want to provide error details, but as of today this is the only non-fatal error
	// Future error details could go into a second property of the Responses object, such as "errors"
	if len(resp.UUID) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err = e.backend.Put(ctx, resp.UUID, toCache, po.TTLSeconds)
		if err != nil {
			if pbcErr, isPbcErr := err.(utils.PBCError); isPbcErr && pbcErr.Type == utils.RECORD_EXISTS {
				// Record didn't get overwritten, return a response with an empty UUID string
				resp.UUID = ""
			} else {
				resp.err = classifyBackendError(err, index)
			}
		}
	}
	return
}

type putRequest struct {
	Puts []putObject `json:"puts"`
}

type putObject struct {
	Type       string          `json:"type"`
	TTLSeconds int             `json:"ttlseconds"`
	Value      json.RawMessage `json:"value"`
	Key        string          `json:"key"`
}

type putResponseObject struct {
	UUID string `json:"uuid"`
	err  error
}

// PutResponse will be marshaled to be written into the http response
type PutResponse struct {
	Responses []putResponseObject `json:"responses"`
}
