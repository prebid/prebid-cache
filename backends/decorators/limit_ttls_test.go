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
					desc:         "reqTTL < maxTTL. Given that both hold negative values, set to REQUEST_MAX_TTL_SECONDS constant",
					inRequestTTL: -2,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "reqTTL = maxTTL. Given that both hold negative values, set to REQUEST_MAX_TTL_SECONDS constant",
					inRequestTTL: -1,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "maxTTL < reqTTL. Go with the non-negative, non-zero request ttl",
					inRequestTTL: 10,
					expectedTTL:  10,
				},
			},
		},
		{
			groupDesc: "maxTTL is zero",
			maxTTL:    0,
			testCases: []testCase{
				{
					desc:         "reqTTL < maxTTL. In the absence of a positive ttl value, set to REQUEST_MAX_TTL_SECONDS constant",
					inRequestTTL: -1,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "reqTTL = maxTTL. In the absence of a positive ttl value, set to REQUEST_MAX_TTL_SECONDS constant",
					inRequestTTL: 0,
					expectedTTL:  utils.REQUEST_MAX_TTL_SECONDS,
				},
				{
					desc:         "maxTTL < reqTTL. Go with the non-negative, non-zero request ttl",
					inRequestTTL: 10,
					expectedTTL:  10,
				},
			},
		},
		{
			groupDesc: "maxTTL is non-negative nor zero",
			maxTTL:    10,
			testCases: []testCase{
				{
					desc:         "reqTTL < 0 < maxTTL. Given that the request ttl is negative, set to maxTTL",
					inRequestTTL: -1,
					expectedTTL:  10,
				},
				{
					desc:         "reqTTL equals zero. Given that the request ttl equals zero, set to maxTTL",
					inRequestTTL: 0,
					expectedTTL:  10,
				},
				{
					desc:         "0 < reqTTL < maxTTL. Set to request ttl because its value is between zero and the maxTTL",
					inRequestTTL: 5,
					expectedTTL:  5,
				},
				{
					desc:         "reqTTL equals maxTTL; set to request ttl because its value does not surpases maxTTL",
					inRequestTTL: 10,
					expectedTTL:  10,
				},
				{
					desc:         "0 < maxTTL < reqTTL; set to maxTTL because request ttl goes past maxTTL",
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
