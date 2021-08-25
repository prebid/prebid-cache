package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"

	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

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

func TestValidatePutObject(t *testing.T) {
	testCases := []struct {
		desc          string
		in            PutObject
		expectedError error
	}{
		{
			"empty value, expect error",
			PutObject{},
			errors.New("Missing required field value."),
		},
		{
			"negative time-to-live, expect error",
			PutObject{
				TTLSeconds: -1,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			errors.New("ttlseconds must not be negative -1."),
		},
		{
			"non xml nor json type, expect error",
			PutObject{
				Type:       "unknown",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			errors.New("Type must be one of [\"json\", \"xml\"]. Found unknown"),
		},
		{
			"xml type value is not a string, expect error",
			PutObject{
				Type:       "xml",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>XML</tag>`),
			},
			errors.New("XML messages must have a String value. Found [60 116 97 103 62 88 77 76 60 47 116 97 103 62]"),
		},
		{
			"valid xml input, no errors expected",
			PutObject{
				Type:       "xml",
				TTLSeconds: 60,
				Value:      json.RawMessage(`"<tag>XML</tag>"`),
			},
			nil,
		},
		{
			"valid JSON input, no errors expected",
			PutObject{
				Type:       "json",
				TTLSeconds: 60,
				Value:      json.RawMessage(`{"native":"{\"context\":1,\"plcmttype\":1,\"assets\":[{\"img\":{\"wmin\":30}}]}}`),
			},
			nil,
		},
	}
	for _, tc := range testCases {
		// run
		outErr := validatePutObject(tc.in)
		// assertions
		assert.Equal(t, tc.expectedError, outErr, tc.desc)
	}
}

func TestFormatPutError(t *testing.T) {
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
		err, errCode := formatPutError(tc.inError, 0)

		// assertions
		assert.Equal(t, tc.expected.err, err, tc.desc)
		assert.Equal(t, tc.expected.code, errCode, tc.desc)
	}
}
