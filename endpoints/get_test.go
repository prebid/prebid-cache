package endpoints

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendConfig "github.com/prebid/prebid-cache/backends/config"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	testLogrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestGetJsonTests(t *testing.T) {
	testGroups := []struct {
		desc  string
		tests []string
	}{
		// GET tests:
		// - successful
		{
			desc: "Sucessful",
			tests: []string{
				"sample-requests/get-endpoint/valid/element-found.json",
			},
		},
		// - element is not in the backend (key not found)
		// - Request missing UUID
		// - UUID invalid somehow
		//{
		//	desc: "Expect error",
		//	tests: []string{
		//		"sample-requests/get-endpoint/invalid/missing-uuid.json",
		//		"sample-requests/get-endpoint/invalid/key-not-found.json",
		//		"sample-requests/get-endpoint/invalid/uuid-length.json",
		//		"sample-requests/get-endpoint/invalid/data-corrupted.json",
		//	},
		//},
	}

	// Log entries
	hook := testLogrus.NewGlobal()
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, group := range testGroups {
		for _, testFile := range group.tests {
			tc, err := parseTestInfo(testFile)
			if !assert.NoError(t, err, "%v", err) {
				hook.Reset()
				continue
			}

			// Setup
			v := buildViperConfig(tc)
			cfg := config.Configuration{}
			err = v.Unmarshal(&cfg)
			if !assert.NoError(t, err, "Viper could not parse configuration from test file: %s. Error:%s\n", testFile, err) {
				hook.Reset()
				continue
			}

			mockMetrics := metricstest.CreateMockMetrics()
			m := &metrics.Metrics{
				MetricEngines: []metrics.CacheMetrics{
					&mockMetrics,
				},
			}

			var backend backends.Backend
			if len(tc.ServerConfig.StoredData) > 0 {
				backend, err = newMemoryBackendWithValues(tc.ServerConfig.StoredData)
				if !assert.NoError(t, err, "Failed to create Mock backend for test: %s Error: %v", testFile, err) {
					hook.Reset()
					continue
				}
				backend = backendConfig.DecorateBackend(cfg, m, backend)
			} else {
				backend = backendConfig.NewBackend(cfg, m)
			}

			router := httprouter.New()
			router.GET("/cache", NewGetHandler(backend, m, tc.ServerConfig.AllowSettingKeys))

			// Run test
			getResults := httptest.NewRecorder()
			getReq, err := http.NewRequest("GET", "/cache?"+tc.Query, nil)
			if !assert.NoError(t, err, "Failed to create a GET request: %v", err) {
				hook.Reset()
				continue
			}
			router.ServeHTTP(getResults, getReq)

			// Assertions
			assert.Equal(t, tc.ExpectedResponse.Code, getResults.Code, testFile)

			// Assert this is a valid test that expects either an error or a GetResponse
			if !assert.NotEqual(t, len(tc.ExpectedResponse.ErrorMsg) > 0, len(tc.ExpectedResponse.GetOutput) > 0, "%s must come with either an expected error message or an expected response", testFile) {
				hook.Reset()
				continue
			}

			// If error is expected, assert error message with the response body
			if len(tc.ExpectedResponse.ErrorMsg) > 0 {
				assert.Equal(t, tc.ExpectedResponse.ErrorMsg, getResults.Body.String(), testFile)
				hook.Reset()
				assert.Nil(t, hook.LastEntry())
				continue
			}

			if len(tc.ExpectedResponse.GetOutput) > 0 {
				//out := ""
				//err := json.Unmarshal(tc.ExpectedResponse.GetOutput, &out)
				//assert.NoError(t, err, "Test file GetOutput could not be unmarshaled: %s. Error:%s\n", testFile, err)

				assert.Equal(t, tc.ExpectedResponse.GetOutput, getResults.Body.String(), testFile)
				//assert.Equal(t, out, getResults.Body.String(), testFile)
				hook.Reset()
				assert.Nil(t, hook.LastEntry())
				continue
			}

			// Assert logrus expected entries
			assertLogEntries(t, tc.ExpectedLogEntries, hook.Entries, testFile)

			metricstest.AssertMetrics(t, tc.ExpectedMetrics, mockMetrics)

			// Reset log
			hook.Reset()
		}
	}
}

func TestGetInvalidUUIDs(t *testing.T) {
	backend := backends.NewMemoryBackend()
	router := httprouter.New()

	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}

	router.GET("/cache", NewGetHandler(backend, m, false))

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

