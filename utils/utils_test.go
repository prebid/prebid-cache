package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPick(t *testing.T) {
	testCases := []struct {
		desc              string
		inPickProbability float32
		expected          bool
	}{
		{
			desc:              "Zero logging rate. Expect false",
			inPickProbability: 0.00,
			expected:          false,
		},
		{
			desc:              "100% logging rate, expect true",
			inPickProbability: 1.00,
			expected:          true,
		},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.expected, RandomPick(tc.inPickProbability), tc.desc)
	}
}
