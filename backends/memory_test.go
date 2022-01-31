package backends

import (
	"context"
	"testing"

	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestMemoryBackend(t *testing.T) {
	type testExpectedValues struct {
		value string
		err   error
	}

	type aTest struct {
		desc     string
		backend  *MemoryBackend
		setup    func(b *MemoryBackend)
		run      func(b *MemoryBackend) (string, error)
		expected testExpectedValues
	}

	testGroups := []struct {
		desc      string
		testCases []aTest
	}{
		{
			"Put tests",
			[]aTest{
				{
					desc:    "succesful put",
					backend: NewMemoryBackend(),
					setup:   func(b *MemoryBackend) {},
					run: func(b *MemoryBackend) (string, error) {
						err := b.Put(context.TODO(), "someKey", "someValye", 0)
						return "", err
					},
					expected: testExpectedValues{err: nil},
				},
				{
					desc:    "Put returns a RecordExistsError",
					backend: NewMemoryBackend(),
					setup: func(b *MemoryBackend) {
						b.Put(context.TODO(), "someKey", "someValue", 0)
					},
					run: func(b *MemoryBackend) (string, error) {
						err := b.Put(context.TODO(), "someKey", "someValye", 0)
						return "", err
					},
					expected: testExpectedValues{"", utils.NewPBCError(utils.RECORD_EXISTS)},
				},
			},
		},
		{
			"Get tests",
			[]aTest{
				{
					desc:    "succesful get",
					backend: NewMemoryBackend(),
					setup: func(b *MemoryBackend) {
						b.Put(context.TODO(), "someKey", "someValue", 0)
					},
					run: func(b *MemoryBackend) (string, error) {
						return b.Get(context.TODO(), "someKey")
					},
					expected: testExpectedValues{"someValue", nil},
				},
				{
					desc:    "Get returns a Key not found error",
					backend: NewMemoryBackend(),
					setup: func(b *MemoryBackend) {
						b.Put(context.TODO(), "someKey", "someValue", 0)
					},
					run: func(b *MemoryBackend) (string, error) {
						return b.Get(context.TODO(), "anotherKey")
					},
					expected: testExpectedValues{"", utils.NewPBCError(utils.KEY_NOT_FOUND)},
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// Setup
			tc.setup(tc.backend)

			//Run
			resultingValue, resultingError := tc.run(tc.backend)

			//Assert
			assert.Equal(t, tc.expected.value, resultingValue, "%s - %s", group.desc, tc.desc)
			assert.Equal(t, tc.expected.err, resultingError, "%s - %s", group.desc, tc.desc)
		}
	}
}
