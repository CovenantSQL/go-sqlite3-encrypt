//go:build cgo
// +build cgo

package sqlite3

/*
#ifndef USE_LIBSQLITE3
#include "sqlite3-binding.h"
#else
#include <sqlite3.h>
#endif

static sqlite3_vfs noCurrentTimeVfs;

static int noCurrentTime(sqlite3_vfs *vfs, double *now) {
  *now = 0;
  return SQLITE_ERROR;
}

static int noCurrentTimeInt64(sqlite3_vfs *vfs, sqlite3_int64 *now) {
  *now = 0;
  return SQLITE_ERROR;
}

// Register a copy of the default VFS named "nocurrenttime" whose current time
// is unavailable.
static int _sqlite3_register_no_current_time_vfs(void) {
  sqlite3_vfs *base = sqlite3_vfs_find(0);
  if (base == 0) {
    return SQLITE_ERROR;
  }
  noCurrentTimeVfs = *base;
  noCurrentTimeVfs.zName = "nocurrenttime";
  noCurrentTimeVfs.pNext = 0;
  noCurrentTimeVfs.xCurrentTime = noCurrentTime;
  if (noCurrentTimeVfs.iVersion >= 2) {
    noCurrentTimeVfs.xCurrentTimeInt64 = noCurrentTimeInt64;
  }
  return sqlite3_vfs_register(&noCurrentTimeVfs, 0);
}
*/
import "C"

import (
	"fmt"
	"sync"
)

// noCurrentTimeVFS is the VFS that the DSN parameter vfs=nocurrenttime selects.
// Date and time functions get no current time from it, so 'now' and the
// CURRENT_* keywords give NULL, and every connection computes the same results
// from the same data whenever it runs.
const noCurrentTimeVFS = "nocurrenttime"

var (
	noCurrentTimeOnce sync.Once
	noCurrentTimeErr  error
)

func registerNoCurrentTimeVFS() error {
	noCurrentTimeOnce.Do(func() {
		if rv := C._sqlite3_register_no_current_time_vfs(); rv != C.SQLITE_OK {
			noCurrentTimeErr = fmt.Errorf("sqlite3: registering the %s VFS failed: %d", noCurrentTimeVFS, rv)
		}
	})
	return noCurrentTimeErr
}
