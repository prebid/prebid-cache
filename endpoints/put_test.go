package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendConfig "github.com/prebid/prebid-cache/backends/config"
	"github.com/prebid/prebid-cache/backends/decorators"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
	testLogrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPutJsonTests(t *testing.T) {
	testGroups := []struct {
		desc        string
		expectError bool
		tests       []string
	}{
		{
			desc:        "Valid put requests. Expect 200 response",
			expectError: false,
			tests: []string{
				"sample-requests/put-endpoint/valid-whole/single-element-to-store.json",
				"sample-requests/put-endpoint/valid-whole/no-elements-to-store.json",
				"sample-requests/put-endpoint/valid-whole/multiple-elements-to-store.json",
				"sample-requests/put-endpoint/valid-whole/valid-type-json.json",
				"sample-requests/put-endpoint/valid-whole/valid-type-xml.json",
				"sample-requests/put-endpoint/valid-whole/ttl-more-than-max.json",
				"sample-requests/put-endpoint/valid-whole/ttl-missing.json",
			},
		},
		{
			desc:        "Request tries to store more elements than the max allowed. Return error",
			expectError: true,
			tests: []string{
				"sample-requests/put-endpoint/invalid-number-of-elements/puts-max-num-values.json",
			},
		},
		{
			desc:        "Invalid 'type' field values, expect error",
			expectError: true,
			tests: []string{
				"sample-requests/put-endpoint/invalid-types/type-missing.json",
				"sample-requests/put-endpoint/invalid-types/type-unknown.json",
			},
		},
		{
			desc:        "invalid 'value' field values, expect error",
			expectError: true,
			tests: []string{
				"sample-requests/put-endpoint/invalid-value/value-missing.json",
				"sample-requests/put-endpoint/invalid-value/value-greater-than-max.json",
			},
		},
		{
			desc:        "Valid when storing under custom keys is allowed, expect 200 responses",
			expectError: false,
			tests: []string{
				"sample-requests/put-endpoint/custom-keys/allowed/key-field-included.json",
				"sample-requests/put-endpoint/custom-keys/allowed/key-field-missing.json",
			},
		},
		{
			desc:        "Valid when storing under custom keys is not allowed, expect 200 responses",
			expectError: false,
			tests: []string{
				"sample-requests/put-endpoint/custom-keys/not-allowed/key-field-included.json",
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := testLogrus.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, group := range testGroups {
		for _, testFile := range group.tests {
			// TEST SETUP
			//   Read file
			testInfo, err := parseTestInfo(testFile)
			if !assert.NoError(t, err, "%v", err) {
				continue
			}

			//   Read test config
			v := buildViperConfig(testInfo)
			cfg := config.Configuration{}
			err = v.Unmarshal(&cfg)
			if !assert.NoError(t, err, "Viper could not parse configuration from test file: %s. Error:%s\n", testFile, err) {
				continue
			}

			//   Instantiate memory backend, request, router, recorder
			mockMetrics := metricstest.CreateMockMetrics()
			m := &metrics.Metrics{
				MetricEngines: []metrics.CacheMetrics{
					&mockMetrics,
				},
			}

			backend := backendConfig.NewBackend(cfg, m)
			router := httprouter.New()
			router.POST("/cache", NewPutHandler(backend, m, testInfo.ServerConfig.MaxNumValues, testInfo.ServerConfig.AllowSettingKeys))
			request, err := http.NewRequest("POST", "/cache", strings.NewReader(string(testInfo.PutRequest)))
			assert.NoError(t, err, "Failed to create a POST request. Test file: %s Error: %v", testFile, err)
			rr := httptest.NewRecorder()

			// RUN TEST
			router.ServeHTTP(rr, request)

			// DO ASSERTIONS
			// If error is expected, assert error message and non-200 status code
			if group.expectError {
				// Given that Prebid Cache still doesn't provide error details in an "errors" field describing the particular issues
				// of each element that could not be stored, compare the entire response body that will contain the error message of
				// the element that could not be stored.
				assert.NotEqual(t, http.StatusOK, rr.Code, "Test %s failed. Expected error status code.", testFile)
				assert.Equal(t, testInfo.ExpectedError, rr.Body.String(), "Error message differs from expected. Test file: %s", testFile)
			} else {
				// Given that no error is expected, assert a 200 status code was returned
				if !assert.Equal(t, http.StatusOK, rr.Code, "Test %s failed. StatusCode = %d. Returned error: %s", testFile, rr.Code, rr.Body.String()) {
					continue
				}

				// Assert we returned the exact same elements in the 'Responses' array than in the request 'Puts' array
				actualPutResponse := PutResponse{}
				err = json.Unmarshal(rr.Body.Bytes(), &actualPutResponse)
				if !assert.NoError(t, err, "Could not unmarshal %s. Test file: %s. Error:%s\n", rr.Body.String(), testFile, err) {
					continue
				}
				assert.Len(t, actualPutResponse.Responses, len(testInfo.ExpectedResponse.Responses), "Actual response elements differ with expected. Test file: %s", testFile)

				// If custom keys are allowed, assert they are found in the actualPutResponse.Responses array
				if testInfo.ServerConfig.AllowSettingKeys {
					customKeyIndexes := []int{}

					// Unmarshal test request to extract custom keys
					put := &putRequest{
						Puts: make([]putObject, 0),
					}
					err = json.Unmarshal(testInfo.PutRequest, put)
					if !assert.NoError(t, err, "Could not put request %s. Test file: %s. Error:%s\n", testInfo.PutRequest, testFile, err) {
						continue
					}
					for i, testInputElem := range put.Puts {
						if len(testInputElem.Key) > 0 {
							customKeyIndexes = append(customKeyIndexes, i)
						}
					}

					// Custom keys values must match and their position in the `actualPutResponse.Responses` array must be the exact same as they came in
					// the incoming request
					for _, index := range customKeyIndexes {
						assert.Equal(t, testInfo.ExpectedResponse.Responses[index].UUID, actualPutResponse.Responses[index].UUID, "Custom key differs from expected in position %d. Test file: %s", index, testFile)
					}
				}
			}

			// Assert logrus expected entries
			assertLogEntries(t, testInfo.ExpectedLogEntries, hook.Entries, testFile)

			// Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry())

			// assert the put call above logged the expected metrics
			metricstest.AssertMetrics(t, testInfo.ExpectedMetrics, mockMetrics)
		}
	}
}

type testData struct {
	ServerConfig       testConfig      `json:"serverConfig"`
	PutRequest         json.RawMessage `json:"putRequest"`
	ExpectedResponse   PutResponse     `json:"expectedResponse"`
	ExpectedLogEntries []logEntry      `json:"expectedLogEntries"`
	ExpectedError      string          `json:"expectedErrorMessage"`
	ExpectedMetrics    []string        `json:"expectedMetrics"`
}

type logEntry struct {
	Message string `json:"message"`
	Level   uint32 `json:"level"`
}

type testConfig struct {
	AllowSettingKeys bool `json:"allow_setting_keys"`
	MaxSizeBytes     int  `json:"max_size_bytes"`
	MaxNumValues     int  `json:"max_num_values"`
	MaxTTLSeconds    int  `json:"max_ttl_seconds"`
}

// Remove this function in the future and make it part of the mock metrics to self-assert if possible.
func parseTestInfo(testFile string) (*testData, error) {
	var jsonTest []byte
	var err error
	if jsonTest, err = ioutil.ReadFile(testFile); err != nil {
		return nil, fmt.Errorf("Could not read test file: %s. Error: %v \n", testFile, err)
	}

	testInfo := &testData{}
	if err = json.Unmarshal(jsonTest, testInfo); err != nil {
		return nil, fmt.Errorf("Could not unmarshal test file: %s. Error:%s\n", testFile, err)
	}
	return testInfo, nil
}

func buildViperConfig(testInfo *testData) *viper.Viper {
	v := viper.New()
	v.SetDefault("backend.type", "memory")
	v.SetDefault("compression.type", "none")
	v.SetDefault("request_limits.allow_setting_keys", testInfo.ServerConfig.AllowSettingKeys)
	if testInfo.ServerConfig.MaxSizeBytes == 0 {
		testInfo.ServerConfig.MaxSizeBytes = 50
	}
	v.SetDefault("request_limits.max_size_bytes", testInfo.ServerConfig.MaxSizeBytes)

	if testInfo.ServerConfig.MaxNumValues == 0 {
		testInfo.ServerConfig.MaxNumValues = 1
	}
	v.SetDefault("request_limits.max_num_values", testInfo.ServerConfig.MaxNumValues)
	v.SetDefault("request_limits.max_ttl_seconds", testInfo.ServerConfig.MaxTTLSeconds)
	return v
}

// assertLogEntries asserts logrus entries with expectedLogEntries. It is a test helper function that will make a unit test fail if
// expected values are not found
func assertLogEntries(t *testing.T, expectedLogEntries []logEntry, actualLogEntries []logrus.Entry, testFile string) {
	t.Helper()

	assert.Equal(t, len(expectedLogEntries), len(actualLogEntries), "Incorrect number of entries were logged to logrus in test %s: len(expectedLogEntries) = %d, len(actualLogEntries) = %d", testFile, len(expectedLogEntries), len(actualLogEntries))
	for i := 0; i < len(actualLogEntries); i++ {
		assert.Equal(t, expectedLogEntries[i].Message, actualLogEntries[i].Message, "Test case %s log message differs", testFile)
		assert.Equal(t, expectedLogEntries[i].Level, uint32(actualLogEntries[i].Level), "Test case %s log level differs", testFile)
	}
}

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
		expectedMetrics     []string
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
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestEscapedString",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":\"esca\\\"ped\"}]}",
					expectedStoredValue: "\"esca\\\"ped\"",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestNumber",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":5}]}",
					expectedStoredValue: "5",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestObject",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":{\"custom_key\":\"foo\"}}]}",
					expectedStoredValue: "{\"custom_key\":\"foo\"}",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestNull",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":null}]}",
					expectedStoredValue: "null",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestBoolean",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":true}]}",
					expectedStoredValue: "true",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestExtraProperty",
					inPutBody:           "{\"puts\":[{\"type\":\"json\",\"value\":null,\"irrelevant\":\"foo\"}]}",
					expectedStoredValue: "null",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
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
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:                "TestCrossScriptEscaping",
					inPutBody:           "{\"puts\":[{\"type\":\"xml\",\"value\":\"<tag>esc\\\"aped</tag>\"}]}",
					expectedStoredValue: "<tag>esc\"aped</tag>",
					expectedMetrics: []string{
						"RecordGetTotal",
						"RecordGetDuration",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// set test
			router := httprouter.New()
			backend := backends.NewMemoryBackend()
			mockMetrics := metricstest.CreateMockMetrics()
			m := &metrics.Metrics{
				MetricEngines: []metrics.CacheMetrics{
					&mockMetrics,
				},
			}

			router.POST("/cache", NewPutHandler(backend, m, 10, true))
			router.GET("/cache", NewGetHandler(backend, m, true))

			// Feed the tests input put request to the endpoint's handle
			putResponse := doPut(t, router, tc.inPutBody)
			if !assert.Equal(t, http.StatusOK, putResponse.Code, "%s - %s: Put() call failed. Status: %d, Msg: %v", group.groupDesc, tc.desc, putResponse.Code, putResponse.Body.String()) {
				return
			}

			// Response was a 200, extract responses
			var parsed PutResponse
			err := json.Unmarshal(putResponse.Body.Bytes(), &parsed)
			assert.NoError(t, err, "Response from POST doesn't conform to the expected format: %s", putResponse.Body.String())

			// Assert responses
			for _, putResponse := range parsed.Responses {
				// assert the put call above acurately stored the expected test data.
				getResults := doMockGet(t, router, putResponse.UUID)
				if !assert.Equal(t, http.StatusOK, getResults.Code, "%s - %s: Get() failed with status: %d", group.groupDesc, tc.desc, getResults.Code) {
					return
				}
				if !assert.Equal(t, tc.expectedStoredValue, getResults.Body.String(), "%s - %s: Put() call didn't store the expected value", group.groupDesc, tc.desc) {
					return
				}
				if getResults.Header().Get("Content-Type") != group.expectedResponseType {
					t.Fatalf("%s - %s: Expected GET response Content-Type %v to equal %v", group.groupDesc, tc.desc, getResults.Header().Get("Content-Type"), group.expectedResponseType)
				}

				// assert the put call above logged expected metrics
				metricstest.AssertMetrics(t, tc.expectedMetrics, mockMetrics)
			}

		}
	}
}

