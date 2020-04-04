package backends

import (
	"context"
	"fmt"
	"time"
)

type MockReturnErrorBackend struct{}

func (b *MockReturnErrorBackend) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("This is a mock backend that returns this error on Get() operation")
}

func (b *MockReturnErrorBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return fmt.Errorf("This is a mock backend that returns this error on Put() operation")
}

func NewErrorReturningBackend() *MockReturnErrorBackend {
	return &MockReturnErrorBackend{}
}

type MockDeadlineExceededBackend struct{}

func (b *MockDeadlineExceededBackend) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (b *MockDeadlineExceededBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	var err error

	d := time.Now().Add(50 * time.Millisecond)
	sampleCtx, cancel := context.WithDeadline(context.Background(), d)

	// Even though ctx will be expired, it is good practice to call its
	// cancellation function in any case. Failure to do so may keep the
	// context and its parent alive longer than necessary.
	defer cancel()

	select {
	case <-time.After(1 * time.Second):
		//err = fmt.Errorf("Some other error")
		err = nil
	case <-sampleCtx.Done():
		err = sampleCtx.Err()
	}
	return err
}

func NewDeadlineExceededBackend() *MockDeadlineExceededBackend {
	return &MockDeadlineExceededBackend{}
}
