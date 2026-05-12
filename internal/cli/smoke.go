package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// globalSmoke is set by the persistent --smoke flag on rootCmd.
var globalSmoke bool

// smokeRegistry maps top-level command names to their registered smoke
// functions.  Commands that haven't called RegisterSmoke get the default
// behaviour: help is printed and the run exits 0 (no-op).
var smokeRegistry = map[string]func(*cobra.Command) error{}

// RegisterSmoke registers fn as the smoke implementation for cmd.
// Call once per command from an init() function.
// fn should exercise the command's happy path and return nil on success.
func RegisterSmoke(cmd *cobra.Command, fn func(*cobra.Command) error) {
	smokeRegistry[cmd.Name()] = fn
}

// smokeInterceptor wraps a RunE so that when globalSmoke is true the
// call goes to runSmokeFor instead of the real command body.
// Applied once at init time to every top-level command with a RunE.
func smokeInterceptor(original func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if globalSmoke {
			return runSmokeFor(cmd)
		}
		return original(cmd, args)
	}
}

// runSmokeFor executes the smoke pass: prints the command's help, then
// calls the registered smoke fn.  If no fn is registered, exits 0
// with a one-line "OK (no-op)" message.
func runSmokeFor(cmd *cobra.Command) error {
	// Always print help so humans and CI can confirm the right binary ran.
	if err := cmd.Help(); err != nil {
		return err
	}
	fmt.Println()

	fn, ok := smokeRegistry[cmd.Name()]
	if !ok {
		fmt.Printf("smoke: %s — OK (no-op)\n", cmd.Name())
		return nil
	}
	return fn(cmd)
}
