package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPick(t *testing.T) {
	testCases := []struct {
		name              string
		inPickProbability float64
		expected          bool
	}{
		{
			name:              "zero", // Zero probablity of true, expect false
			inPickProbability: 0.00,
			expected:          false,
		},
		{
			name:              "one", // 100% probability of true, expect true
			inPickProbability: 1.00,
			expected:          true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, RandomPick(tc.inPickProbability), tc.name)
		})
	}
}
