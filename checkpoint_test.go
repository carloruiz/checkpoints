package checkpoints

import (
	"context"
	"strings"
	"testing"
)

// memBackend is an in-memory backend for unit tests.
type memBackend struct {
	data map[string][]byte
}

func newMemBackend() *memBackend {
	return &memBackend{data: make(map[string][]byte)}
}

func (m *memBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *memBackend) Set(_ context.Context, key string, value []byte) error {
	m.data[key] = append([]byte(nil), value...)
	return nil
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"valid", "my-key", nil},
		{"with slashes", "workflow/step-3", nil},
		{"max length", strings.Repeat("a", MaxKeyLength), nil},
		{"too long", strings.Repeat("a", MaxKeyLength+1), ErrKeyTooLong},
		{"empty", "", ErrKeyEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateKey(tt.key); err != tt.wantErr {
				t.Errorf("validateKey(%q) = %v, want %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestSetGet(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	type checkpoint struct {
		Step   int    `json:"step"`
		Status string `json:"status"`
	}

	in := checkpoint{Step: 3, Status: "done"}
	if err := s.Set(ctx, "workflow/step-3", in); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var out checkpoint
	found, err := s.Get(ctx, "workflow/step-3", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if out != in {
		t.Errorf("Get = %+v, want %+v", out, in)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	var dest map[string]any
	found, err := s.Get(ctx, "missing", &dest)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected found=false for missing key")
	}
}

func TestUpsert(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	if err := s.Set(ctx, "k", map[string]int{"v": 1}); err != nil {
		t.Fatalf("Set v1: %v", err)
	}
	if err := s.Set(ctx, "k", map[string]int{"v": 2}); err != nil {
		t.Fatalf("Set v2: %v", err)
	}

	var out map[string]int
	found, err := s.Get(ctx, "k", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if out["v"] != 2 {
		t.Errorf("v = %d, want 2", out["v"])
	}
}

func TestSetMarshalError(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	if err := s.Set(ctx, "k", make(chan int)); err == nil {
		t.Fatal("expected marshal error for chan type")
	}
}

func TestSetKeyValidation(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	if err := s.Set(ctx, "", "val"); err == nil {
		t.Error("expected error for empty key")
	}
	if err := s.Set(ctx, strings.Repeat("x", MaxKeyLength+1), "val"); err == nil {
		t.Error("expected error for oversized key")
	}
}

func TestGetKeyValidation(t *testing.T) {
	s := newStore(newMemBackend())
	ctx := context.Background()

	if _, err := s.Get(ctx, "", new(any)); err == nil {
		t.Error("expected error for empty key")
	}
	if _, err := s.Get(ctx, strings.Repeat("x", MaxKeyLength+1), new(any)); err == nil {
		t.Error("expected error for oversized key")
	}
}
