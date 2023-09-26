package backends

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestDoRequest(t *testing.T) {
	type testInput struct {
		ctx      context.Context
		headers  http.Header
		httpResp *http.Response
		httpErr  error
	}

	type testOutput struct {
		resp []byte
		err  error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "http.NewRequestWithContext returns error because nil Context",
			expected: testOutput{
				resp: nil,
				err:  errors.New("net/http: nil Context"),
			},
		},
		{
			desc: "http.Client.Do() returns an error",
			in: testInput{
				ctx:      context.TODO(),
				httpResp: nil,
				httpErr: &url.Error{
					Op:  "GET",
					Err: errors.New("fake http.Client error"),
				},
			},
			expected: testOutput{
				resp: nil,
				err:  &url.Error{Op: "GET", Err: errors.New("fake http.Client error")},
			},
		},
		{
			desc: "nil http.Response.Body is returned",
			in: testInput{
				ctx: context.TODO(),
				httpResp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       nil,
				},
				httpErr: nil,
			},
			expected: testOutput{
				resp: nil,
				err:  errors.New("Ignite error. Received empty httpResp.Body"),
			},
		},
		{
			desc: "Non 200 status code is returned in the http.Response",
			in: testInput{
				ctx: context.TODO(),
				httpResp: &http.Response{
					StatusCode: http.StatusNotFound,
					Body: fakeReadCloser{
						body: []byte{0x0},
						err:  io.EOF,
					},
				},
				httpErr: nil,
			},
			expected: testOutput{
				resp: []byte{0x0},
				err:  errors.New("Ignite error. Unexpected status code: 404"),
			},
		},
		{
			desc: "http.Response.Body read error",
			in: testInput{
				ctx: context.TODO(),
				httpResp: &http.Response{
					StatusCode: http.StatusOK,
					Body: fakeReadCloser{
						body: []byte{0x0},
						err:  io.ErrShortBuffer,
					},
				},
				httpErr: nil,
			},
			expected: testOutput{
				resp: nil,
				err:  errors.New("Ignite error. IO reader error: short buffer"),
			},
		},
		{
			desc: "Success",
			in: testInput{
				ctx: context.TODO(),
				headers: http.Header{
					"HEADER": []string{"value"},
				},
				httpResp: &http.Response{
					StatusCode: http.StatusOK,
					Body: fakeReadCloser{
						body: []byte(`{"jsonObject":"value"}`),
						err:  io.EOF,
					},
				},
				httpErr: nil,
			},
			expected: testOutput{
				resp: []byte(`{"jsonObject":"value"}`),
				err:  nil,
			},
		},
	}
	for _, tc := range testCases {
		fakeIgniteClient := &igniteSender{
			httpClient: &fakeHttpClient{
				mockFunction: func() (*http.Response, error) {
					return tc.in.httpResp, tc.in.httpErr
				},
			},
		}
		actualResp, actualErr := fakeIgniteClient.DoRequest(tc.in.ctx, &url.URL{}, tc.in.headers)

		assert.Equal(t, tc.expected.resp, actualResp, tc.desc)
		assert.Equal(t, tc.expected.err, actualErr, tc.desc)
	}
}

