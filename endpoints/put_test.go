package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestStatusEndpointReadiness asserts the http://<prebid-cache-host>/status endpoint
// is responds as expected.
func TestStatusEndpointReadiness(t *testing.T) {
	// Set up
	requestRecorder := httptest.NewRecorder()

	router := httprouter.New()
	router.GET("/status", Status)
	req, _ := http.NewRequest("GET", "/status", new(bytes.Buffer))

	// Run
	router.ServeHTTP(requestRecorder, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, requestRecorder.Code, "/status endpoint should always return a 204. Got %d", requestRecorder.Code)
}

// TestSuccessfulPut asserts the *PuntHandler.handle() function both successfully
// stores the incomming request value and responds with an http.StatusOK code
func TestSuccessfulPut(t *testing.T) {
	type testCase struct {
		desc                string
		inPutBody           string
		expectedStoredValue string
	}

	testGroups := []struct {
		groupDesc            string
		expectedResponseType string
		testCases            []testCase
	}{
		{
			groupDesc:            "Store Json values",
			expectedResponseType: "application/json",
			testCases: []testCase{
				{
					desc:                "TestJSONString",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":\"plain text\"}]}",
					expectedStoredValue: "\"plain text\"",
				},
				{
					desc:                "TestEscapedString",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":\"esca\\\"ped\"}]}",
					expectedStoredValue: "\"esca\\\"ped\"",
				},
				{
					desc:                "TestNumber",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":5}]}",
					expectedStoredValue: "5",
				},
				{
					desc:                "TestObject",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":{\"custom_key\":\"foo\"}}]}",
					expectedStoredValue: "{\"custom_key\":\"foo\"}",
				},
				{
					desc:                "TestNull",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":null}]}",
					expectedStoredValue: "null",
				},
				{
					desc:                "TestBoolean",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":true}]}",
					expectedStoredValue: "true",
				},
				{
					desc:                "TestExtraProperty",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":null,\"irrelevant\":\"foo\"}]}",
					expectedStoredValue: "null",
				},
			},
		},
		{
			groupDesc:            "Store XML",
			expectedResponseType: "application/xml",
			testCases: []testCase{
				{
					desc:                "Regular ",
					inPutBody:           "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag></tag>\"}]}",
					expectedStoredValue: "<tag></tag>",
				},
				{
					desc:                "TestCrossScriptEscaping",
					inPutBody:           "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag>esc\\\"aped</tag>\"}]}",
					expectedStoredValue: "<tag>esc\"aped</tag>",
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// set test
			router := httprouter.New()
			backend := backends.NewMemoryBackend()
			m := metricstest.CreateMockMetrics()

			router.POST("/cache", NewPutHandler(backend, m, 10, true))
			router.GET("/cache", NewGetHandler(backend, m, true))

			// Feed the tests input put request to the endpoint's handle
			uuid, putTrace := doMockPut(t, router, tc.inPutBody)
			if !assert.Equal(t, http.StatusOK, putTrace.Code, "%s - %s: Put() call failed. Status: %d, Msg: %v", group.groupDesc, tc.desc, putTrace.Code, putTrace.Body.String()) {
				return
			}

			// assert the put call above acurately stored the expected test data.
			getResults := doMockGet(t, router, uuid)
			if !assert.Equal(t, http.StatusOK, getResults.Code, "%s - %s: Get() failed with status: %d", group.groupDesc, tc.desc, getResults.Code) {
				return
			}
			if !assert.Equal(t, tc.expectedStoredValue, getResults.Body.String(), "%s - %s: Put() call didn't store the expected value", group.groupDesc, tc.desc) {
				return
			}
			if getResults.Header().Get("Content-Type") != group.expectedResponseType {
				t.Fatalf("%s - %s: Expected GET response Content-Type %v to equal %v", group.groupDesc, tc.desc, getResults.Header().Get("Content-Type"), group.expectedResponseType)
			}
		}
	}
}

// TestMalformedOrInvalidValue asserts the *PuntHandler.handle() function successfully responds
// with a http.StatusBadRequest given malformed or missing `value` field
func TestMalformedOrInvalidValue(t *testing.T) {
	testCases := []struct {
		desc      string
		inPutBody string
	}{
		{
			"Badly escaped character in value field",
			"{\"puts\":[{\"type\":\"json\",\"value\":\"badly-esca\"ped\"}]}",
		},
		{
			"Malformed JSON in value field",
			"{\"puts\":[{\"type\":\"json\",\"value\":malformed}]}",
		},
		{
			"Missing value field in sole element of puts array",
			"{\"puts\":[{\"type\":\"json\",\"unrecognized\":true}]}",
		},
		{
			"Missing value field in at least one element of puts array",
			"{\"puts\":[{\"type\":\"json\",\"value\":true}, {\"type\":\"json\",\"unrecognized\":true}]}",
		},
		{
			"Invalid XML",
			"{\"puts\":[{\"type\":\"xml\",\"value\":5}]}",
		},
	}

	for _, tc := range testCases {
		// setup test
		router := httprouter.New()
		backend := backends.NewMemoryBackend()
		m := metricstest.CreateMockMetrics()

		router.POST("/cache", NewPutHandler(backend, m, 10, true))

		// Run test
		_, putTrace := doMockPut(t, router, tc.inPutBody)

		// Assert
		assert.Equal(t, http.StatusBadRequest, putTrace.Code, "%s: Put() call expected 400 response. Got: %d, Msg: %v", tc.desc, putTrace.Code, putTrace.Body.String())
	}
}

// TestNonSupportedType asserts the *PuntHandler.handle() function successfully responds
// with a http.StatusBadRequest code if the value under the incomming request's `type` field
// refers to an unsupported data type.
func TestNonSupportedType(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"yaml\",\"value\":\"<tag></tag>\"}]}")
}

func TestPutNegativeTTL(t *testing.T) {
	// Input
	inReqBody := "{\"puts\":[{\"type\":\"json\",\"value\":\"<tag>YourXMLcontentgoeshere.</tag>\",\"ttlseconds\":-1}]}"
	inRequest, err := http.NewRequest("POST", "/cache", strings.NewReader(inReqBody))
	assert.NoError(t, err, "Failed to create a POST request: %v", err)

	// Expected Values
	expectedErrorMsg := "ttlseconds must not be negative -1.\n"
	expectedStatusCode := http.StatusBadRequest

	// Set up server to run our test
	testRouter := httprouter.New()
	testBackend := backends.NewMemoryBackend()
	m := metricstest.CreateMockMetrics()

	testRouter.POST("/cache", NewPutHandler(testBackend, m, 10, true))

	recorder := httptest.NewRecorder()

	// Run test
	testRouter.ServeHTTP(recorder, inRequest)

	// Assertions
	assert.Equal(t, expectedErrorMsg, recorder.Body.String(), "Put should have failed because we passed a negative ttlseconds value.\n")
	assert.Equalf(t, expectedStatusCode, recorder.Code, "Expected 400 response. Got: %d", recorder.Code)
}

func TestCustomKey(t *testing.T) {
	type aTest struct {
		desc         string
		inCustomKey  string
		expectedUuid string
	}
	testGroups := []struct {
		allowSettingKeys bool
		testCases        []aTest
	}{
		{
			allowSettingKeys: false,
			testCases: []aTest{
				{
					desc:         "Custom key maps to element in cache but setting keys is not allowed, set value with random UUID",
					inCustomKey:  "36-char-key-maps-to-actual-xml-value",
					expectedUuid: `[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`,
				},
				{
					desc:         "Custom key maps to no element in cache, set value with random UUID and respond 200",
					inCustomKey:  "36-char-key-maps-to-actual-xml-value",
					expectedUuid: `[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`,
				},
			},
		},
		{
			allowSettingKeys: true,
			testCases: []aTest{
				{
					desc:         "Setting keys allowed but key already maps to an element in cache, don't set value and respond with blank UUID",
					inCustomKey:  "36-char-key-maps-to-actual-xml-value",
					expectedUuid: "",
				},
				{
					desc:         "Custom key maps to no element in cache, set value and respond with 200 and the custom UUID",
					inCustomKey:  "cust-key-maps-to-no-value-in-backend",
					expectedUuid: "cust-key-maps-to-no-value-in-backend",
				},
			},
		},
	}

	mockBackendWithValues := newMockBackend()
	m := metricstest.CreateMockMetrics()

	for _, tgroup := range testGroups {
		for _, tc := range tgroup.testCases {
			// Instantiate prebid cache prod server with mock metrics and a mock metrics that
			// already contains some values
			router := httprouter.New()
			putEndpointHandler := NewPutHandler(mockBackendWithValues, m, 10, tgroup.allowSettingKeys)
			router.POST("/cache", putEndpointHandler)

			recorder := httptest.NewRecorder()

			reqBody := fmt.Sprintf(`{"puts":[{"type":"json","value":"xml<tag>updated_value</tag>","key":"%s"}]}`, tc.inCustomKey)
			request, err := http.NewRequest("POST", "/cache", strings.NewReader(reqBody))
			assert.NoError(t, err, "Test request could not be created")

			// Run test
			router.ServeHTTP(recorder, request)

			// Assert status code. All scenarios should return a 200 code
			assert.Equal(t, http.StatusOK, recorder.Code, tc.desc)

			// Assert response UUID
			if tc.expectedUuid == "" {
				assert.Equalf(t, `{"responses":[{"uuid":""}]}`, recorder.Body.String(), tc.desc)
			} else {
				re, err := regexp.Compile(tc.expectedUuid)
				assert.NoError(t, err, tc.desc)
				assert.Greater(t, len(re.Find(recorder.Body.Bytes())), 0, tc.desc)
			}
		}
	}
}

func TestRequestReadError(t *testing.T) {
	// Setup server and mock body request reader
	mockBackendWithValues := newMockBackend()
	m := metricstest.CreateMockMetrics()
	putEndpointHandler := NewPutHandler(mockBackendWithValues, m, 10, false)

	router := httprouter.New()
	router.POST("/cache", putEndpointHandler)

	recorder := httptest.NewRecorder()

	// make our request body reader's Read() and Close() methods to return errors
	mockRequestReader := faultyRequestBodyReader{}
	mockRequestReader.On("Read", mock.AnythingOfType("[]uint8")).Return(0, errors.New("Read error"))
	mockRequestReader.On("Close").Return(errors.New("Read error"))

	request, _ := http.NewRequest("POST", "/cache", &mockRequestReader)

	// Run test
	router.ServeHTTP(recorder, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Expected a bad request status code from a malformed request")
}

func TestTooManyPutElements(t *testing.T) {
	// Test case: request with more than elements than put handler's max number of values
	putElements := []string{
		"{\"type\":\"json\",\"value\":true}",
		"{\"type\":\"xml\",\"value\":\"plain text\"}",
		"{\"type\":\"xml\",\"value\":\"2\"}",
	}
	reqBody := fmt.Sprintf("{\"puts\":[%s, %s, %s]}", putElements[0], putElements[1], putElements[2])

	//Set up server with capacity to handle less than putElements.size()
	backend := backends.NewMemoryBackend()
	router := httprouter.New()
	m := metricstest.CreateMockMetrics()
	router.POST("/cache", NewPutHandler(backend, m, len(putElements)-1, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)
	assert.Equalf(t, http.StatusBadRequest, httpTestRecorder.Code, "doMockPut should have failed when trying to store %d elements because capacity is %d ", len(putElements), len(putElements)-1)
}

func TestMultiPutRequest(t *testing.T) {
	// Test case: request with more than one element in the "puts" array
	type aTest struct {
		description         string
		elemToPut           string
		expectedStoredValue string
	}
	testCases := []aTest{
		{
			description:         "Post in JSON format that contains a bool",
			elemToPut:           "{\"type\":\"json\",\"value\":true}",
			expectedStoredValue: "true",
		},
		{
			description:         "Post in XML format containing plain text",
			elemToPut:           "{\"type\":\"xml\",\"value\":\"plain text\"}",
			expectedStoredValue: "plain text",
		},
		{
			description:         "Post in XML format containing escaped double quotes",
			elemToPut:           "{\"type\":\"xml\",\"value\":\"2\"}",
			expectedStoredValue: "2",
		},
	}
	reqBody := fmt.Sprintf("{\"puts\":[%s, %s, %s]}", testCases[0].elemToPut, testCases[1].elemToPut, testCases[2].elemToPut)

	request, err := http.NewRequest("POST", "/cache", strings.NewReader(reqBody))
	assert.NoError(t, err, "Failed to create a POST request: %v", err)

	//Set up server and run
	router := httprouter.New()
	backend := backends.NewMemoryBackend()
	m := metricstest.CreateMockMetrics()

	router.POST("/cache", NewPutHandler(backend, m, 10, true))
	router.GET("/cache", NewGetHandler(backend, m, true))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, request)

	// validate results
	var parsed PutResponse
	err = json.Unmarshal([]byte(rr.Body.String()), &parsed)
	assert.NoError(t, err, "Response from POST doesn't conform to the expected format: %s", rr.Body.String())

	for i, resp := range parsed.Responses {
		// Get value for this UUID. It is supposed to have been stored
		getResult := doMockGet(t, router, resp.UUID)

		// Assertions
		assert.Equalf(t, http.StatusOK, getResult.Code, "Description: %s \n Multi-element put failed to store:%s \n", testCases[i].description, testCases[i].elemToPut)
		assert.Equalf(t, testCases[i].expectedStoredValue, getResult.Body.String(), "GET response error. Expected %v. Actual %v", testCases[i].expectedStoredValue, getResult.Body.String())
	}
}

func TestBadPayloadSizePutError(t *testing.T) {
	// Stored value size_limit
	sizeLimit := 3

	// Request with a string longer than sizeLimit
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"text longer than size limit\"}]}"

	// Declare a sizeCappedBackend client
	backend := backendDecorators.EnforceSizeLimit(backends.NewMemoryBackend(), sizeLimit)

	// Run client
	router := httprouter.New()
	m := metricstest.CreateMockMetrics()
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, http.StatusBadRequest, httpTestRecorder.Code, "doMockPut should have failed when trying to store elements in sizeCappedBackend")
}

func TestInternalPutClientError(t *testing.T) {
	// Valid request
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"text longer than size limit\"}]}"

	// Use mock client that will return an error
	backend := NewErrorReturningBackend()

	// Run client
	router := httprouter.New()
	m := metricstest.CreateMockMetrics()
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, httpTestRecorder.Code, "Put should have failed because we are using an MockReturnErrorBackend")
}

