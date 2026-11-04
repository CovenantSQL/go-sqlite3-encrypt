//go:build cgo && sqlite_omit_localtime && !libsqlite3

package sqlite3

import (
	"database/sql"
	"testing"
)

func TestOmitLocaltime(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, modifier := range []string{"localtime", "utc"} {
		var got sql.NullString
		if err := db.QueryRow("SELECT datetime('2000-01-01 12:00:00', ?)", modifier).Scan(&got); err != nil {
			t.Errorf("%s: %v", modifier, err)
		} else if got.Valid {
			t.Errorf("%s: got %q, want NULL", modifier, got.String)
		}
	}
}
