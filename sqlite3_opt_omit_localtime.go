//go:build sqlite_omit_localtime

package sqlite3

// Without the localtime and utc modifiers, date and time functions do not
// depend on the time zone of the host.

/*
#cgo CFLAGS: -DSQLITE_OMIT_LOCALTIME
*/
import "C"
