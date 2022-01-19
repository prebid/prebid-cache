package utils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPBCError(t *testing.T) {
	type testInput struct {
		errType int
		msgs    string
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected PBCError
	}{
		{
			desc: "Valid error type maps to a constant error message",
			in: testInput{
				errType: MISSING_KEY,
			},
			expected: PBCError{
				Type:       MISSING_KEY,
				StatusCode: http.StatusBadRequest,
				msg:        "missing required parameter uuid",
			},
		},
		{
			desc: "Valid error type doesn't map to a constant error message but no error message was passed to NewPBCError, expect blank error message.",
			in: testInput{
				errType: PUT_MAX_NUM_VALUES,
			},
			expected: PBCError{
				Type:       PUT_MAX_NUM_VALUES,
				StatusCode: http.StatusBadRequest,
			},
		},
		{
			desc: "Valid error type doesn't map to a constant error message, custom error message passed as parameter",
			in: testInput{
				errType: PUT_MAX_NUM_VALUES,
				msgs:    "Some error message",
			},
			expected: PBCError{
				Type:       PUT_MAX_NUM_VALUES,
				StatusCode: http.StatusBadRequest,
				msg:        "Some error message",
			},
		},
		{
			desc: "Unknown error type no 'msgs' param was passed",
			in: testInput{
				errType: 100,
			},
			expected: PBCError{
				Type:       100,
				StatusCode: http.StatusInternalServerError,
			},
		},
		{
			desc: "Unknown error type. 'msgs' param was passed",
			in: testInput{
				errType: 100,
				msgs:    "Some error message",
			},
			expected: PBCError{
				Type:       100,
				StatusCode: http.StatusInternalServerError,
				msg:        "Some error message",
			},
		},
	}
	for _, tc := range testCases {
		// set test
		var pbcError PBCError

		// run
		if len(tc.in.msgs) > 0 {
			pbcError = NewPBCError(tc.in.errType, tc.in.msgs)
		} else {
			pbcError = NewPBCError(tc.in.errType)
		}

		// assertions
		assert.Equal(t, tc.expected.Type, pbcError.Type, tc.desc)
		assert.Equal(t, tc.expected.StatusCode, pbcError.StatusCode, tc.desc)
		assert.Equal(t, tc.expected.Error(), pbcError.Error(), tc.desc)
	}
}
