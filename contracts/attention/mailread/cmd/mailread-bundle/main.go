package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/contracts/attention/mailread"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mailread-bundle <project-root>")
		os.Exit(2)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatal(err)
	}
	packageDir := filepath.Join(root, "contracts", "attention", "mailread")
	bundle, err := mailread.BuildBundle(packageDir)
	if err != nil {
		fatal(err)
	}
	if err := mailread.WriteBundle(filepath.Join(root, filepath.FromSlash(mailread.BundlePath)), bundle); err != nil {
		fatal(err)
	}
	fmt.Println(bundle.ManifestSHA256)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