func TestNewIgniteBackend(t *testing.T) {
	type logEntry struct {
		msg string
		lvl logrus.Level
	}

	type testOut struct {
		backend      *IgniteBackend
		panicHappens bool
		logEntries   []logEntry
	}

	type testCase struct {
		desc     string
		in       config.Ignite
		expected testOut
	}
	testGroups := []struct {
		desc      string
		testCases []testCase
	}{
		{
			desc: "config validation error",
			testCases: []testCase{
				{
					desc: "empty scheme",
					in:   config.Ignite{},
					expected: testOut{
						backend:      nil,
						panicHappens: true,
						logEntries: []logEntry{
							{
								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
								lvl: logrus.FatalLevel,
							},
						},
					},
				},
				{
					desc: "empty host",
					in: config.Ignite{
						Scheme: "http",
					},
					expected: testOut{
						backend:      nil,
						panicHappens: true,
						logEntries: []logEntry{
							{
								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
								lvl: logrus.FatalLevel,
							},
						},
					},
				},
				{
					desc: "empty port",
					in: config.Ignite{
						Scheme: "http",
						Host:   "127.0.0.1",
						Port:   0,
					},
					expected: testOut{
						backend:      nil,
						panicHappens: true,
						logEntries: []logEntry{
							{
								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
								lvl: logrus.FatalLevel,
							},
						},
					},
				},
				{
					desc: "No cache name",
					in: config.Ignite{
						Scheme: "http",
						Host:   "127.0.0.1",
						Port:   8080,
						Cache:  config.IgniteCache{},
					},
					expected: testOut{
						backend:      nil,
						panicHappens: true,
						logEntries: []logEntry{
							{
								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
								lvl: logrus.FatalLevel,
							},
						},
					},
				},
			},
		},
		{
			desc: "parse URL error",
			testCases: []testCase{
				{
					desc: "Non-empty scheme holds an invalid value",
					in: config.Ignite{
						Scheme: ":invalid:",
						Host:   "127.0.0.1",
						Port:   8080,
						Cache: config.IgniteCache{
							Name: "myCache",
						},
					},
					expected: testOut{
						backend:      nil,
						panicHappens: true,
						logEntries: []logEntry{
							{
								msg: "Error creating Ignite backend: error parsing Ignite host URL parse \":invalid:://127.0.0.1:8080/ignite\": missing protocol scheme",
								lvl: logrus.FatalLevel,
							},
						},
					},
				},
			},
		},
		{
			desc: "Non error",
			testCases: []testCase{
				{
					desc: "Expect validation to pass and a default client with secure http transport",
					in: config.Ignite{
						Scheme: "http",
						Host:   "127.0.0.1",
						Port:   8080,
						Secure: true,
						//Headers: http.Header{
						//	"HEADER": []string{"value"},
						//},
						Cache: config.IgniteCache{
							Name:          "myCache",
							CreateOnStart: false,
						},
					},
					expected: testOut{
						backend: &IgniteBackend{
							serverURL: &url.URL{
								Scheme: "http",
								Host:   "127.0.0.1:8080",
								Path:   "/ignite",
							},
							sender: &igniteSender{httpClient: http.DefaultClient},
						},
						panicHappens: false,
						logEntries: []logEntry{
							{
								msg: "Prebid Cache will write to Ignite cache name: myCache",
								lvl: logrus.InfoLevel,
							},
						},
					},
				},
				{
					desc: "Expect validation to pass but with Secure is set to false. Expect client with insecure http transport",
					in: config.Ignite{
						Scheme: "http",
						Host:   "127.0.0.1",
						Port:   8080,
						Secure: false,
						Cache: config.IgniteCache{
							Name:          "myCache",
							CreateOnStart: false,
						},
					},
					expected: testOut{
						backend: &IgniteBackend{
							serverURL: &url.URL{
								Scheme: "http",
								Host:   "127.0.0.1:8080",
								Path:   "/ignite",
							},
							sender: &igniteSender{
								httpClient: &http.Client{
									Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
								},
							},
						},
						panicHappens: false,
						logEntries: []logEntry{
							{
								msg: "Prebid Cache will write to Ignite cache name: myCache",
								lvl: logrus.InfoLevel,
							},
						},
					},
				},
			},
		},
	}

	// logrus entries will be recorded to this `hook` object so we can compare and assert them
	hook := test.NewGlobal()

	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	logrus.StandardLogger().ExitFunc = func(int) {}

	for _, group := range testGroups {
		for _, tc := range group.testCases {

			var resultingBackend *IgniteBackend
			if tc.expected.panicHappens {
				assert.Panics(t, func() { resultingBackend = NewIgniteBackend(tc.in) }, "NewIgniteBackend() should have panicked and it didn't happen")
			} else {
				resultingBackend = NewIgniteBackend(tc.in)
			}
			if assert.Len(t, hook.Entries, len(tc.expected.logEntries), "%s - %s", group.desc, tc.desc) {
				for i := 0; i < len(tc.expected.logEntries); i++ {
					assert.Equalf(t, tc.expected.logEntries[i].msg, hook.Entries[i].Message, "%s - %s", group.desc, tc.desc)
					assert.Equalf(t, tc.expected.logEntries[i].lvl, hook.Entries[i].Level, "%s - %s", group.desc, tc.desc)
				}
			}

			assert.Equalf(t, tc.expected.backend, resultingBackend, "%s - %s", group.desc, tc.desc)

			//Reset log after every test and assert successful reset
			hook.Reset()
			assert.Nil(t, hook.LastEntry())
		}
	}
}

type fakeIgniteClient struct {
	respond func() ([]byte, error)
}

func (c *fakeIgniteClient) DoRequest(ctx context.Context, url *url.URL, headers http.Header) ([]byte, error) {
	return c.respond()
}

