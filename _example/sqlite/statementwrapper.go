package main

import (
	"crawshaw.io/sqlite"
	"github.com/jyopp/absorb"
)

// CastAbsorbable returns a simple typecast of an *sqlite.Stmt
// to the alias type StmtAbsorbable, which is an Absorbable facade.
func CastAbsorbable(s *sqlite.Stmt) *StmtAbsorbable {
	return (*StmtAbsorbable)(s)
}

// Casting a *sqlite.Stmt to a *StatementWrapper makes it absorb.Absorbable
type StmtAbsorbable sqlite.Stmt

func (a *StmtAbsorbable) Stmt() *sqlite.Stmt {
	return (*sqlite.Stmt)(a)
}

// StmtAbsorbable implements absorb.Absorbable
func (a *StmtAbsorbable) Emit(into absorb.Absorber) error {
	s := (*sqlite.Stmt)(a)
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
			rowData[idx] = a.colInterface(idx)
		}
		into.Absorb(rowData...)
	}
	return err
}

func (a *StmtAbsorbable) colInterface(colNum int) interface{} {
	s := (*sqlite.Stmt)(a)
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
			blob := make([]byte, l)
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
