package endpoints

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionEndpoint(t *testing.T) {
	var testCases = []struct {
		description string
		revision    string
		version     string
		expected    string
	}{
		{
			description: "Empty",
			expected:    `{"revision":"not-set","version":"not-set"}`,
		},
		{
			description: "Revision Only",
			revision:    "sha",
			expected:    `{"revision":"sha","version":"not-set"}`,
		},
		{
			description: "Version Only",
			version:     "1.2.3",
			expected:    `{"revision":"not-set","version":"1.2.3"}`,
		},
		{
			description: "Fully Populated",
			revision:    "sha",
			version:     "1.2.3",
			expected:    `{"revision":"sha","version":"1.2.3"}`,
		},
	}

	for _, test := range testCases {
		handler := NewVersionEndpoint(test.version, test.revision)
		recorder := httptest.NewRecorder()

		// Run
		handler(recorder, nil, nil)

		// Assert
		response, err := ioutil.ReadAll(recorder.Result().Body)
		if assert.NoError(t, err, test.description) {
			assert.JSONEq(t, test.expected, string(response), test.description)
		}
	}
}
