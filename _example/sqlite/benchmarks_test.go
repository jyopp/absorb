package main

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/jyopp/absorb"
)

type BenchRow struct {
	ID    int64  `sqlite:"id"`
	Label string `sqlite:"value"`
}

// Creates a unique in-memory database pool per test name.
// Each DB is seeded with 10 test rows.
// The database pool is closed in the test's cleanup phase.
func attachInMemoryTestDB(b *testing.B) *sqlitex.Pool {
	uri := "file:" + b.Name() + "?mode=memory&cache=shared"
	// b.Logf("Opening %s\n", uri)
	pool, err := sqlitex.Open(uri, 0, 10)
	b.Cleanup(func() {
		if pool != nil {
			if err := pool.Close(); err != nil {
				b.Error(err)
			}
		}
	})

	if err == nil {
		conn := pool.Get(context.TODO())
		defer pool.Put(conn)

		// Create a test table and insert some values.
		err = sqlitex.ExecScript(conn, `
		CREATE TABLE Test(id INTEGER PRIMARY KEY, value TEXT) WITHOUT ROWID;
		-- Use a recursive expression to insert a set of junk rows
		WITH rows(id,value) AS (
			SELECT 1, 'row 1'
			UNION ALL
			SELECT id+1, 'row '||(id+1) FROM rows
			LIMIT 10
		)
		INSERT INTO Test SELECT id,value FROM rows;
		`)
	}
	if err != nil {
		b.Fatal("Could not open DB", err)
	}
	return pool
}

func BenchmarkManualStructs(b *testing.B) {
	pool := attachInMemoryTestDB(b)

	b.RunParallel(func(pb *testing.PB) {
		conn := pool.Get(context.TODO())
		defer pool.Put(conn)

		output := []*BenchRow{}

		// Hand-tuned to be as fast as possible for a fair comparison.
		// In testing, looking up from names seems faster than using column indexes.
		// As the number of rows selected increases, this becomes relatively slower
		// and around 1k rows becomes slower than the Absorb methods.
		for testIdx := b.N; pb.Next(); testIdx++ {
			sqlitex.Exec(conn, "SELECT * FROM Test", func(stmt *sqlite.Stmt) error {
				output = append(output, &BenchRow{
					ID:    stmt.GetInt64("id"),
					Label: stmt.GetText("value"),
				})
				return nil
			})
		}
	})
}

func BenchmarkSliceAbsorption(b *testing.B) {
	pool := attachInMemoryTestDB(b)

	b.RunParallel(func(pb *testing.PB) {
		conn := pool.Get(context.TODO())
		defer pool.Put(conn)

		for testIdx := b.N; pb.Next(); testIdx++ {
			stmt := conn.Prep("SELECT * FROM Test")
			absorb.Absorb(&[]*BenchRow{}, CastAbsorbable(stmt))
		}
	})
}

func BenchmarkChannelAbsorption(b *testing.B) {
	pool := attachInMemoryTestDB(b)

	b.RunParallel(func(pb *testing.PB) {
		conn := pool.Get(context.TODO())
		defer pool.Put(conn)

		for testIdx := b.N; pb.Next(); testIdx++ {
			// Consume pointers to the structs
			ch := make(chan *BenchRow, 25)
			go func() {
				stmt := conn.Prep("SELECT * FROM Test")
				absorb.Absorb(ch, CastAbsorbable(stmt))
				close(ch)
			}()
			// Drain the channel
			for range ch {
			}
		}
	})
}
