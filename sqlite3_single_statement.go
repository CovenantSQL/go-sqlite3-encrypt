package sqlite3

import (
	"errors"
	"strings"
)

// ErrMultipleStatements is returned in single-statement mode, see
// SQLiteDriver.SingleStatement, for a query string that holds more than
// one SQL statement.
var ErrMultipleStatements = errors.New("sqlite3: more than one statement in the query")

// onlySQLiteTrivia reports whether the tokenizer of the bundled SQLite,
// sqlite3GetToken in sqlite3-binding.c, finds nothing in s but whitespace,
// comments and semicolons, that is, no further statement.
func onlySQLiteTrivia(s string) bool {
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == 0:
			// SQLite reads no further than a NUL.
			return true
		case c == ';':
			i++
		case c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r':
			// A space token starts with one of these, not '\v', and goes on
			// over sqlite3Isspace, which includes '\v'.
			for i++; i < len(s) && (s[i] == ' ' || s[i] >= '\t' && s[i] <= '\r'); i++ {
			}
		case strings.HasPrefix(s[i:], "--"):
			// Up to a LF or the end.
			j := strings.IndexAny(s[i:], "\n\x00")
			if j < 0 {
				return true
			}
			i += j
		case strings.HasPrefix(s[i:], "/*") && i+2 < len(s) && s[i+2] != 0:
			// Up to the first "*/" or the end; comments do not nest. A "/*"
			// at the very end is a slash and a star, not a comment.
			j := strings.Index(s[i+2:], "*/")
			if n := strings.IndexByte(s[i+2:], 0); j < 0 || n >= 0 && n < j {
				return true
			}
			i += 2 + j + 2
		case strings.HasPrefix(s[i:], "\xef\xbb\xbf"):
			// A UTF-8 byte order mark is a space token.
			i += 3
		default:
			return false
		}
	}
	return true
}
