package main

import (
	"crawshaw.io/sqlite"
	"github.com/jyopp/absorb"
)

// StatementWrapper implements absorb.Absorbable for an embedded *sqlite.Stmt
type StatementWrapper struct {
	*sqlite.Stmt
}

func (s *StatementWrapper) colInterface(colNum int) interface{} {
	switch s.ColumnType(colNum) {
	case sqlite.SQLITE_INTEGER:
		return s.ColumnInt64(colNum)
	case sqlite.SQLITE_FLOAT:
		return s.ColumnFloat(colNum)
	case sqlite.SQLITE_TEXT:
		return s.ColumnText(colNum)
	case sqlite.SQLITE_BLOB:
		l := s.ColumnLen(colNum)
		if l > 0 {
			blob := make([]byte, 0, l)
			if l == s.ColumnBytes(colNum, blob) {
				return blob
			}
		}
		return []byte{}
	case sqlite.SQLITE_NULL:
		return nil
	default:
		panic("Encountered undocumented sqlite type")
	}
}

// StatementWrapper implements absorb.Absorbable
func (s *StatementWrapper) Emit(into absorb.Absorber) error {
	colCount := s.ColumnCount()
	keys := make([]string, colCount)
	for idx := range keys {
		keys[idx] = s.ColumnName(idx)
	}

	into.Open("sqlite", -1, keys...)
	defer into.Close()

	// Create one slice that we'll overwrite with each row.
	rowData := make([]interface{}, colCount)

	hasRow, err := s.Step()
	for ; hasRow && (err == nil); hasRow, err = s.Step() {
		// Copy stmt into rowData
		for idx := range rowData {
			rowData[idx] = s.colInterface(idx)
		}
		into.Absorb(rowData...)
	}
	return err
}