func TestEmptyPutRequests(t *testing.T) {
	// Test case: request with more than one element in the "puts" array
	type aTest struct {
		description      string
		reqBody          string
		expectedResponse string
		emptyResponses   bool
	}
	testCases := []aTest{
		{
			description:      "Blank value in put element",
			reqBody:          "{\"puts\":[{\"type\":\"xml\",\"value\":\"\"}]}",
			expectedResponse: "{\"responses\":[\"uuid\":\"\"]}",
			emptyResponses:   false,
		},
		// This test is meant to come right after the "Blank value in put element" test in order to assert the correction
		// of a bug in the pre-PR#64 version of `endpoints/put.go`
		{
			description:      "All empty body. ",
			reqBody:          "{}",
			expectedResponse: "{\"responses\":[]}",
			emptyResponses:   true,
		},
		{
			description:      "Empty puts arrray",
			reqBody:          "{\"puts\":[]}",
			expectedResponse: "{\"responses\":[]}",
			emptyResponses:   true,
		},
	}

	// Set up server
	router := httprouter.New()
	backend := backends.NewMemoryBackend()
	m := metricstest.CreateMockMetrics()

	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	for i, test := range testCases {
		rr := httptest.NewRecorder()

		// Create request everytime
		request, err := http.NewRequest("POST", "/cache", strings.NewReader(test.reqBody))
		assert.NoError(t, err, "[%d] Failed to create a POST request: %v", i, err)

		// Run
		router.ServeHTTP(rr, request)
		assert.Equal(t, http.StatusOK, rr.Code, "[%d] ServeHTTP(rr, request) failed = %v \n", i, rr.Result())

		// validate results
		if test.emptyResponses && !assert.Equal(t, test.expectedResponse, rr.Body.String(), "[%d] Text response not empty as expected", i) {
			return
		}

		var parsed PutResponse
		err2 := json.Unmarshal([]byte(rr.Body.String()), &parsed)
		assert.NoError(t, err2, "[%d] Error found trying to unmarshal: %s \n", i, rr.Body.String())

		if test.emptyResponses {
			assert.Equal(t, 0, len(parsed.Responses), "[%d] This is NOT an empty response len(parsed.Responses) = %d; parsed.Responses = %v \n", i, len(parsed.Responses), parsed.Responses)
		} else {
			assert.Greater(t, len(parsed.Responses), 0, "[%d] This is an empty response len(parsed.Responses) = %d; parsed.Responses = %v \n", i, len(parsed.Responses), parsed.Responses)
		}
	}
}

