package checkpoints

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the database interface satisfied by *pgxpool.Pool, *pgx.Conn, and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewPGXStore creates a Store backed by PostgreSQL or CockroachDB.
func NewPGXStore(db DBTX) Store {
	return newStore(&pgxBackend{db: db})
}

type pgxBackend struct {
	db DBTX
}

func (p *pgxBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	err := p.db.QueryRow(ctx, `SELECT value FROM checkpoints WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (p *pgxBackend) Set(ctx context.Context, key string, value []byte) error {
	_, err := p.db.Exec(ctx,
		`INSERT INTO checkpoints (key, value, updated_at)
		VALUES ($1, $2::jsonb, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		key, string(value),
	)
	return err
}
