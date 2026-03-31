package checkpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// MaxKeyLength is the maximum allowed length of a checkpoint key in bytes.
const MaxKeyLength = 256

var (
	ErrKeyTooLong = errors.New("checkpoint key exceeds 256 bytes")
	ErrKeyEmpty   = errors.New("checkpoint key is empty")
)

// Store is the interface for reading and writing checkpoints.
type Store interface {
	Get(ctx context.Context, key string, dest any) (bool, error)
	Set(ctx context.Context, key string, value any) error
}

// backend is the internal storage interface. Implementations deal in raw
// JSON bytes; serialization and key validation happen in the Store layer.
type backend interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte) error
}

type store struct {
	b backend
}

func newStore(b backend) Store {
	return &store{b: b}
}

func validateKey(key string) error {
	if key == "" {
		return ErrKeyEmpty
	}
	if len(key) > MaxKeyLength {
		return ErrKeyTooLong
	}
	return nil
}

func (s *store) Get(ctx context.Context, key string, dest any) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}
	data, found, err := s.b.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("checkpoint get %q: %w", key, err)
	}
	if !found {
		return false, nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("checkpoint unmarshal %q: %w", key, err)
	}
	return true, nil
}

func (s *store) Set(ctx context.Context, key string, value any) error {
	if err := validateKey(key); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("checkpoint marshal %q: %w", key, err)
	}
	if err := s.b.Set(ctx, key, data); err != nil {
		return fmt.Errorf("checkpoint set %q: %w", key, err)
	}
	return nil
}
