//go:build cgo && !libsqlite3
// +build cgo,!libsqlite3

package sqlite3

import (
	"bytes"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCryptoKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encrypted.db")
	// The quote checks that the key is escaped in PRAGMA key.
	keyed := "file:" + path + "?_crypto_key=it's-a-secret"

	db, err := sql.Open("sqlite3", keyed)
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err := db.QueryRow("PRAGMA cipher_version").Scan(&version); err != nil || version == "" {
		t.Fatalf("SQLCipher is not active: version %q, err %v", version, err)
	}
	if _, err := db.Exec("CREATE TABLE t (v TEXT); INSERT INTO t VALUES ('hello')"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	header := make([]byte, 16)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Read(header)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(header, []byte("SQLite format 3\x00")) {
		t.Fatal("database file is not encrypted")
	}

	for _, dsn := range []string{"file:" + path + "?_crypto_key=wrong", "file:" + path} {
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			t.Fatal(err)
		}
		var n int
		err = db.QueryRow("SELECT count(*) FROM t").Scan(&n)
		var sqliteErr Error
		if !errors.As(err, &sqliteErr) || sqliteErr.Code != ErrNotADB {
			t.Errorf("read the encrypted database with %q: want ErrNotADB, got %v", dsn, err)
		}
		db.Close()
	}

	db, err = sql.Open("sqlite3", keyed)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var v string
	if err := db.QueryRow("SELECT v FROM t").Scan(&v); err != nil || v != "hello" {
		t.Fatalf("read back %q, err %v", v, err)
	}
}

func TestCryptoKeyWithNUL(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "nul.db")+"?_crypto_key=secret%00part")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err == nil {
		t.Fatal("opened a database with a key containing NUL")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error shows the key: %v", err)
	}
}
