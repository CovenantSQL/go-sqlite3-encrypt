//go:build cgo && libsqlite3
// +build cgo,libsqlite3

package sqlite3

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestCryptoKeyWithoutSQLCipher(t *testing.T) {
	plain, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	var hasCodec bool
	if err := plain.QueryRow("SELECT sqlite_compileoption_used('HAS_CODEC')").Scan(&hasCodec); err != nil {
		t.Fatal(err)
	}
	if hasCodec {
		t.Skip("the system SQLite is built with SQLCipher")
	}

	path := filepath.Join(t.TempDir(), "plain.db")
	db, err := sql.Open("sqlite3", "file:"+path+"?_crypto_key=secret")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err == nil {
		t.Fatal("opened a database with _crypto_key without SQLCipher")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("created the database file, stat error %v", err)
	}
}
