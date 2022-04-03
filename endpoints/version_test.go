package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionEndpoint(t *testing.T) {
	var testCases = []struct {
		description string
		version     string
		expected    string
	}{
		{
			description: "Empty",
			version:     "",
			expected:    `{"version":"not-set"}`,
		},
		{
			description: "Version Only",
			version:     "1.2.3",
			expected:    `{"version":"1.2.3"}`,
		},
		{
			description: "Revision Only",
			version:     "",
			expected:    `{"version":"not-set"}`,
		},
		{
			description: "Fully Populated",
			version:     "1.2.3",
			expected:    `{"version":"1.2.3"}`,
		},
	}

	for _, test := range testCases {
		handler := NewVersionEndpoint(test.version)
		assert.HTTPBodyContains(t, handler, "GET", "/version", nil, test.expected, "Error on version endpoint response. Test: %s", test.description)
	}
}
