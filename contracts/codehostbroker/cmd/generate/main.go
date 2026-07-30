package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

func main() {
	data, err := codehostbroker.CanonicalFixture()
	if err != nil {
		panic(err)
	}
	digest, err := codehostbroker.CanonicalFixtureSHA256()
	if err != nil {
		panic(err)
	}
	directory := filepath.Join("testdata", "v2")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "consumer-fixture.json"), data, 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "consumer-fixture.sha256"), []byte(digest+"\n"), 0o644); err != nil {
		panic(err)
	}
	fmt.Println(digest)
}