func TestGetHandler(t *testing.T) {

	//preExistentDataInBackend := []putObject{
	//	{Key: "non-36-char-key-maps-to-json", Value: json.RawMessage(`json{"field":"value"}`), TTLSeconds: 0},
	//	{Key: "36-char-key-maps-to-non-xml-nor-json", Value: json.RawMessage(`#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`), TTLSeconds: 0},
	//	{Key: "36-char-key-maps-to-actual-xml-value", Value: json.RawMessage("xml<tag>xml data here</tag>"), TTLSeconds: 0},
	//}
	preExistentDataInBackend := []storedData{
		{Key: "non-36-char-key-maps-to-json", Value: `json{"field":"value"}`},
		{Key: "36-char-key-maps-to-non-xml-nor-json", Value: `#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`},
		{Key: "36-char-key-maps-to-actual-xml-value", Value: "xml<tag>xml data here</tag>"},
	}

	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	type testInput struct {
		uuid      string
		allowKeys bool
	}
	type testOutput struct {
		responseCode    int
		responseBody    string
		logEntries      []logEntry
		expectedMetrics []string
	}

	testCases := []struct {
		desc string
		in   testInput
		out  testOutput
	}{
		{
			"Missing UUID. Return http error but don't interrupt server's execution",
			testInput{
				uuid:      "",
				allowKeys: false,
			},
			testOutput{
				responseCode: http.StatusBadRequest,
				responseBody: "GET /cache: Missing required parameter uuid\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache: Missing required parameter uuid",
						lvl: logrus.ErrorLevel,
					},
				},
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetBadRequest",
				},
			},
		},
		{
			"Prebid Cache wasn't configured to allow custom keys therefore, it doesn't allow for keys different than 36 char long. Respond with http error and don't interrupt server's execution",
			testInput{
				uuid:      "non-36-char-key-maps-to-json",
				allowKeys: false,
			},
			testOutput{
				responseCode: http.StatusNotFound,
				responseBody: "GET /cache uuid=non-36-char-key-maps-to-json: invalid uuid length\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache uuid=non-36-char-key-maps-to-json: invalid uuid length",
						lvl: logrus.ErrorLevel,
					},
				},
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetBadRequest",
				},
			},
		},
		{
			"Configuration that allows custom keys. These are not required to be 36 char long. Since the uuid maps to a value, return it along a 200 status code",
			testInput{
				uuid:      "non-36-char-key-maps-to-json",
				allowKeys: true,
			},
			testOutput{
				responseCode: http.StatusOK,
				responseBody: `{"field":"value"}`,
				logEntries:   []logEntry{},
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetDuration",
				},
			},
		},
		{
			"Valid 36 char long UUID not found in database. Return http error but don't interrupt server's execution",
			testInput{uuid: "uuid-not-found-and-links-to-no-value"},
			testOutput{
				responseCode: http.StatusNotFound,
				responseBody: "GET /cache uuid=uuid-not-found-and-links-to-no-value: Key not found\n",
				logEntries: []logEntry{
					{
						msg: "GET /cache uuid=uuid-not-found-and-links-to-no-value: Key not found",
						lvl: logrus.DebugLevel,
					},
				},
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetBadRequest",
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
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetError",
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
				expectedMetrics: []string{
					"RecordGetTotal",
					"RecordGetDuration",
				},
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
		backend, err := newMemoryBackendWithValues(preExistentDataInBackend)
		if !assert.NoError(t, err, "%s. Mock backend could not be created", test.desc) {
			continue
			hook.Reset()
		}
		router := httprouter.New()
		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}
		router.GET("/cache", NewGetHandler(backend, m, test.in.allowKeys))

		// Run test
		getResults := httptest.NewRecorder()

		body := new(bytes.Buffer)
		getReq, err := http.NewRequest("GET", "/cache"+"?uuid="+test.in.uuid, body)
		if !assert.NoError(t, err, "Failed to create a GET request: %v", err) {
			continue
			hook.Reset()
		}
		router.ServeHTTP(getResults, getReq)
		//  //getResults := doMockGet(t, router, test.in.uuid)

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

		// Assert recorded metrics
		metricstest.AssertMetrics(t, test.out.expectedMetrics, mockMetrics)

		// Reset log
		hook.Reset()
	}
}
