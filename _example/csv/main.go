package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jyopp/absorb"
)

type PersonRecord struct {
	First    string
	Last     string
	Location string `csv:"Last-Seen"`
}

func (r *PersonRecord) FullName() string {
	return strings.TrimSpace(r.First + " " + r.Last)
}

type CSVReader struct {
	*os.File
}

// Emit implements absorb.Absorbable
func (r *CSVReader) Emit(into absorb.Absorber) error {
	if _, err := r.File.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// The underlying reader should own the line storage and reuse it per line.
	reader := csv.NewReader(r.File)

	line, err := reader.Read()
	if err == nil {
		// Configure the absorber with the struct tag "csv" and the first row for its keys.
		into.Open("csv", -1, line...)
		defer into.Close()

		// After the first line, we prefer the string storage to be reused
		reader.ReuseRecord = true
		var asInterface = make([]interface{}, reader.FieldsPerRecord)
		for line, err = reader.Read(); err == nil; line, err = reader.Read() {
			for idx, strVal := range line {
				asInterface[idx] = strVal
			}
			into.Absorb(asInterface...)
		}
		if err == io.EOF {
			return nil
		}
	}
	return err
}

func main() {
	f, err := os.Open("testdata/test.csv")
	if err != nil {
		panic(err)
	}
	reader := &CSVReader{File: f}

	fmt.Println("=== Reading structs from CSV ===")
	var persons []PersonRecord
	if err = absorb.Absorb(&persons, reader); err != nil {
		panic(err)
	}

	for _, person := range persons {
		fmt.Printf("%s was last seen in %s\n", person.FullName(), person.Location)
	}

	// Now read the same data again, but this time as an array of string maps
	fmt.Println("\n=== Reading map of strings from CSV ===")
	var anonymousData []map[string]string
	if err = absorb.Absorb(&anonymousData, reader); err != nil {
		panic(err)
	}
	fmt.Printf("Got %+v\n", anonymousData)

	// And for good measure, read the file in the background
	fmt.Println("\n=== Reading structs in the background ===")
	ch := make(chan *PersonRecord)
	go func() {
		defer close(ch)
		absorb.Absorb(ch, reader)
	}()

	for person := range ch {
		fmt.Printf("Got %v via channel\n", person)
	}
}