// TestMalformedOrInvalidValue asserts the *PuntHandler.handle() function successfully responds
// with a http.StatusBadRequest given malformed or missing `value` field
func TestMalformedOrInvalidValue(t *testing.T) {
	testCases := []struct {
		desc             string
		inPutBody        string
		expectedError    error
		expectedPutCalls int
	}{
		{
			desc:             "Badly escaped character in value field",
			inPutBody:        `{"puts":[{"type":"json","value":"badly-esca"ped"}]}`,
			expectedError:    utils.NewPBCError(utils.PUT_BAD_REQUEST, `{"puts":[{"type":"json","value":"badly-esca"ped"}]}`),
			expectedPutCalls: 0,
		},
		{
			desc:             "Malformed JSON in value field",
			inPutBody:        `{"puts":[{"type":"json","value":malformed}]}`,
			expectedError:    utils.NewPBCError(utils.PUT_BAD_REQUEST, `{"puts":[{"type":"json","value":malformed}]}`),
			expectedPutCalls: 0,
		},
		{
			desc:             "Missing value field in sole element of puts array",
			inPutBody:        `{"puts":[{"type":"json","unrecognized":true}]}`,
			expectedError:    utils.NewPBCError(utils.MISSING_VALUE),
			expectedPutCalls: 0,
		},
		{
			desc:             "Missing value field in at least one element of puts array",
			inPutBody:        `{"puts":[{"type":"json","value":true}, {"type":"json","unrecognized":true}]}`,
			expectedError:    utils.NewPBCError(utils.MISSING_VALUE),
			expectedPutCalls: 1,
		},
		{
			desc:             "Invalid XML",
			inPutBody:        `{"puts":[{"type":"xml","value":5}]}`,
			expectedError:    utils.NewPBCError(utils.MALFORMED_XML, "XML messages must have a String value. Found [53]"),
			expectedPutCalls: 0,
		},
	}

	for _, tc := range testCases {
		router := httprouter.New()

		backend := &mockBackend{}
		backend.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}

		router.POST("/cache", NewPutHandler(backend, m, 10, true))

		// Run test
		putResponse := doPut(t, router, tc.inPutBody)

		// Assert expected response
		assert.Equal(t, http.StatusBadRequest, putResponse.Code, "%s: Put() call expected 400 response. Got: %d, Msg: %v", tc.desc, putResponse.Code, putResponse.Body.String())
		assert.Equal(t, tc.expectedError.Error()+"\n", putResponse.Body.String(), "%s: Put() return error doesn't match expected.", tc.desc)

		backend.AssertNumberOfCalls(t, "Put", tc.expectedPutCalls)

		// assert the put call above logged expected metrics
		expectedMetrics := []string{
			"RecordPutTotal",
			"RecordPutBadRequest",
		}
		metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
	}
}

