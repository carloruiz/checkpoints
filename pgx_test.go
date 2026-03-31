package checkpoints

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests run against every configured database engine.
// Set one or both environment variables to enable:
//
//	CHECKPOINT_TEST_POSTGRES_DSN="postgres://test:test@localhost:5433/checkpoints_test?sslmode=disable"
//	CHECKPOINT_TEST_CRDB_DSN="postgres://root@localhost:26257/defaultdb?sslmode=disable"

var engines = []struct {
	name   string
	envVar string
}{
	{"postgres", "CHECKPOINT_TEST_POSTGRES_DSN"},
	{"cockroachdb", "CHECKPOINT_TEST_CRDB_DSN"},
}

// forEachDB runs fn as a subtest against every configured database.
func forEachDB(t *testing.T, fn func(t *testing.T, s Store)) {
	t.Helper()
	ran := false
	for _, e := range engines {
		dsn := os.Getenv(e.envVar)
		if dsn == "" {
			continue
		}
		ran = true
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()
			pool, err := pgxpool.New(ctx, dsn)
			if err != nil {
				t.Fatalf("pgxpool.New: %v", err)
			}
			t.Cleanup(pool.Close)

			if err := CreateTable(ctx, pool); err != nil {
				t.Fatalf("CreateTable: %v", err)
			}
			t.Cleanup(func() {
				pool.Exec(context.Background(), "DELETE FROM checkpoints")
			})
			fn(t, NewPGXStore(pool))
		})
	}
	if !ran {
		t.Skip("set CHECKPOINT_TEST_POSTGRES_DSN or CHECKPOINT_TEST_CRDB_DSN to run integration tests")
	}
}

func TestPGXSetGet(t *testing.T) {
	forEachDB(t, func(t *testing.T, s Store) {
		ctx := context.Background()

		type state struct {
			Count   int    `json:"count"`
			Message string `json:"message"`
		}

		in := state{Count: 42, Message: "hello"}
		if err := s.Set(ctx, "pgx-test", in); err != nil {
			t.Fatalf("Set: %v", err)
		}

		var out state
		found, err := s.Get(ctx, "pgx-test", &out)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if out != in {
			t.Errorf("Get = %+v, want %+v", out, in)
		}
	})
}

func TestPGXGetNotFound(t *testing.T) {
	forEachDB(t, func(t *testing.T, s Store) {
		ctx := context.Background()

		var dest map[string]any
		found, err := s.Get(ctx, "does-not-exist", &dest)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if found {
			t.Error("expected found=false")
		}
	})
}

func TestPGXUpsert(t *testing.T) {
	forEachDB(t, func(t *testing.T, s Store) {
		ctx := context.Background()

		if err := s.Set(ctx, "upsert-key", map[string]int{"v": 1}); err != nil {
			t.Fatalf("Set v1: %v", err)
		}
		if err := s.Set(ctx, "upsert-key", map[string]int{"v": 2}); err != nil {
			t.Fatalf("Set v2: %v", err)
		}

		var out map[string]int
		found, err := s.Get(ctx, "upsert-key", &out)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if out["v"] != 2 {
			t.Errorf("v = %d, want 2", out["v"])
		}
	})
}
