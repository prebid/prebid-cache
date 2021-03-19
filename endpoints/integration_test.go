package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

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

// expectStored makes a POST request with the given putBody, and then makes sure that expectedGet
// is returned by the GET request for whatever UUID the server chose.
func expectStored(t *testing.T, putBody string, expectedGet string, expectedMimeType string) {
	router := httprouter.New()
	backend := backends.NewMemoryBackend()

	router.POST("/cache", NewPutHandler(backend, 10, true))
	router.GET("/cache", NewGetHandler(backend, true))

	uuid, putTrace := doMockPut(t, router, putBody)
	if putTrace.Code != http.StatusOK {
		t.Fatalf("Put command failed. Status: %d, Msg: %v", putTrace.Code, putTrace.Body.String())
		return
	}

	getResults := doMockGet(t, router, uuid)
	if getResults.Code != http.StatusOK {
		t.Fatalf("Get command failed with status: %d", getResults.Code)
		return
	}
	if getResults.Body.String() != expectedGet {
		t.Fatalf("Expected GET response %v to equal %v", getResults.Body.String(), expectedGet)
		return
	}
	if getResults.Header().Get("Content-Type") != expectedMimeType {
		t.Fatalf("Expected GET response Content-Type %v to equal %v", getResults.Header().Get("Content-Type"), expectedMimeType)
	}
}

// expectFailedPut makes a POST request with the given request body, and fails unless the server
// responds with a 400
func expectFailedPut(t *testing.T, requestBody string) {
	backend := backends.NewMemoryBackend()
	router := httprouter.New()
	router.POST("/cache", NewPutHandler(backend, 10, true))

	_, putTrace := doMockPut(t, router, requestBody)
	if putTrace.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 response. Got: %d, Msg: %v", putTrace.Code, putTrace.Body.String())
		return
	}
}

func TestJSONString(t *testing.T) {
	expectStored(
		t,
		"{\"puts\":[{\"type\":\"json\",\"value\":\"plain text\"}]}",
		"\"plain text\"",
		"application/json")
}

func TestEscapedString(t *testing.T) {
	expectStored(
		t,
		"{\"puts\":[{\"type\":\"json\",\"value\":\"esca\\\"ped\"}]}",
		"\"esca\\\"ped\"",
		"application/json")
}

func TestUnescapedString(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"json\",\"value\":\"badly-esca\"ped\"}]}")
}

func TestNumber(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"json\",\"value\":5}]}", "5", "application/json")
}

func TestObject(t *testing.T) {
	expectStored(
		t,
		"{\"puts\":[{\"type\":\"json\",\"value\":{\"custom_key\":\"foo\"}}]}",
		"{\"custom_key\":\"foo\"}",
		"application/json")
}

func TestNull(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"json\",\"value\":null}]}", "null", "application/json")
}

func TestBoolean(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"json\",\"value\":true}]}", "true", "application/json")
}

func TestExtraProperty(t *testing.T) {
	expectStored(
		t,
		"{\"puts\":[{\"type\":\"json\",\"value\":null,\"irrelevant\":\"foo\"}]}",
		"null",
		"application/json")
}

func TestInvalidJSON(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"json\",\"value\":malformed}]}")
}

func TestMissingProperty(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"json\",\"unrecognized\":true}]}")
}

func TestMixedValidityPuts(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"json\",\"value\":true}, {\"type\":\"json\",\"unrecognized\":true}]}")
}

func TestXMLString(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag></tag>\"}]}", "<tag></tag>", "application/xml")
}

func TestCrossScriptEscaping(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag>esc\\\"aped</tag>\"}]}", "<tag>esc\"aped</tag>", "application/xml")
}

func TestXMLOther(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"xml\",\"value\":5}]}")
}

