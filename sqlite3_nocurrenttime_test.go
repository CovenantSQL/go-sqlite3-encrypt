//go:build cgo
// +build cgo

package sqlite3

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestNoCurrentTimeVFS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nocurrenttime.db")
	db, err := sql.Open("sqlite3", "file:"+path+"?vfs=nocurrenttime&_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Files are read and written as with the default VFS.
	if _, err := db.Exec("CREATE TABLE t (v TEXT); INSERT INTO t VALUES (date('2000-01-02'))"); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := db.QueryRow("SELECT v FROM t").Scan(&v); err != nil || v != "2000-01-02" {
		t.Fatalf("read back %q, err %v", v, err)
	}

	for _, q := range []struct {
		query string
		args  []any
	}{
		{"SELECT date('now')", nil},
		{"SELECT date(?)", []any{"now"}},
		{"SELECT date('n' || 'ow')", nil},
		{"SELECT datetime()", nil},
		{"SELECT julianday('now')", nil},
		{"SELECT unixepoch()", nil},
		{"SELECT strftime('%Y')", nil},
		{"SELECT CURRENT_TIMESTAMP", nil},
	} {
		var got sql.NullString
		if err := db.QueryRow(q.query, q.args...).Scan(&got); err != nil {
			t.Errorf("%s: %v", q.query, err)
		} else if got.Valid {
			t.Errorf("%s: got the current time %q, want NULL", q.query, got.String)
		}
	}

	plain, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	var now sql.NullString
	if err := plain.QueryRow("SELECT date('now')").Scan(&now); err != nil || !now.Valid {
		t.Fatalf("the default VFS has no current time: %q, err %v", now.String, err)
	}
}

func TestMoreThanOneVFS(t *testing.T) {
	// SQLite would open this with the last one, the default VFS.
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "two.db")+"?vfs=nocurrenttime&vfs=unix")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err == nil {
		t.Fatal("opened a database with two vfs parameters")
	}
}
