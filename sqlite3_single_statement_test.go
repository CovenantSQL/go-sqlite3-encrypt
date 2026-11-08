//go:build cgo

package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func init() {
	sql.Register("sqlite3_single_statement", &SQLiteDriver{SingleStatement: true})
}

func openSingleStatement(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3_single_statement", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	// One connection, so that a PRAGMA would show on the next query.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE r (v); CREATE TABLE s (v)"); !errors.Is(err, ErrMultipleStatements) {
		t.Fatalf("batch error = %v, want ErrMultipleStatements", err)
	}
	for _, q := range []string{"CREATE TABLE r (v)", "CREATE TABLE s (v)"} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// singleStatementCalls runs query, with args, through every way the driver
// executes a string, and returns their errors.
func singleStatementCalls(db *sql.DB, query string, args ...any) map[string]error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := map[string]error{}
	_, errs["Exec"] = db.Exec(query, args...)
	_, errs["ExecContext"] = db.ExecContext(ctx, query, args...)
	for name, run := range map[string]func() (*sql.Rows, error){
		"Query":        func() (*sql.Rows, error) { return db.Query(query, args...) },
		"QueryContext": func() (*sql.Rows, error) { return db.QueryContext(ctx, query, args...) },
	} {
		rows, err := run()
		if err == nil {
			for rows.Next() {
			}
			err = rows.Err()
			rows.Close()
		}
		errs[name] = err
	}
	for name, prepare := range map[string]func() (*sql.Stmt, error){
		"Prepare":        func() (*sql.Stmt, error) { return db.Prepare(query) },
		"PrepareContext": func() (*sql.Stmt, error) { return db.PrepareContext(ctx, query) },
	} {
		stmt, err := prepare()
		if err == nil {
			_, err = stmt.Exec(args...)
			stmt.Close()
		}
		errs[name] = err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err == nil {
		if _, err = tx.Exec(query, args...); err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
	}
	errs["Tx"] = err
	return errs
}

func TestSingleStatement(t *testing.T) {
	for _, dsn := range []string{":memory:", ":memory:?_stmt_cache_size=8"} {
		t.Run(dsn, func(t *testing.T) {
			db := openSingleStatement(t, dsn)
			count := func() (n int) {
				t.Helper()
				if err := db.QueryRow("SELECT count(*) FROM r").Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n
			}

			for _, q := range []struct {
				query string
				args  []any
			}{
				{"INSERT INTO r VALUES (1); INSERT INTO r VALUES (2)", nil},
				{"INSERT INTO r VALUES (?); INSERT INTO r VALUES (?)", []any{1, 2}},
				{"INSERT INTO r VALUES (1); x", nil},
				{"INSERT INTO r VALUES (1); /* outer /* inner */ SELECT 2; -- */", nil},
				{"INSERT INTO r VALUES (1); /*", nil},
				{"INSERT INTO r VALUES (1);\v", nil},
				{"CREATE TABLE d (v); SELECT 1", nil},
				{"; INSERT INTO r VALUES (1); SELECT 2", nil},
			} {
				for call, err := range singleStatementCalls(db, q.query, q.args...) {
					if !errors.Is(err, ErrMultipleStatements) {
						t.Errorf("%s(%q): error = %v, want ErrMultipleStatements", call, q.query, err)
					}
				}
			}
			if n := count(); n != 0 {
				t.Errorf("refused strings inserted %d rows", n)
			}
			var tables int
			if err := db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE name = 'd'").Scan(&tables); err != nil || tables != 0 {
				t.Errorf("refused string created table d: %d, %v", tables, err)
			}

			for _, query := range []string{
				"INSERT INTO r VALUES (?);",
				"INSERT INTO r VALUES (?) ;; ",
				"; /* c */ ; INSERT INTO r VALUES (?);",
				"INSERT INTO r VALUES (?); --x",
				"INSERT INTO r VALUES (?); /*x",
				"INSERT INTO r VALUES (?); \v",
				"INSERT INTO r VALUES (?);\xef\xbb\xbf",
			} {
				before := count()
				for call, err := range singleStatementCalls(db, query, 1) {
					if err != nil {
						t.Errorf("%s(%q): %v", call, query, err)
					}
				}
				// Exec, ExecContext, Prepare, PrepareContext and Tx insert;
				// Query and QueryContext do too, as they step the INSERT.
				if n := count() - before; n != 7 {
					t.Errorf("%q inserted %d rows, want 7", query, n)
				}
			}

			before := count()
			if _, err := db.Exec("CREATE TRIGGER tr AFTER INSERT ON s BEGIN INSERT INTO r VALUES (1); INSERT INTO r VALUES (2); END;"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("INSERT INTO s VALUES (1)"); err != nil {
				t.Fatal(err)
			}
			if n := count() - before; n != 2 {
				t.Errorf("the trigger inserted %d rows, want 2", n)
			}
		})
	}
}

// TestSingleStatementTailNotPrepared checks that the rest of a refused
// string is not compiled: PRAGMA case_sensitive_like acts when it is
// prepared.
func TestSingleStatementTailNotPrepared(t *testing.T) {
	db := openSingleStatement(t, ":memory:")
	if _, err := db.Query("SELECT 1; PRAGMA case_sensitive_like=1"); !errors.Is(err, ErrMultipleStatements) {
		t.Fatalf("error = %v, want ErrMultipleStatements", err)
	}
	var like bool
	if err := db.QueryRow("SELECT 'a' LIKE 'A'").Scan(&like); err != nil {
		t.Fatal(err)
	}
	if !like {
		t.Error("LIKE became case sensitive")
	}
}

func TestSingleStatementOffRunsBatches(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE r (v); INSERT INTO r VALUES (1); INSERT INTO r VALUES (2)"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM r").Scan(&n); err != nil || n != 2 {
		t.Errorf("count = %d, %v, want 2", n, err)
	}
}
