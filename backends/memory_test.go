package backends

import (
	"context"
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestMemoryBackend(t *testing.T) {
	type testExpectedValues struct {
		value string
		err   error
	}

	testCases := []struct {
		desc     string
		backend  *MemoryBackend
		setup    func(b *MemoryBackend)
		run      func(b *MemoryBackend) (string, error)
		expected testExpectedValues
	}{
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
			expected: testExpectedValues{"", utils.RecordExistsError{}},
		},
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
			expected: testExpectedValues{"", utils.KeyNotFoundError{}},
		},
	}

	for _, tc := range testCases {
		// Setup
		tc.setup(tc.backend)

		//Run
		resultingValue, resultingError := tc.run(tc.backend)

		//Assert
		assert.Equal(t, tc.expected.value, resultingValue, tc.desc)
		assert.Equal(t, tc.expected.err, resultingError, tc.desc)
	}
}
