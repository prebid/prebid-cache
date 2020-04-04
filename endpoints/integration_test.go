package endpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/backends/decorators"
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

func TestNonJsonNorXMLString(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"yaml\",\"value\":\"<tag></tag>\"}]}")
}

func TestEmptyPutsRequest(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[]}")
}

func TestCrossScriptEscaping(t *testing.T) {
	expectStored(t, "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag>esc\\\"aped</tag>\"}]}", "<tag>esc\"aped</tag>", "application/xml")
}

func TestXMLOther(t *testing.T) {
	expectFailedPut(t, "{\"puts\":[{\"type\":\"xml\",\"value\":5}]}")
}

func TestGetInvalidUUIDs(t *testing.T) {
	backend := backends.NewMemoryBackend()
	router := httprouter.New()
	router.GET("/cache", NewGetHandler(backend, false))

	getResults := doMockGet(t, router, "fdd9405b-ef2b-46da-a55a-2f526d338e16")
	if getResults.Code != http.StatusNotFound {
		t.Fatalf("Expected GET to return 404 on unrecognized ID. Got: %d", getResults.Code)
		return
	}

	getResults = doMockGet(t, router, "abc")
	if getResults.Code != http.StatusNotFound {
		t.Fatalf("Expected GET to return 404 on unrecognized ID. Got: %d", getResults.Code)
		return
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

func TestNegativeTTL(t *testing.T) {
	exitData := &exitInfo{errMsg: "", status: http.StatusOK}
	backend := &backendCallObject{
		put: &PutRequest{
			Puts: []PutObject{
				{
					Type:       "xml",
					TTLSeconds: -1,
					Value:      json.RawMessage(`<tag></tag>`),
					Key:        "key-value",
				},
			},
		},
	}
	validateAndEncode(backend, exitData)
	assert.Equalf(t, http.StatusBadRequest, exitData.status, "Expected 400 response. Got: %d", exitData.status)
}

func TestUpdateElement(t *testing.T) {
	// Original value to store
	originalValue := "true"
	reqBody := fmt.Sprintf("{\"puts\":[{\"type\":\"json\",\"value\":%s,\"key\":\"\"}]}", originalValue)

	request, err := http.NewRequest("POST", "/cache", strings.NewReader(reqBody))
	if err != nil {
		t.Errorf("Failed to create a POST request: %v", err)
	}

	//Set up server and run
	router := httprouter.New()
	backend := backends.NewMemoryBackend()

	router.POST("/cache", NewPutHandler(backend, 10, true))
	router.GET("/cache", NewGetHandler(backend, true))

	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, request)

	// extract newly created UUID
	var resp1, resp2 PutResponse
	err = json.Unmarshal([]byte(rec1.Body.String()), &resp1)
	if err != nil {
		t.Errorf("Response from POST doesn't conform to the expected format: %s", rec1.Body.String())
	}

	// Store new value in same location
	newValue := "false"
	locationToStoreNewValue := resp1.Responses[0].UUID

	reqBody = fmt.Sprintf("{\"puts\":[{\"type\":\"json\",\"value\":%s,\"key\":\"%s\"}]}", newValue, locationToStoreNewValue)

	anotherRequest, err2 := http.NewRequest("POST", "/cache", strings.NewReader(reqBody))
	if err2 != nil {
		t.Errorf("Failed to create a POST request: %v", err)
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, anotherRequest)

	// Unmarshall new results and assert
	err = json.Unmarshal([]byte(rec2.Body.String()), &resp2)
	if err != nil {
		t.Errorf("Response from POST doesn't conform to the expected format: %s", rec2.Body.String())
	}
	getResult := doMockGet(t, router, resp2.Responses[0].UUID)

	assert.Equal(t, http.StatusOK, rec2.Code, "Failed to rewrite element with known UUID")
	assert.Equalf(t, "", resp2.Responses[0].UUID, "Element with known UUID was expected to hold the new value: %s. Actual: %s \n", newValue, getResult.Body.String())
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
	router.POST("/cache", NewPutHandler(backend, len(putElements)-1, true))

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
	if err != nil {
		t.Errorf("Failed to create a POST request: %v", err)
	}

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
	if err != nil {
		t.Errorf("Response from POST doesn't conform to the expected format: %s", rr.Body.String())
	}
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
	backend := decorators.EnforceSizeLimit(backends.NewMemoryBackend(), sizeLimit)

	// Run client
	router := httprouter.New()
	router.POST("/cache", NewPutHandler(backend, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, http.StatusBadRequest, httpTestRecorder.Code, "doMockPut should have failed when trying to store elements in sizeCappedBackend")
}

func TestInternalPutClientError(t *testing.T) {
	// Valid request
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"text longer than size limit\"}]}"

	// Use mock client that will return an error
	backend := backends.NewErrorReturningBackend()

	// Run client
	router := httprouter.New()
	router.POST("/cache", NewPutHandler(backend, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, httpTestRecorder.Code, "Put should have failed because we are using an MockReturnErrorBackend")
}

func TestPutClientDeadlineExceeded(t *testing.T) {
	// Valid request
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"text longer than size limit\"}]}"

	// Use mock client that will return an error
	backend := backends.NewDeadlineExceededBackend()

	// Run client
	router := httprouter.New()
	router.POST("/cache", NewPutHandler(backend, 10, true))

	_, httpTestRecorder := doMockPut(t, router, reqBody)

	// Assert
	assert.Equal(t, HttpDependencyTimeout, httpTestRecorder.Code, "Put should have failed because we are using a MockDeadlineExceededBackend")
}

func TestEmptyRequestBodyError(t *testing.T) {
	emptyRequest := ""

	// Use mock client that will return an error
	backend := backends.NewMemoryBackend()

	// Run client
	router := httprouter.New()
	router.POST("/cache", NewPutHandler(backend, 10, true))

	_, httpTestRecorder := doMockPut(t, router, emptyRequest)

	// Assert
	assert.Equalf(t, http.StatusBadRequest, httpTestRecorder.Code, "Put client should have failed because we passed badly escaped Json: %s \n", httpTestRecorder.Body.String())
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
