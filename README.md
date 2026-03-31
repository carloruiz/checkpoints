# checkpoints

A Go library for key/value checkpoints backed by pluggable SQL stores. Supports PostgreSQL and CockroachDB.

## Install

```bash
go get github.com/carloruiz/checkpoints
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/carloruiz/checkpoints"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Progress struct {
	Step    int    `json:"step"`
	Status  string `json:"status"`
}

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://localhost:5432/mydb")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Create the checkpoints table (idempotent).
	if err := checkpoints.CreateTable(ctx, pool); err != nil {
		log.Fatal(err)
	}

	store := checkpoints.NewPGXStore(pool)

	// Save a checkpoint.
	err = store.Set(ctx, "pipeline/step-3", Progress{Step: 3, Status: "done"})
	if err != nil {
		log.Fatal(err)
	}

	// Load it back.
	var p Progress
	found, err := store.Get(ctx, "pipeline/step-3", &p)
	if err != nil {
		log.Fatal(err)
	}
	if !found {
		fmt.Println("no checkpoint found")
		return
	}
	fmt.Printf("step=%d status=%s\n", p.Step, p.Status)
	// Output: step=3 status=done
}
```

## Testing

Start PostgreSQL and CockroachDB:

```bash
docker compose up -d
```

Run all tests:

```bash
CHECKPOINT_TEST_POSTGRES_DSN="postgres://test:test@localhost:5433/checkpoints_test?sslmode=disable" \
CHECKPOINT_TEST_CRDB_DSN="postgres://root@localhost:26257/defaultdb?sslmode=disable" \
go test -v ./...
```

Omit either variable to skip that engine. Without both, integration tests are skipped and only unit tests run.
