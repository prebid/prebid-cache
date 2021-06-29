package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/stretchr/testify/mock"
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
