package backends

//// errorProneHttpClient is an http.Client mock that returns an error and is used for testing purposes
//type httpDoFailClient struct {
//	IgniteClient
//}
//
//func (ig *httpDoFailClient) HttpDo(req *http.Request) (*http.Response, error) {
//	return nil, errors.New("Fake ignite client throws error in its HttpDo implementation")
//}
//
//func (ig *httpDoFailClient) CreateCache(url *url.URL, cacheName string) error {
//	return ig.IgniteClient.CreateCache(url, cacheName)
//}
//
//func (ig *httpDoFailClient) Get(req *http.Request) (string, error) {
//	return ig.IgniteClient.Get(req)
//}
//
//func (ig *httpDoFailClient) Put(req *http.Request) (bool, error) {
//	return ig.Put(req)
//}

//func TestCreateCache(t *testing.T) {
//	type testInput struct {
//		//handlerFunc http.Handler
//		serverInit func() *httptest.Server
//		url        *url.URL
//		cacheName  string
//	}
//	testCases := []struct {
//		desc        string
//		in          testInput
//		expectError bool
//	}{
//		{
//			desc: "Invalid URL. Expect http.NewRequestWithContext() error",
//			in: testInput{
//				url: &url.URL{Scheme: ":invalid:"},
//				//handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				//	w.WriteHeader(http.StatusOK)
//				//}),
//				serverInit: func() *httptest.Server {
//					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//						w.WriteHeader(http.StatusOK)
//					})
//					return httptest.NewServer(handler)
//				},
//			},
//			expectError: true,
//		},
//		{
//			desc: "Fake client mocks server-side error",
//			in: testInput{
//				//handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				//	w.WriteHeader(http.StatusBadRequest)
//				//	w.Write([]byte(`Server-side error`))
//				//}),
//				serverInit: func() *httptest.Server {
//					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//						w.WriteHeader(http.StatusBadRequest)
//						w.Write([]byte(`Server-side error`))
//					})
//					return httptest.NewServer(handler)
//				},
//			},
//			expectError: true,
//		},
//		{
//			desc: "Fake client returns empty body",
//			in: testInput{
//				//handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				//	w.WriteHeader(http.StatusOK)
//				//}),
//				serverInit: func() *httptest.Server {
//					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//						w.WriteHeader(http.StatusOK)
//					})
//					return httptest.NewServer(handler)
//				},
//			},
//			expectError: true,
//		},
//	}
//
//	for _, tc := range testCases {
//		//fakeServer := httptest.NewServer(tc.in.handlerFunc)
//		//fakeServer := tc.in.serverInit()
//		fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(200)
//			w.Write([]byte("some body"))
//		}))
//		igniteClient := &IgniteClient{
//			client: fakeServer.Client(),
//			//client: tc.in.serverInit().Client(),
//		}
//
//		url := &url.URL{
//			Scheme: "http",
//			Host:   "127.0.0.1:8080",
//			Path:   "/ignite",
//		}
//		if tc.in.url != nil {
//			url = tc.in.url
//		}
//
//		err := igniteClient.CreateCache(url, tc.in.cacheName)
//		if tc.expectError {
//			assert.Error(t, err, tc.desc)
//			assert.Equal(t, "SomeError msg", err.Error(), tc.desc)
//		} else {
//			assert.Nil(t, err, tc.desc)
//		}
//
//		if fakeServer != nil {
//			fakeServer.Close()
//		}
//	}
//}

