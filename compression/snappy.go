package compression

import (
	"context"

	"github.com/golang/snappy"
	"github.com/prebid/prebid-cache/backends"
)

// SnappyCompress runs snappy compression on data before saving it in the backend.
// For more info, see https://en.wikipedia.org/wiki/Snappy_(compression)
func SnappyCompress(backend backends.Backend) backends.Backend {
	return &snappyCompressor{
		delegate: backend,
	}
}

type snappyCompressor struct {
	delegate backends.Backend
}

func (s *snappyCompressor) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return s.delegate.Put(ctx, key, string(snappy.Encode(nil, []byte(value))), ttlSeconds)
}

func (s *snappyCompressor) Get(ctx context.Context, key string) (string, error) {
	compressed, err := s.delegate.Get(ctx, key)
	if err != nil {
		return "", err
	}

	decompressed, err := snappy.Decode(nil, []byte(compressed))
	if err != nil {
		return "", err
	}

	return string(decompressed), nil
}
