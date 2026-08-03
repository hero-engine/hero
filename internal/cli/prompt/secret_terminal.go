package prompt

import "io"

// secretTerminal is deliberately package-private: protected input has no
// caller-provided reader. The opener is a test seam for acquisition failures,
// not an alternate input channel.
type secretTerminal interface {
	io.Writer
	readPassword() ([]byte, error)
	Close() error
}

var openSecretTerminal = openPlatformSecretTerminal