// TestNonSupportedType asserts the *PuntHandler.handle() function successfully responds
// with a http.StatusBadRequest code if the value under the incomming request's `type` field
// refers to an unsupported data type.
func TestNonSupportedType(t *testing.T) {
	requestBody := `{"puts":[{"type":"yaml","value":"<tag></tag>"}]}`

	backend := &mockBackend{}
	backend.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	router := httprouter.New()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	putResponse := doPut(t, router, requestBody)

	require.Equal(t, http.StatusBadRequest, putResponse.Code, "Expected 400 response. Got: %d, Msg: %v", putResponse.Code, putResponse.Body.String())
	require.Equal(t, "Type must be one of [\"json\", \"xml\"]. Found 'yaml'\n", putResponse.Body.String(), "Put() return error doesn't match expected.")

	backend.AssertNotCalled(t, "Put")

	expectedMetrics := []string{
		"RecordPutTotal",
		"RecordPutBadRequest",
	}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
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
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}

	testRouter.POST("/cache", NewPutHandler(testBackend, m, 10, true))

	recorder := httptest.NewRecorder()

	// Run test
	testRouter.ServeHTTP(recorder, inRequest)

	// Assertions
	assert.Equal(t, expectedErrorMsg, recorder.Body.String(), "Put should have failed because we passed a negative ttlseconds value.\n")
	assert.Equalf(t, expectedStatusCode, recorder.Code, "Expected 400 response. Got: %d", recorder.Code)

	expectedMetrics := []string{
		"RecordPutTotal",
		"RecordPutBadRequest",
	}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

