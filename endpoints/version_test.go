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
