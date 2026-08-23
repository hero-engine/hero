package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mailthread-bundle <project-root>")
		os.Exit(2)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatal(err)
	}
	bundle, err := mailthread.BuildBundle(filepath.Join(root, "contracts", "attention", "mailthread"))
	if err != nil {
		fatal(err)
	}
	if err := mailthread.WriteBundle(filepath.Join(root, filepath.FromSlash(mailthread.BundlePath)), bundle); err != nil {
		fatal(err)
	}
	fmt.Println(bundle.ManifestSHA256)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
