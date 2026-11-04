//go:build cgo && sqlite_omit_localtime && libsqlite3

package sqlite3

import (
	"database/sql"
	"testing"
)

func TestOmitLocaltimeWithoutTheOption(t *testing.T) {
	if compileOptionUsed("OMIT_LOCALTIME") {
		t.Skip("the system SQLite is built with SQLITE_OMIT_LOCALTIME")
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err == nil {
		t.Fatal("opened a database with the sqlite_omit_localtime tag and a SQLite that has localtime")
	}
}
