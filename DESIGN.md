# Checkpoints Library — Design Doc

## Overview

A Go library for short-lived key/value checkpoints backed by a pluggable SQL store. Checkpoints are ephemeral (useful for days, not months) — schema evolution and backwards compatibility are non-goals.

## API

```go
package checkpoint

type Store interface {
    Get(ctx context.Context, key string, dest any) (bool, error)
    Set(ctx context.Context, key string, value any) error
}
```

- **Get**: Unmarshals the stored JSON into `dest` (must be a pointer). Returns `(false, nil)` for missing keys — map-access style.
- **Set**: Marshals `value` as JSON and upserts. Idempotent — last write wins.

## Key Design

- Type: `string`
- Max length: 256 bytes. Enforced at the library level and by the schema (`VARCHAR(256)`).
- Flat namespace. Callers can use conventions like `workflow/step` but the library treats keys as opaque strings.

## Value Serialization

JSON via `encoding/json`. The library owns marshal/unmarshal so callers pass Go types directly.

Stored as `JSONB` in the database for human-readable inspection and SQL queryability.

### Alternatives considered

| Option             | Pros                          | Cons                                  |
|--------------------|-------------------------------|---------------------------------------|
| `[]byte`           | Zero opinions, any format     | Caller boilerplate, opaque in DB      |
| `json.RawMessage`  | Valid JSON guarantee, no reflection | Same caller boilerplate as `[]byte` |
| **`any` (chosen)** | Clean caller ergonomics, readable in DB | Locks format to JSON, reflection cost |

`any` wins because debuggability and caller ergonomics matter more than format flexibility for short-lived checkpoints.

## Concurrency

Last-write-wins. No optimistic concurrency or compare-and-swap. The `ON CONFLICT` upsert is atomic at the row level; callers coordinate externally if they need stronger guarantees.

## SQL Schema

Compatible with PostgreSQL and CockroachDB:

```sql
CREATE TABLE IF NOT EXISTS checkpoints (
    key        VARCHAR(256) PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Storage Interface (internal)

The pluggable backend implements a thin internal interface. The public `Store` handles serialization and key validation, then delegates to the backend:

```go
type backend interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte) error
}
```

The backend deals in raw `[]byte` (already-marshaled JSON). This keeps the storage layer free of `encoding/json` concerns and makes it easy to add new backends (Redis, SQLite, etc.) without touching serialization logic.

## Error Handling

- Missing keys: `Get` returns `(false, nil)` — not an error.
- Serialization errors from `json.Marshal`/`json.Unmarshal` are returned directly.
- Database errors are wrapped and returned.

## Out of Scope

- TTL / automatic expiry (handle via external cron or `DELETE WHERE updated_at < ...`)
- Batch get/set
- Namespacing / multi-tenancy
- Encryption at rest
- Schema migration tooling