func TestGetHandler(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	type testInput struct {
		uuid      string
		allowKeys bool
	}
	type testOutput struct {
		responseCode int
		responseBody string
		logEntries   []logEntry
	}

	testCases := []struct {
		desc string
		in   testInput
		out  testOutput
	}{
		{
			"Missing UUID. Return http error but don't interrupt server's execution",
			testInput{uuid: ""},
			testOutput{
				responseCode: http.StatusBadRequest,
				responseBody: "GET /cache: Missing required parameter uuid\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache: Missing required parameter uuid",
						lvl: logrus.ErrorLevel,
					},
				},
			},
		},
		{
			"Test uses backend that doesn't allow for keys different than 36 char long. Respond with http error and don't interrupt server's execution",
			testInput{uuid: "non-36-char-key-maps-to-json"},
			testOutput{
				responseCode: http.StatusNotFound,
				responseBody: "GET /cache uuid=non-36-char-key-maps-to-json: invalid uuid length\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache uuid=non-36-char-key-maps-to-json: invalid uuid length",
						lvl: logrus.ErrorLevel,
					},
				},
			},
		},
		{
			"Test uses backend that allows for different than 36 char long uuids. Since the uuid maps to a value, return it along a 200 status code",
			testInput{
				uuid:      "non-36-char-key-maps-to-json",
				allowKeys: true,
			},
			testOutput{
				responseCode: http.StatusOK,
				responseBody: `{"field":"value"}`,
				logEntries:   []logEntry{},
			},
		},
		{
			"Valid 36 char long UUID not found in database. Return http error but don't interrupt server's execution",
			testInput{uuid: "uuid-not-found-and-links-to-no-value"},
			testOutput{
				responseCode: http.StatusNotFound,
				responseBody: "GET /cache uuid=uuid-not-found-and-links-to-no-value:  Key not found\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache uuid=uuid-not-found-and-links-to-no-value:  Key not found",
						lvl: logrus.DebugLevel,
					},
				},
			},
		},
		{
			"Data from backend is not preceeded by 'xml' nor 'json' string. Return http error but don't interrupt server's execution",
			testInput{uuid: "36-char-key-maps-to-non-xml-nor-json"},
			testOutput{
				responseCode: http.StatusInternalServerError,
				responseBody: "GET /cache uuid=36-char-key-maps-to-non-xml-nor-json: Cache data was corrupted. Cannot determine type.\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache uuid=36-char-key-maps-to-non-xml-nor-json: Cache data was corrupted. Cannot determine type.",
						lvl: logrus.ErrorLevel,
					},
				},
			},
		},
		{
			"Valid 36 char long UUID returns valid XML. Don't return nor log error",
			testInput{uuid: "36-char-key-maps-to-actual-xml-value"},
			testOutput{
				responseCode: http.StatusOK,
				responseBody: "<tag>xml data here</tag>",
				logEntries:   []logEntry{},
			},
		},
	}

	// Lower Log Treshold so we can see DebugLevel entries in our mock logrus log
	logrus.SetLevel(logrus.DebugLevel)

	// Test suite-wide objects
	hook := test.NewGlobal()

	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	var fatal bool
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	for _, test := range testCases {
		// Reset the fatal flag to false every test
		fatal = false

		// Set up test object
		backend := newMockBackend()
		router := httprouter.New()
		router.GET("/cache", NewGetHandler(backend, test.in.allowKeys))

		// Run test
		getResults := doMockGet(t, router, test.in.uuid)

		// Assert server response and status code
		assert.Equal(t, test.out.responseCode, getResults.Code, test.desc)
		assert.Equal(t, test.out.responseBody, getResults.Body.String(), test.desc)

		// Assert log entries
		if assert.Len(t, hook.Entries, len(test.out.logEntries), test.desc) {
			for i := 0; i < len(test.out.logEntries); i++ {
				assert.Equal(t, test.out.logEntries[i].msg, hook.Entries[i].Message, test.desc)
				assert.Equal(t, test.out.logEntries[i].lvl, hook.Entries[i].Level, test.desc)
			}
			// Assert the logger didn't exit the program
			assert.False(t, fatal, test.desc)
		}

		// Reset log
		hook.Reset()
	}
}

func TestLogAtLevel(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	testCases := []struct {
		desc        string
		inLogLevel  logrus.Level
		outLogEntry logEntry
	}{
		{
			"Fatal log entry",
			logrus.FatalLevel,
			logEntry{msg: "Fatal log entry", lvl: logrus.FatalLevel},
		},
		{
			"Error log entry",
			logrus.ErrorLevel,
			logEntry{msg: "Error log entry", lvl: logrus.ErrorLevel},
		},
		{
			"Warning log entry",
			logrus.WarnLevel,
			logEntry{msg: "Warning log entry", lvl: logrus.WarnLevel},
		},
		{
			"Info log entry",
			logrus.InfoLevel,
			logEntry{msg: "Info log entry", lvl: logrus.InfoLevel},
		},
		{
			"Debug log entry",
			logrus.DebugLevel,
			logEntry{msg: "Debug log entry", lvl: logrus.DebugLevel},
		},
		{
			"Trace log entry",
			logrus.TraceLevel,
			logEntry{msg: "Trace log entry", lvl: logrus.TraceLevel},
		},
	}

	// Lower Log Treshold so logrus picks all entries in this test
	logrus.SetLevel(logrus.TraceLevel)

	// Test suite-wide objects
	hook := test.NewGlobal()

	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, tc := range testCases {
		// Run test
		logAtLevel(tc.inLogLevel, tc.desc)

		// assert log entry
		assert.Equal(t, tc.outLogEntry.msg, hook.Entries[0].Message, tc.desc)
		assert.Equal(t, tc.outLogEntry.lvl, hook.Entries[0].Level, tc.desc)

		// Reset log
		hook.Reset()
	}
}

func TestReadinessCheck(t *testing.T) {
	requestRecorder := httptest.NewRecorder()

	router := httprouter.New()
	router.GET("/status", Status)
	req, _ := http.NewRequest("GET", "/status", new(bytes.Buffer))
	router.ServeHTTP(requestRecorder, req)

	if requestRecorder.Code != http.StatusNoContent {
		t.Errorf("/status endpoint should always return a 204. Got %d", requestRecorder.Code)
	}
}

func TestMultiPutRequestGotStored(t *testing.T) {
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

	router.POST("/cache", NewPutHandler(backend, 10, true))
	router.GET("/cache", NewGetHandler(backend, true))

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

	router.POST("/cache", NewPutHandler(backend, 10, true))

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

	router.POST("/cache", NewPutHandler(backend, 10, true))
	router.GET("/cache", NewGetHandler(backend, true))

	rr := httptest.NewRecorder()

	//for statement to execute handler function
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		router.ServeHTTP(rr, request)
		b.StopTimer()
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

type mockBackend struct {
	data map[string]string
}

func (b *mockBackend) Get(ctx context.Context, key string) (string, error) {
	v, ok := b.data[key]
	if !ok {
		return "", backends.KeyNotFoundError{}
	}
	return v, nil
}

func (b *mockBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	b.data[key] = value
	return nil
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		data: map[string]string{
			"non-36-char-key-maps-to-json":         `json{"field":"value"}`,
			"36-char-key-maps-to-non-xml-nor-json": `#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`,
			"36-char-key-maps-to-actual-xml-value": "xml<tag>xml data here</tag>",
		},
	}
}
