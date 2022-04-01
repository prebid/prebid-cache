package decorators_test

import (
	"context"
	"testing"

	"github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestLimitTTLDecorator(t *testing.T) {
	type testCase struct {
		desc         string
		inRequestTTL int
		expectedTTL  int
	}
	testGroups := []struct {
		groupDesc string
		maxTTL    int
		testCases []testCase
	}{
		{
			groupDesc: "maxTTL is negative. Set to REQUEST_MAX_TTL_SECONDS constant in every scenario",
			maxTTL:    -1,
			testCases: []testCase{
				{
					desc:         "reqTTL < maxTTL",
					inRequestTTL: -2,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "reqTTL = maxTTL",
					inRequestTTL: -1,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "maxTTL < reqTTL",
					inRequestTTL: 10,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
			},
		},
		{
			groupDesc: "maxTTL is zero. Set to REQUEST_MAX_TTL_SECONDS constant in every scenario",
			maxTTL:    0,
			testCases: []testCase{
				{
					desc:         "reqTTL < maxTTL",
					inRequestTTL: -1,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "reqTTL = maxTTL",
					inRequestTTL: 0,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "maxTTL < reqTTL",
					inRequestTTL: 10,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
			},
		},
		{
			groupDesc: "maxTTL is non-negative nor zero",
			maxTTL:    10,
			testCases: []testCase{
				{
					desc:         "reqTTL < 0 < maxTTL; set to maxTTL",
					inRequestTTL: -1,
					expectedTTL:  10,
				},
				{
					desc:         "reqTTL equals zero. Set to non-zero maxTTL",
					inRequestTTL: 0,
					expectedTTL:  10,
				},
				{
					desc:         "0 < reqTTL < maxTTL; set to request maxTTL",
					inRequestTTL: 5,
					expectedTTL:  5,
				},
				{
					desc:         "reqTTL equals maxTTL; set to request maxTTL",
					inRequestTTL: 10,
					expectedTTL:  10,
				},
				{
					desc:         "0 < maxTTL < reqTTL; set to request maxTTL",
					inRequestTTL: 50,
					expectedTTL:  10,
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// set test
			delegate := &ttlCapturer{}
			wrapped := decorators.LimitTTLs(delegate, group.maxTTL)

			// run
			wrapped.Put(context.Background(), "key", "value", tc.inRequestTTL)

			// assertions
			assert.Equal(t, tc.expectedTTL, delegate.lastTTL, "%s - %s", group.groupDesc, tc.desc)
		}
	}
}

func TestExcessiveTTL(t *testing.T) {
	delegate := &ttlCapturer{}
	wrapped := decorators.LimitTTLs(delegate, 100)
	wrapped.Put(context.Background(), "foo", "bar", 200)
	if delegate.lastTTL != 100 {
		t.Errorf("lastTTL should be %d. Got %d", 100, delegate.lastTTL)
	}
}

func TestSafeTTL(t *testing.T) {
	delegate := &ttlCapturer{}
	wrapped := decorators.LimitTTLs(delegate, 100)
	wrapped.Put(context.Background(), "foo", "bar", 50)
	if delegate.lastTTL != 50 {
		t.Errorf("lastTTL should be %d. Got %d", 50, delegate.lastTTL)
	}
}

type ttlCapturer struct {
	lastTTL int
}

func (c *ttlCapturer) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	c.lastTTL = ttlSeconds
	return nil
}

func (c *ttlCapturer) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}
