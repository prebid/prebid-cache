package endpoints

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestGetInvalidUUIDs(t *testing.T) {
	backend := backends.NewMemoryBackend()
	router := httprouter.New()
	mockmetrics := metricstest.CreateMockMetrics()

	router.GET("/cache", NewGetHandler(backend, mockmetrics, false))

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
	type logEntry struct {
		msg string
		lvl logrus.Level
	}
	type testInput struct {
		uuid      string
		allowKeys bool
	}
	type metricsRecords struct {
		totalRequests int64
		badRequests   int64
		requestErrs   int64
		requestDur    float64
	}
	type testOutput struct {
		responseCode    int
		responseBody    string
		logEntries      []logEntry
		metricsRecorded metricsRecords
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					badRequests:   int64(1),
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					badRequests:   int64(1),
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					requestDur:    1.00,
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					badRequests:   int64(1),
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					requestErrs:   int64(1),
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
				metricsRecorded: metricsRecords{
					totalRequests: int64(1),
					requestDur:    1.00,
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
		backend := newMockBackend()
		router := httprouter.New()
		mockmetrics := metricstest.CreateMockMetrics()
		router.GET("/cache", NewGetHandler(backend, mockmetrics, test.in.allowKeys))

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

		// Assert recorded metrics
		assert.Equal(t, test.out.metricsRecorded.totalRequests, metricstest.MockCounters["gets.current_url.request.total"], "%s - handle function should record every incomming GET request", test.desc)
		assert.Equal(t, test.out.metricsRecorded.badRequests, metricstest.MockCounters["gets.current_url.request.bad_request"], "%s - Bad request wasn't recorded", test.desc)
		assert.Equal(t, test.out.metricsRecorded.requestErrs, metricstest.MockCounters["gets.current_url.request.error"], "%s - WriteGetResponse error should have been recorded", test.desc)
		assert.Equal(t, test.out.metricsRecorded.requestDur, metricstest.MockHistograms["gets.current_url.duration"], "%s - Successful GET request should have recorded duration", test.desc)

		// Reset log
		hook.Reset()
	}
}