func TestPutClientDeadlineExceeded(t *testing.T) {
	// Valid request
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"text longer than size limit\"}]}"

	// Use mock client that will return an error
	backend := NewDeadlineExceededBackend()

	// Run client
	router := httprouter.New()
	m := metricstest.CreateMockMetrics()
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, HttpDependencyTimeout, httpTestRecorder.Code, "Put should have failed because we are using a MockDeadlineExceededBackend")
}

// TestParseRequest asserts *PutHandler's parseRequest(r *http.Request) method
func TestParseRequest(t *testing.T) {
	type testOut struct {
		put *PutRequest
		err error
	}
	testCases := []struct {
		desc            string
		getInputRequest func() *http.Request
		expected        testOut
	}{
		{
			"nil request",
			func() *http.Request { return nil },
			testOut{nil, utils.PutBadRequestError{}},
		},
		{
			"request with malformed body throws unmarshal error",
			func() *http.Request {
				r, _ := http.NewRequest("POST", "http://fakeurl.com", bytes.NewBuffer([]byte(`malformed`)))
				return r
			},
			testOut{nil, utils.PutBadRequestError{[]byte(`malformed`)}},
		},
		{
			"valid request body. Expect no error",
			func() *http.Request {
				requestBody := []byte(`{"puts":[{"type":"json","value":{"valueField":5}}]}`)
				r, _ := http.NewRequest("POST", "http://fakeurl.com", bytes.NewBuffer(requestBody))
				return r
			},
			testOut{
				&PutRequest{
					Puts: []PutObject{
						{Type: "json", Value: json.RawMessage(`{"valueField":5}`)},
					},
				},
				nil,
			},
		},
		{
			"valid request body comes with more elements in 'puts' array than PutHandler was configured to support",
			func() *http.Request {
				requestBody := []byte(`{"puts":[{"type":"xml","value":"XmlValue"}, {"type":"json","value":{"valueField":5}}]}`)
				r, _ := http.NewRequest("POST", "http://fakeurl.com", bytes.NewBuffer(requestBody))
				return r
			},
			testOut{nil, utils.PutMaxNumValuesError{2, 1}},
		},
	}
	for _, tc := range testCases {
		// set test
		putHandler := &PutHandler{
			memory: syncPools{
				requestPool: sync.Pool{
					New: func() interface{} { return &PutRequest{} },
				},
			},
			cfg: putHandlerConfig{maxNumValues: 1},
		}
		// run
		put, err := putHandler.parseRequest(tc.getInputRequest())
		// assertions
		assert.Equal(t, tc.expected.put, put, tc.desc)
		assert.Equal(t, tc.expected.err, err, tc.desc)
	}
}

