package endpoints

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	testLogrus "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
)

func TestGetJsonTests(t *testing.T) {
	hook := testLogrus.NewGlobal()
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	jsonTests := listJsonFiles("sample-requests/get-endpoint")
	for _, testFile := range jsonTests {
		var backend backends.Backend
		mockMetrics := metricstest.CreateMockMetrics()
		tc, backend, m, err := setupJsonTest(&mockMetrics, backend, testFile)
		if !assert.NoError(t, err, "%s", testFile) {
			hook.Reset()
			continue
		}

		router := httprouter.New()
		router.GET("/cache", NewGetHandler(backend, m, tc.HostConfig.AllowSettingKeys, tc.HostConfig.RefererLogRate))
		request, err := http.NewRequest("GET", "/cache?"+tc.Query, nil)
		if !assert.NoError(t, err, "Failed to create a GET request: %v", err) {
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
			continue
		}
		rr := httptest.NewRecorder()

		// Run test
		router.ServeHTTP(rr, request)

		// Assertions
		assert.Equal(t, tc.ExpectedOutput.Code, rr.Code, testFile)

		// Assert this is a valid test that expects either an error or a GetResponse
		if !assert.False(t, len(tc.ExpectedOutput.ErrorMsg) > 0 && len(tc.ExpectedOutput.GetOutput) > 0, "%s must come with either an expected error message or an expected response", testFile) {
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
			continue
		}

		// If error is expected, assert error message with the response body
		if len(tc.ExpectedOutput.ErrorMsg) > 0 {
			assert.Equal(t, tc.ExpectedOutput.ErrorMsg, rr.Body.String(), testFile)
		} else {
			assert.Equal(t, tc.ExpectedOutput.GetOutput, rr.Body.String(), testFile)
		}

		assertLogEntries(t, tc.ExpectedLogEntries, hook.Entries, testFile)
		metricstest.AssertMetrics(t, tc.ExpectedMetrics, mockMetrics)

		// Reset log after every test and assert successful reset
		hook.Reset()
		assert.Nil(t, hook.LastEntry())
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

	router.GET("/cache", NewGetHandler(backend, m, false, 0.0))

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
	preExistentDataInBackend := map[string]string{
		"non-36-char-key-maps-to-json":         `json{"field":"value"}`,
		"36-char-key-maps-to-non-xml-nor-json": `#@!*{"desc":"data got malformed and is not prefixed with 'xml' nor 'json' substring"}`,
		"36-char-key-maps-to-actual-xml-value": "xml<tag>xml data here</tag>",
	}

	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	type testConfig struct {
		allowKeys           bool
		refererSamplingRate float64
	}
	type testInput struct {
		uuid       string
		cfg        testConfig
		reqHeaders map[string]string
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
				uuid: "",
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
				uuid: "non-36-char-key-maps-to-json",
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
				uuid: "non-36-char-key-maps-to-json",
				cfg:  testConfig{allowKeys: true},
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
		{
			"Sampling rate is set to 100% but request comes with no referer header. No logs expected.",
			testInput{
				uuid:       "36-char-key-maps-to-actual-xml-value",
				cfg:        testConfig{refererSamplingRate: 1.0},
				reqHeaders: map[string]string{"OtherHeader": "headervalue"},
			},
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
		{
			"Sampling rate is set to 100%. Expect request referer header to be logged.",
			testInput{
				uuid: "36-char-key-maps-to-actual-xml-value",
				cfg:  testConfig{refererSamplingRate: 1.0},
				reqHeaders: map[string]string{
					"Referer":     "anyreferer",
					"OtherHeader": "headervalue",
				},
			},
			testOutput{
				responseCode: http.StatusOK,
				responseBody: "<tag>xml data here</tag>",
				logEntries: []logEntry{
					{
						msg: "GET request Referer header: anyreferer",
						lvl: logrus.InfoLevel,
					},
				},
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
		backend, err := backends.NewMemoryBackendWithValues(preExistentDataInBackend)
		if !assert.NoError(t, err, "%s. Mock backend could not be created", test.desc) {
			hook.Reset()
			continue
		}
		router := httprouter.New()
		mockMetrics := metricstest.CreateMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}
		router.GET("/cache", NewGetHandler(backend, m, test.in.cfg.allowKeys, test.in.cfg.refererSamplingRate))

		// Run test
		getResults := httptest.NewRecorder()

		body := new(bytes.Buffer)
		getReq, err := http.NewRequest("GET", "/cache"+"?uuid="+test.in.uuid, body)

		if len(test.in.reqHeaders) > 0 {
			for k, v := range test.in.reqHeaders {
				getReq.Header.Set(k, v)
			}
		}

		if !assert.NoError(t, err, "Failed to create a GET request: %v", err) {
			hook.Reset()
			continue
		}
		router.ServeHTTP(getResults, getReq)

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