// TestCustomKey will assert the correct behavior when we try to store values that come with their own custom keys both
// when `cfg.allowKeys` is set to `true` and `false`. It will use two custom keys, one that is already holding data in our
// backend storage (36-char-key-maps-to-actual-xml-value) and one that doesn't (cust-key-maps-to-no-value-in-backend).
func TestCustomKey(t *testing.T) {
	type aTest struct {
		desc            string
		inCustomKey     string
		expectedUUID    string
		expectedMetrics []string
	}
	testGroups := []struct {
		allowSettingKeys bool
		testCases        []aTest
	}{
		{
			allowSettingKeys: false,
			testCases: []aTest{
				{
					desc:         "Custom key exists in cache but, because allowKeys is set to false we store the value using a random UUID and respond 200",
					inCustomKey:  "36-char-key-maps-to-actual-xml-value",
					expectedUUID: `[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`,
					expectedMetrics: []string{
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
				{
					desc:         "Custom key doesn't exist in cache but we can't store data under it because allowKeys is set to false. Store value with random UUID and respond 200",
					inCustomKey:  "cust-key-maps-to-no-value-in-backend",
					expectedUUID: `[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`,
					expectedMetrics: []string{
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
			},
		},
		{
			allowSettingKeys: true,
			testCases: []aTest{
				{
					desc:         "Setting keys allowed but key already maps to an element in cache, don't overwrite the value in the data storage and simply respond with blank UUID and a 200 code",
					inCustomKey:  "36-char-key-maps-to-actual-xml-value",
					expectedUUID: "",
					expectedMetrics: []string{
						"RecordPutTotal",
						"RecordPutDuration",
						"RecordPutKeyProvided",
					},
				},
				{
					desc:         "Custom key maps to no element in cache, store value using custom key and respond with a 200 code and the custom UUID",
					inCustomKey:  "cust-key-maps-to-no-value-in-backend",
					expectedUUID: "cust-key-maps-to-no-value-in-backend",
					expectedMetrics: []string{
						"RecordPutKeyProvided",
						"RecordPutTotal",
						"RecordPutDuration",
					},
				},
			},
		},
	}

	for _, tgroup := range testGroups {
		for _, tc := range tgroup.testCases {
			// Instantiate prebid cache prod server with mock metrics and a mock metrics that
			// already contains some values
			mockBackendWithValues := newMockBackend()
			mockMetrics := metricstest.CreateMockMetrics()
			m := &metrics.Metrics{
				MetricEngines: []metrics.CacheMetrics{
					&mockMetrics,
				},
			}

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

			// assert the put call above logged expected metrics
			metricstest.AssertMetrics(t, tc.expectedMetrics, mockMetrics)

			// Assert response UUID
			if tc.expectedUUID == "" {
				assert.Equalf(t, `{"responses":[{"uuid":""}]}`, recorder.Body.String(), tc.desc)
			} else {
				assert.Regexp(t, regexp.MustCompile(tc.expectedUUID), recorder.Body.String(), tc.desc)
			}
		}
	}
}

func TestRequestReadError(t *testing.T) {
	// Setup server and mock body request reader
	mockBackendWithValues := newMockBackend()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
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

	// assert the put call above logged expected metrics
	expectedMetrics := []string{"RecordPutTotal", "RecordPutBadRequest"}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestTooManyPutElements(t *testing.T) {
	// Test case: request with more than elements than put handler's max number of values
	putElements := []string{
		`{"type":"json","value":true}`,
		`{"type":"xml","value":"plain text"}`,
		`{"type":"xml","value":"2"}`,
	}
	reqBody := fmt.Sprintf(`{"puts":[%s, %s, %s]}`, putElements[0], putElements[1], putElements[2])

	//Set up server with capacity to handle less than putElements.size()
	backend := &mockBackend{}
	backend.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	router := httprouter.New()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	router.POST("/cache", NewPutHandler(backend, m, len(putElements)-1, true))

	putResponse := doPut(t, router, reqBody)

	assert.Equalf(t, http.StatusBadRequest, putResponse.Code, "doPut should have failed when trying to store %d elements because capacity is %d ", len(putElements), len(putElements)-1)
	assert.Equal(t, "More keys than allowed: 2\n", putResponse.Body.String(), "Put() return error doesn't match expected.")

	backend.AssertNotCalled(t, "Put")

	// assert the put call above logged expected metrics
	expectedMetrics := []string{"RecordPutTotal", "RecordPutBadRequest"}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

// TestMultiPutRequest asserts results for requests with more than one element in the "puts" array
func TestMultiPutRequest(t *testing.T) {
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

	// Set up sample multi put request
	reqBody := fmt.Sprintf("{\"puts\":[%s, %s, %s]}", testCases[0].elemToPut, testCases[1].elemToPut, testCases[2].elemToPut)

	request, err := http.NewRequest("POST", "/cache", strings.NewReader(reqBody))
	assert.NoError(t, err, "Failed to create a POST request: %v", err)

	// Set up server and run
	router := httprouter.New()
	backend := backends.NewMemoryBackend()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}

	router.POST("/cache", NewPutHandler(backend, m, 10, true))
	router.GET("/cache", NewGetHandler(backend, m, true))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, request)

	// Validate results
	//   Assert put metrics
	// assert the put call above logged expected metrics
	expectedMetrics := []string{"RecordPutTotal", "RecordPutDuration"}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)

	//   Assert put request
	var parsed PutResponse
	err = json.Unmarshal([]byte(rr.Body.String()), &parsed)
	assert.NoError(t, err, "Response from POST doesn't conform to the expected format: %s", rr.Body.String())

	//   call Get() on the UUIDs to assert they were correctly put
	for i, resp := range parsed.Responses {
		// Get value for this UUID. It is supposed to have been stored
		getResult := doMockGet(t, router, resp.UUID)

		// Assertions
		assert.Equalf(t, http.StatusOK, getResult.Code, "Description: %s \n Multi-element put failed to store:%s \n", testCases[i].description, testCases[i].elemToPut)
		assert.Equalf(t, testCases[i].expectedStoredValue, getResult.Body.String(), "GET response error. Expected %v. Actual %v", testCases[i].expectedStoredValue, getResult.Body.String())
	}

	//   Assert get metrics
	expectedMetrics = append(expectedMetrics, "RecordGetTotal", "RecordGetDuration")
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
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
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	putResponse := doPut(t, router, reqBody)

	// Assert expected response
	assert.Equal(t, http.StatusBadRequest, putResponse.Code, "doPut should have failed when trying to store elements in sizeCappedBackend")
	assert.Equal(t, "POST /cache element 0 exceeded max size: Payload size 30 exceeded max 3\n", putResponse.Body.String(), "Put() return error doesn't match expected.")

	//   metrics
	expectedMetrics := []string{"RecordPutTotal", "RecordPutBadRequest"}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestInternalPutClientError(t *testing.T) {
	// Input
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"some data\"}]}"
	// Expected metrics
	expectedMetrics := []string{
		"RecordPutTotal",
		"RecordPutError",
		"RecordPutBackendXml",
		"RecordPutBackendError",
		"RecordPutBackendSize",
		"RecordPutBackendTTLSeconds",
	}

	// Init test objects:
	router := httprouter.New()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	// Use mock client that will return an error
	backendWithMetrics := decorators.LogMetrics(newErrorReturningBackend(), m)

	router.POST("/cache", NewPutHandler(backendWithMetrics, m, 10, true))

	// Run test
	putResponse := doPut(t, router, reqBody)

	// Assert expected response
	assert.Equal(t, http.StatusInternalServerError, putResponse.Code, "Put should have failed because we are using an MockReturnErrorBackend")
	assert.Equal(t, "This is a mock backend that returns this error on Put() operation\n", putResponse.Body.String(), "Put() return error doesn't match expected.")
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestEmptyPutRequests(t *testing.T) {
	type testOutput struct {
		jsonResponse            string
		statusCode              int
		badRequestMetricsLogged bool
		durationMetricsLogged   bool
	}
	type aTest struct {
		desc     string
		reqBody  string
		expected testOutput
	}
	testCases := []aTest{
		{
			desc:    "No value in put element",
			reqBody: `{"puts":[{"type":"xml"}]}`,
			expected: testOutput{
				jsonResponse:            `Missing value`,
				statusCode:              http.StatusBadRequest,
				badRequestMetricsLogged: true,
			},
		},
		{
			desc:    "Blank value in put element",
			reqBody: `{"puts":[{"type":"xml","value":""}]}`,
			expected: testOutput{
				jsonResponse:          `{"responses":\[\{"uuid":"[a-z0-9-]+"\}\]}`,
				statusCode:            http.StatusOK,
				durationMetricsLogged: true,
			},
		},
		// This test is meant to come right after the "Blank value in put element" test in order to assert the correction
		// of a bug in the pre-PR#64 version of `endpoints/put.go`
		{
			desc:    "All empty body. ",
			reqBody: "{}",
			expected: testOutput{
				jsonResponse:          `{"responses":\[\]}`,
				statusCode:            http.StatusOK,
				durationMetricsLogged: true,
			},
		},
		{
			desc:    "Empty puts arrray",
			reqBody: "{\"puts\":[]}",
			expected: testOutput{
				jsonResponse:          `{"responses":\[\]}`,
				statusCode:            http.StatusOK,
				durationMetricsLogged: true,
			},
		},
	}

	for i, tc := range testCases {
		// Set up server
		backend := backends.NewMemoryBackend()
		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}
		router := httprouter.New()
		router.POST("/cache", NewPutHandler(backend, m, 10, true))
		rr := httptest.NewRecorder()

		// Create request everytime
		request, err := http.NewRequest("POST", "/cache", strings.NewReader(tc.reqBody))
		assert.NoError(t, err, "[%d] Failed to create a POST request: %v", i, err)

		// Run
		router.ServeHTTP(rr, request)
		assert.Equal(t, tc.expected.statusCode, rr.Code, "[%d] ServeHTTP(rr, request) failed = %v - %s", i, rr.Result())

		// Assert expected JSON response
		if !assert.Regexp(t, regexp.MustCompile(tc.expected.jsonResponse), rr.Body.String(), "[%d] Response body differs from expected - %s", i, tc.desc) {
			return
		}

		// Assert metrics
		expectedMetrics := []string{"RecordPutTotal"}
		if tc.expected.badRequestMetricsLogged {
			expectedMetrics = append(expectedMetrics, "RecordPutBadRequest")
		}
		if tc.expected.durationMetricsLogged {
			expectedMetrics = append(expectedMetrics, "RecordPutDuration")
		}
		metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
	}
}

func TestPutClientDeadlineExceeded(t *testing.T) {
	// Valid request
	reqBody := "{\"puts\":[{\"type\":\"xml\",\"value\":\"some data\"}]}"

	// Use mock client that will return an error
	backend := newDeadlineExceededBackend()

	// Run client
	router := httprouter.New()
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	router.POST("/cache", NewPutHandler(backend, m, 10, true))

	putResponse := doPut(t, router, reqBody)

	// Assert expected response
	assert.Equal(t, utils.HTTPDependencyTimeout, putResponse.Code, "Put should have failed because we are using a MockDeadlineExceededBackend")
	assert.Equal(t, "timeout writing value to the backend.\n", putResponse.Body.String(), "Put() return error doesn't match expected.")

	// Assert this request is accounted under the "puts.current_url.request.error" metrics
	expectedMetrics := []string{
		"RecordPutTotal",
		"RecordPutError",
	}
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

// TestParseRequest asserts *PutHandler's parseRequest(r *http.Request) method
func TestParseRequest(t *testing.T) {
	type testOut struct {
		put *putRequest
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
			testOut{nil, utils.NewPBCError(utils.PUT_BAD_REQUEST)},
		},
		{
			"request with malformed body throws unmarshal error",
			func() *http.Request {
				r, _ := http.NewRequest("POST", "http://fakeurl.com", bytes.NewBuffer([]byte(`malformed`)))
				return r
			},
			testOut{nil, utils.NewPBCError(utils.PUT_BAD_REQUEST, "malformed")},
		},
		{
			"valid request body. Expect no error",
			func() *http.Request {
				requestBody := []byte(`{"puts":[{"type":"json","value":{"valueField":5}}]}`)
				r, _ := http.NewRequest("POST", "http://fakeurl.com", bytes.NewBuffer(requestBody))
				return r
			},
			testOut{
				&putRequest{
					Puts: []putObject{
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
			testOut{nil, utils.NewPBCError(utils.PUT_MAX_NUM_VALUES, "More keys than allowed: 1")},
		},
	}
	for _, tc := range testCases {
		// set test
		putHandler := &PutHandler{
			memory: syncPools{
				requestPool: sync.Pool{
					New: func() interface{} { return &putRequest{} },
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
		in       putObject
		expected testOut
	}{
		{
			"empty value, expect error",
			putObject{},
			testOut{
				value: "",
				err:   utils.NewPBCError(utils.MISSING_VALUE),
			},
		},
		{
			"negative time-to-live, expect error",
			putObject{
				TTLSeconds: -1,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			testOut{
				value: "",
				err:   utils.NewPBCError(utils.NEGATIVE_TTL, "ttlseconds must not be negative -1."),
			},
		},
		{
			"non xml nor json type, expect error",
			putObject{
				Type:       "unknown",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>Your XML content goes here.</tag>`),
			},
			testOut{
				value: "",
				err:   utils.NewPBCError(utils.UNSUPPORTED_DATA_TO_STORE, "Type must be one of [\"json\", \"xml\"]. Found 'unknown'"),
			},
		},
		{
			"xml type value is not a string, expect error",
			putObject{
				Type:       "xml",
				TTLSeconds: 60,
				Value:      json.RawMessage(`<tag>XML</tag>`),
			},
			testOut{
				value: "",
				err:   utils.NewPBCError(utils.MALFORMED_XML, "XML messages must have a String value. Found [60 116 97 103 62 88 77 76 60 47 116 97 103 62]"),
			},
		},
		{
			"xml type value is surrounded by quotes and, therefore, a string. No errors expected",
			putObject{
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
			putObject{
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
func TestClassifyBackendError(t *testing.T) {
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
				utils.NewPBCError(utils.BAD_PAYLOAD_SIZE, "POST /cache element 0 exceeded max size: Payload size 2 exceeded max 1"),
				http.StatusBadRequest,
			},
		},
		{
			"DeadlineExceeded error",
			context.DeadlineExceeded,
			testOutput{
				utils.NewPBCError(utils.PUT_DEADLINE_EXCEEDED),
				utils.HTTPDependencyTimeout,
			},
		},
		{
			"Backend client error",
			errors.New("Server memory error"),
			testOutput{
				utils.NewPBCError(utils.PUT_INTERNAL_SERVER, "Server memory error"),
				http.StatusInternalServerError,
			},
		},
	}
	for _, tc := range testCases {
		// run
		err := classifyBackendError(tc.inError, 0)

		// assert error type:
		assert.Equal(t, tc.expected.err, err, tc.desc)

		// assert error code:
		assert.Equal(t, tc.expected.code, err.(utils.PBCError).StatusCode, tc.desc)
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

func doPut(t *testing.T, router *httprouter.Router, content string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()

	request, err := http.NewRequest("POST", "/cache", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to create a POST request: %v", err)
	}
	router.ServeHTTP(rr, request)

	return rr
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
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}

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

	backend.Put(context.Background(), "non-36-char-key-maps-to-json", `json{"field":"value"}`, 0)
	backend.Put(context.Background(), "36-char-key-maps-to-non-xml-nor-json", `#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`, 0)
	backend.Put(context.Background(), "36-char-key-maps-to-actual-xml-value", "xml<tag>xml data here</tag>", 0)

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

func newErrorReturningBackend() *errorReturningBackend {
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

func newDeadlineExceededBackend() *deadlineExceedingBackend {
	return &deadlineExceedingBackend{}
}

type mockBackend struct {
	mock.Mock
}

func (m *mockBackend) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockBackend) Put(ctx context.Context, key, value string, ttlSeconds int) error {
	args := m.Called(ctx, key, value, ttlSeconds)
	return args.Error(0)
}