// TestParsePutObject asserts *PutHandler's parsePutObject(p PutObject) method
func TestParsePutObject(t *testing.T) {
	type testOut struct {
		value string
		err   error
	}
	testCases := []struct {
		desc     string
		in       PutObject
		expected testOut
	}{
		{
			"empty value, expect error",
			PutObject{},
			testOut{
				value: "",
				err:   utils.MissingValueError{},
			},
		},
		{
			"negative time-to-live, expect error",
			PutObject{
				TTLSeconds: -1,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			testOut{
				value: "",
				err:   utils.NegativeTTLError{-1},
			},
		},
		{
			"non xml nor json type, expect error",
			PutObject{
				Type:       "unknown",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			testOut{
				value: "",
				err:   utils.UnsupportedDataToStoreError{"unknown"},
			},
		},
		{
			"xml type value is not a string, expect error",
			PutObject{
				Type:       "xml",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>XML</tag>`),
			},
			testOut{
				value: "",
				err:   utils.MalformedXMLError{"XML messages must have a String value. Found [60 116 97 103 62 88 77 76 60 47 116 97 103 62]"},
			},
		},
		{
			"xml type value is surrounded by quotes and, therefore, a string. No errors expected",
			PutObject{
				Type:       "xml",
				TTLSeconds: 60,
				Value:      json.RawMessage(`"<tag>XML</tag>"`),
			},
			testOut{
				"xml<tag>XML</tag>",
				nil,
			},
		},
		{
			"valid JSON input, no errors expected",
			PutObject{
				Type:       "json",
				TTLSeconds: 60,
				Value:      json.RawMessage(`{"native":"{\"context\":1,\"plcmttype\":1,\"assets\":[{\"img\":{\"wmin\":30}}]}}`),
			},
			testOut{
				`json{"native":"{\"context\":1,\"plcmttype\":1,\"assets\":[{\"img\":{\"wmin\":30}}]}}`,
				nil,
			},
		},
	}
	for _, tc := range testCases {
		// run
		actualPutString, actualError := parsePutObject(tc.in)

		// assertions
		assert.Equal(t, tc.expected.value, actualPutString, tc.desc)
		assert.Equal(t, tc.expected.err, actualError, tc.desc)
	}
}

// TestLogBackendError asserts this package's logBackendError(err error, index int) function
func TestLogBackendError(t *testing.T) {
	type testOutput struct {
		err  error
		code int
	}

	testCases := []struct {
		desc     string
		inError  error
		expected testOutput
	}{
		{
			"Bad payload size error",
			&backendDecorators.BadPayloadSize{Limit: 1, Size: 2},
			testOutput{
				utils.PutBadPayloadSizeError{
					Msg:   "Payload size 2 exceeded max 1",
					Index: 0,
				},
				http.StatusBadRequest,
			},
		},
		{
			"DeadlineExceeded error",
			context.DeadlineExceeded,
			testOutput{
				utils.PutDeadlineExceededError{},
				utils.HttpDependencyTimeout,
			},
		},
		{
			"Backend client error",
			errors.New("Key exist error"),
			testOutput{
				utils.PutInternalServerError{"Key exist error"},
				http.StatusInternalServerError,
			},
		},
	}
	for _, tc := range testCases {
		// run
		err := logBackendError(tc.inError, 0)

		// assertions
		assert.Equal(t, tc.expected.err, err, tc.desc)
	}
}

// expectFailedPut makes a POST request with the given request body, and fails unless the server
// responds with a 400
func expectFailedPut(t *testing.T, requestBody string) {
	backend := backends.NewMemoryBackend()
	router := httprouter.New()
	m := metricstest.CreateMockMetrics()
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	_, putTrace := doMockPut(t, router, requestBody)
	if putTrace.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 response. Got: %d, Msg: %v", putTrace.Code, putTrace.Body.String())
		return
	}
}

func BenchmarkPutHandlerLen1(b *testing.B) {
	b.StopTimer()

	input := "{\"puts\":[{\"type\":\"json\",\"value\":\"plain text\"}]}"
	benchmarkPutHandler(b, input)
}

func BenchmarkPutHandlerLen2(b *testing.B) {
	b.StopTimer()

	//Set up a request that should succeed
	input := "{\"puts\":[{\"type\":\"json\",\"value\":true}, {\"type\":\"xml\",\"value\":\"plain text\"}]}"
	benchmarkPutHandler(b, input)
}

func BenchmarkPutHandlerLen4(b *testing.B) {
	b.StopTimer()

	//Set up a request that should succeed
	input := "{\"puts\":[{\"type\":\"json\",\"value\":true}, {\"type\":\"xml\",\"value\":\"plain text\"},{\"type\":\"xml\",\"value\":5}, {\"type\":\"json\",\"value\":\"esca\\\"ped\"}]}"
	benchmarkPutHandler(b, input)
}

func BenchmarkPutHandlerLen8(b *testing.B) {
	b.StopTimer()

	//Set up a request that should succeed
	input := "{\"puts\":[{\"type\":\"json\",\"value\":true}, {\"type\":\"xml\",\"value\":\"plain text\"},{\"type\":\"xml\",\"value\":5}, {\"type\":\"json\",\"value\":\"esca\\\"ped\"}, {\"type\":\"json\",\"value\":{\"custom_key\":\"foo\"}},{\"type\":\"xml\",\"value\":{\"custom_key\":\"foo\"}},{\"type\":\"json\",\"value\":null}, {\"type\":\"xml\",\"value\":\"<tag></tag>\"}]}"
	benchmarkPutHandler(b, input)
}

func doMockGet(t *testing.T, router *httprouter.Router, id string) *httptest.ResponseRecorder {
	requestRecorder := httptest.NewRecorder()

	body := new(bytes.Buffer)
	getReq, err := http.NewRequest("GET", "/cache"+"?uuid="+id, body)
	if err != nil {
		t.Fatalf("Failed to create a GET request: %v", err)
		return requestRecorder
	}
	router.ServeHTTP(requestRecorder, getReq)
	return requestRecorder
}

func doMockPut(t *testing.T, router *httprouter.Router, content string) (string, *httptest.ResponseRecorder) {
	var parseMockUUID = func(t *testing.T, putResponse string) string {
		var parsed PutResponse
		err := json.Unmarshal([]byte(putResponse), &parsed)
		if err != nil {
			t.Errorf("Response from POST doesn't conform to the expected format: %v", putResponse)
		}
		return parsed.Responses[0].UUID
	}

	rr := httptest.NewRecorder()

	request, err := http.NewRequest("POST", "/cache", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to create a POST request: %v", err)
		return "", rr
	}

	router.ServeHTTP(rr, request)
	uuid := ""
	if rr.Code == http.StatusOK {
		uuid = parseMockUUID(t, rr.Body.String())
	}
	return uuid, rr
}

func benchmarkPutHandler(b *testing.B, testCase string) {
	b.StopTimer()
	//Set up a request that should succeed
	request, err := http.NewRequest("POST", "/cache", strings.NewReader(testCase))
	if err != nil {
		b.Errorf("Failed to create a POST request: %v", err)
	}

	//Set up server ready to run
	router := httprouter.New()
	backend := backends.NewMemoryBackend()
	m := metricstest.CreateMockMetrics()

	router.POST("/cache", NewPutHandler(backend, m, 10, true))
	router.GET("/cache", NewGetHandler(backend, m, true))

	rr := httptest.NewRecorder()

	//for statement to execute handler function
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		router.ServeHTTP(rr, request)
		b.StopTimer()
	}
}

func newMockBackend() *backends.MemoryBackend {
	backend := backends.NewMemoryBackend()

	backend.Put(context.TODO(), "non-36-char-key-maps-to-json", `json{"field":"value"}`, 0)
	backend.Put(context.TODO(), "36-char-key-maps-to-non-xml-nor-json", `#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`, 0)
	backend.Put(context.TODO(), "36-char-key-maps-to-actual-xml-value", "xml<tag>xml data here</tag>", 0)

	return backend
}

type faultyRequestBodyReader struct {
	mock.Mock
}

func (b *faultyRequestBodyReader) Read(p []byte) (n int, err error) {
	args := b.Called(p)
	return args.Int(0), args.Error(1)
}

func (b *faultyRequestBodyReader) Close() error {
	args := b.Called()
	return args.Error(0)
}

type errorReturningBackend struct{}

func (b *errorReturningBackend) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("This is a mock backend that returns this error on Get() operation")
}

func (b *errorReturningBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return fmt.Errorf("This is a mock backend that returns this error on Put() operation")
}

func NewErrorReturningBackend() *errorReturningBackend {
	return &errorReturningBackend{}
}

type deadlineExceedingBackend struct{}

func (b *deadlineExceedingBackend) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (b *deadlineExceedingBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	var err error

	d := time.Now().Add(50 * time.Millisecond)
	sampleCtx, cancel := context.WithDeadline(context.Background(), d)

	// Even though ctx will be expired, it is good practice to call its
	// cancellation function in any case. Failure to do so may keep the
	// context and its parent alive longer than necessary.
	defer cancel()

	select {
	case <-time.After(1 * time.Second):
		//err = fmt.Errorf("Some other error")
		err = nil
	case <-sampleCtx.Done():
		err = sampleCtx.Err()
	}
	return err
}

func NewDeadlineExceededBackend() *deadlineExceedingBackend {
	return &deadlineExceedingBackend{}
}