//func TestNewIgniteBackend(t *testing.T) {
//	type logEntry struct {
//		msg string
//		lvl logrus.Level
//	}
//
//	type testOut struct {
//		backend      *IgniteBackend
//		panicHappens bool
//		logEntries   []logEntry
//	}
//
//	type testCase struct {
//		desc     string
//		in       config.Ignite
//		expected testOut
//	}
//	testGroups := []struct {
//		desc      string
//		testCases []testCase
//	}{
//		{
//			desc: "config validation error",
//			testCases: []testCase{
//				{
//					desc: "empty scheme",
//					in:   config.Ignite{},
//					expected: testOut{
//						backend:      nil,
//						panicHappens: true,
//						logEntries: []logEntry{
//							{
//								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
//								lvl: logrus.FatalLevel,
//							},
//						},
//					},
//				},
//				{
//					desc: "empty host",
//					in: config.Ignite{
//						Scheme: "http",
//					},
//					expected: testOut{
//						backend:      nil,
//						panicHappens: true,
//						logEntries: []logEntry{
//							{
//								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
//								lvl: logrus.FatalLevel,
//							},
//						},
//					},
//				},
//				{
//					desc: "empty port",
//					in: config.Ignite{
//						Scheme: "http",
//						Host:   "127.0.0.1",
//						Port:   0,
//					},
//					expected: testOut{
//						backend:      nil,
//						panicHappens: true,
//						logEntries: []logEntry{
//							{
//								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
//								lvl: logrus.FatalLevel,
//							},
//						},
//					},
//				},
//				{
//					desc: "No cache name",
//					in: config.Ignite{
//						Scheme: "http",
//						Host:   "127.0.0.1",
//						Port:   8080,
//						Cache:  config.IgniteCache{},
//					},
//					expected: testOut{
//						backend:      nil,
//						panicHappens: true,
//						logEntries: []logEntry{
//							{
//								msg: "Error creating Ignite backend: configuration is missing ignite.schema, ignite.host, ignite.port or ignite.cache.name",
//								lvl: logrus.FatalLevel,
//							},
//						},
//					},
//				},
//			},
//		},
//		{
//			desc: "parse URL error",
//			testCases: []testCase{
//				{
//					desc: "Non-empty scheme holds an invalid value",
//					in: config.Ignite{
//						Scheme: ":invalid:",
//						Host:   "127.0.0.1",
//						Port:   8080,
//						Cache: config.IgniteCache{
//							Name: "myCache",
//						},
//					},
//					expected: testOut{
//						backend:      nil,
//						panicHappens: true,
//						logEntries: []logEntry{
//							{
//								msg: "Error creating Ignite backend: error parsing Ignite host URL parse \":invalid:://127.0.0.1:8080/ignite?cacheName=myCache\": missing protocol scheme",
//								lvl: logrus.FatalLevel,
//							},
//						},
//					},
//				},
//			},
//		},
//		{
//			desc: "Non error",
//			testCases: []testCase{
//				{
//					desc: "Expect validation to pass and a default client with secure http transport",
//					in: config.Ignite{
//						Scheme: "http",
//						Host:   "127.0.0.1",
//						Port:   8080,
//						Secure: true,
//						Cache: config.IgniteCache{
//							Name:          "myCache",
//							CreateOnStart: false,
//						},
//					},
//					expected: testOut{
//						backend: &IgniteBackend{
//							serverURL: &url.URL{
//								Scheme:   "http",
//								Host:     "127.0.0.1:8080",
//								Path:     "/ignite",
//								RawQuery: "cacheName=myCache",
//							},
//							cacheName: "myCache",
//							client:    &IgniteClient{client: http.DefaultClient},
//						},
//						panicHappens: false,
//						logEntries: []logEntry{
//							{
//								msg: "Prebid Cache will write to Ignite cache name: myCache",
//								lvl: logrus.InfoLevel,
//							},
//						},
//					},
//				},
//				{
//					desc: "Expect validation to pass but with Secure is set to false. Expect client with insecure http transport",
//					in: config.Ignite{
//						Scheme: "http",
//						Host:   "127.0.0.1",
//						Port:   8080,
//						Secure: false,
//						Cache: config.IgniteCache{
//							Name:          "myCache",
//							CreateOnStart: false,
//						},
//					},
//					expected: testOut{
//						backend: &IgniteBackend{
//							serverURL: &url.URL{
//								Scheme:   "http",
//								Host:     "127.0.0.1:8080",
//								Path:     "/ignite",
//								RawQuery: "cacheName=myCache",
//							},
//							cacheName: "myCache",
//							client: &IgniteClient{
//								client: &http.Client{
//									Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
//								},
//							},
//						},
//						panicHappens: false,
//						logEntries: []logEntry{
//							{
//								msg: "Prebid Cache will write to Ignite cache name: myCache",
//								lvl: logrus.InfoLevel,
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//
//	// logrus entries will be recorded to this `hook` object so we can compare and assert them
//	hook := test.NewGlobal()
//
//	//substitute logger exit function so execution doesn't get interrupted when log.Fatalf() call comes
//	defer func() { logrus.StandardLogger().ExitFunc = nil }()
//	logrus.StandardLogger().ExitFunc = func(int) {}
//
//	for _, group := range testGroups {
//		for _, tc := range group.testCases {
//
//			var resultingBackend *IgniteBackend
//			if tc.expected.panicHappens {
//				assert.Panics(t, func() { resultingBackend = NewIgniteBackend(tc.in) }, "NewIgniteBackend() should have panicked and it didn't happen")
//			} else {
//				resultingBackend = NewIgniteBackend(tc.in)
//			}
//			if assert.Len(t, hook.Entries, len(tc.expected.logEntries), "%s - %s", group.desc, tc.desc) {
//				for i := 0; i < len(tc.expected.logEntries); i++ {
//					assert.Equalf(t, tc.expected.logEntries[i].msg, hook.Entries[i].Message, "%s - %s", group.desc, tc.desc)
//					assert.Equalf(t, tc.expected.logEntries[i].lvl, hook.Entries[i].Level, "%s - %s", group.desc, tc.desc)
//				}
//			}
//
//			assert.Equalf(t, tc.expected.backend, resultingBackend, "%s - %s", group.desc, tc.desc)
//
//			//Reset log after every test and assert successful reset
//			hook.Reset()
//			assert.Nil(t, hook.LastEntry())
//		}
//	}
//}
