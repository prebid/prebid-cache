package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionEndpoint(t *testing.T) {
	var testCases = []struct {
		description string
		version     string
		revision    string
		expected    string
	}{
		{
			description: "Empty",
			version:     "",
			expected:    `{"version":"not-set","revision":"not-set"}`,
		},
		{
			description: "Version Only",
			version:     "1.2.3",
			expected:    `{"version":"1.2.3","revision":"not-set"}`,
		},
		{
			description: "Revision Only",
			version:     "",
			revision:    "sha",
			expected:    `{"version":"not-set","revision":"sha"}`,
		},
		{
			description: "Fully Populated",
			version:     "1.2.3",
			revision:    "sha",
			expected:    `{"version":"1.2.3","revision":"sha"}`,
		},
	}

	for _, test := range testCases {
		handler := NewVersionEndpoint(test.version, test.revision)
		assert.HTTPBodyContains(t, handler, "GET", "/version", nil, test.expected, "Error on version endpoint response. Test: %s", test.description)
	}
}
