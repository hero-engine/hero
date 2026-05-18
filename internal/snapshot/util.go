package snapshot

import "os"

// osStat is a tiny indirection that lets unit tests stub file stat
// calls without touching the filesystem.
var osStat = os.Stat
