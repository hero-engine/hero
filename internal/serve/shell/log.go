package shell

import (
	"io"
	"os"
)

// defaultStderr is the destination for shell warnings. Indirected
// through a var so tests can capture output if needed.
var defaultStderr io.Writer = os.Stderr
