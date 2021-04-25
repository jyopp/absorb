package main

import (
	"fmt"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/jyopp/absorb"
)

func main() {
	conn, err := sqlite.OpenConn("file::memory:", 0)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Create a test table and insert some values.
	err = sqlitex.ExecScript(conn, `
	CREATE TABLE Test(id INTEGER PRIMARY KEY, value TEXT);
	-- Use a recursive expression to insert a set of junk rows
	WITH rows(id,value) AS (
		SELECT 1, 'row 1'
		UNION ALL
		SELECT id+1, 'row '||(id+1) FROM rows
		LIMIT 25
	)
	INSERT INTO Test SELECT id,value FROM rows;
	`)
	if err != nil {
		panic(err)
	}

	type TestStruct struct {
		ID    int64  `sqlite:"id"`
		Label string `sqlite:"value"`
	}

	// Declare
	ch := make(chan TestStruct)

	go func(sql string) {
		// Read from the DB and send each row to the channel.
		// Rows are automatically marshaled into structs as they are absorbed.
		stmt := StatementWrapper{
			Stmt: conn.Prep(sql),
		}
		absorb.Absorb(ch, &stmt)
		close(ch)
	}("SELECT * FROM Test")

	for row := range ch {
		fmt.Printf("<-%T%+v\n", row, row)
	}
	fmt.Println("Done")
}