func TestIgniteGet(t *testing.T) {
	type testInput struct {
		igniteResponse []byte
		igniteError    error
	}

	type testOutput struct {
		value string
		err   error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "DoRequest call fails, expect error",
			in: testInput{
				igniteResponse: nil,
				igniteError:    errors.New("Mock Ignite Client DoRequest() error"),
			},
			expected: testOutput{
				err: errors.New("Mock Ignite Client DoRequest() error"),
			},
		},
		{
			desc: "DoRequest call returns malformed JSON blob",
			in: testInput{
				igniteResponse: []byte(`malformed`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: errors.New("Unmarshal response error: invalid character 'm' looking for beginning of value; Response body: malformed"),
			},
		},
		{
			desc: "Ignite server responds with error message",
			in: testInput{
				igniteResponse: []byte(`{"error":"Server side error"}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.GET_INTERNAL_SERVER, "Server side error"),
			},
		},
		{
			desc: "Ignite server responds with non-zero 'successStatus' value",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":1,"error":""}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.GET_INTERNAL_SERVER, "Ignite response.Status not zero"),
			},
		},
		{
			desc: "Ignite responds with 'successStatus' equal to zero and empty 'error' message, but also an empty 'response' field",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":0,"error":"","response":""}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.KEY_NOT_FOUND),
			},
		},
	}

	for _, tc := range testCases {
		back := &IgniteBackend{
			sender: &fakeIgniteClient{
				respond: func() ([]byte, error) {
					return tc.in.igniteResponse, tc.in.igniteError
				},
			},
			serverURL: &url.URL{},
		}

		v, err := back.Get(nil, "someKey")

		assert.Equal(t, tc.expected.value, v, tc.desc)
		assert.Equal(t, tc.expected.err, err, tc.desc)
	}
}

func TestIgnitePut(t *testing.T) {
	type testInput struct {
		igniteResponse []byte
		igniteError    error
	}

	type testOutput struct {
		err error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "DoRequest call fails, expect error",
			in: testInput{
				igniteResponse: nil,
				igniteError:    errors.New("Mock Ignite Client DoRequest() error"),
			},
			expected: testOutput{
				err: errors.New("Mock Ignite Client DoRequest() error"),
			},
		},
		{
			desc: "DoRequest call returns malformed JSON blob",
			in: testInput{
				igniteResponse: []byte(`malformed`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: errors.New("Unmarshal response error: invalid character 'm' looking for beginning of value; Response body: malformed"),
			},
		},
		{
			desc: "Ignite server responds with error message",
			in: testInput{
				igniteResponse: []byte(`{"error":"Server side error"}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.PUT_INTERNAL_SERVER, "Server side error"),
			},
		},
		{
			desc: "Ignite server responds with non-zero 'successStatus' value",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":1,"error":""}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.PUT_INTERNAL_SERVER, "Ignite responded with non-zero successStatus code"),
			},
		},
		{
			desc: "Ignite responds 'response' field set to false",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":0,"error":"","response":false}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: utils.NewPBCError(utils.RECORD_EXISTS),
			},
		},
		{
			desc: "Successful ignite put",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":0,"error":"","response":true}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		back := &IgniteBackend{
			sender: &fakeIgniteClient{
				respond: func() ([]byte, error) {
					return tc.in.igniteResponse, tc.in.igniteError
				},
			},
			serverURL: &url.URL{},
		}

		err := back.Put(nil, "someKey", "someValue", 5)

		assert.Equal(t, tc.expected.err, err, tc.desc)
	}
}

func TestCreateCache(t *testing.T) {
	type testInput struct {
		igniteResponse []byte
		igniteError    error
	}

	type testOutput struct {
		err error
	}

	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "DoRequest call fails, expect error",
			in: testInput{
				igniteResponse: nil,
				igniteError:    errors.New("Mock Ignite Client DoRequest() error"),
			},
			expected: testOutput{
				err: errors.New("Mock Ignite Client DoRequest() error"),
			},
		},
		{
			desc: "DoRequest call returns malformed JSON blob",
			in: testInput{
				igniteResponse: []byte(`malformed`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: errors.New("Unmarshal response error: invalid character 'm' looking for beginning of value; Response body: malformed"),
			},
		},
		{
			desc: "Ignite server responds with error message",
			in: testInput{
				igniteResponse: []byte(`{"error":"Server side error"}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: errors.New("Ignite error. Server side error"),
			},
		},
		{
			desc: "Ignite server responds with non-zero 'successStatus' value",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":1,"error":""}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: errors.New(`Ignite error. successStatus does not equal 0 {  1}`),
			},
		},
		{
			desc: "Successfully created cache",
			in: testInput{
				igniteResponse: []byte(`{"successStatus":0,"error":""}`),
				igniteError:    nil,
			},
			expected: testOutput{
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		back := &IgniteBackend{
			sender: &fakeIgniteClient{
				respond: func() ([]byte, error) {
					return tc.in.igniteResponse, tc.in.igniteError
				},
			},
			serverURL: &url.URL{},
		}

		assert.Equal(t, tc.expected.err, createCache(back), tc.desc)
	}
}
