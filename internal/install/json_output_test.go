package install

import (
	"encoding/json"
	"strings"
	"testing"
)

// json_output_test.go — round-trip stability tests for the P5 ops API
// JSON output shapes. These are the contract that a Hero-native client and any
// future Hero consumer programmatic-clients depend on; if a field
// renames or disappears, this test fails (intentionally — it's a
// breaking-change tripwire).

func TestInstallJSONOutput_StableShape(t *testing.T) {
	out := InstallJSONOutput{
		Target:     "claude",
		Mode:       "project",
		TargetDir:  "/path",
		Version:    "v0.7.1",
		Result:     &Result{Copied: []CopyAction{{Source: "src", Dest: "dst"}}, Merged: []string{"AGENTS.md"}},
		DurationMs: 42,
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)

	// Required fields (snake_case is the canonical shape consumers parse).
	for _, key := range []string{
		`"target":"claude"`,
		`"mode":"project"`,
		`"target_dir":"/path"`,
		`"hero_version":"v0.7.1"`,
		`"result":{`,
		`"copied":`,
		`"source":"src"`,
		`"dest":"dst"`,
		`"merged":["AGENTS.md"]`,
		`"duration_ms":42`,
	} {
		if !strings.Contains(s, key) {
			t.Errorf("expected %s in JSON, got: %s", key, s)
		}
	}

	// Error field is omitempty.
	if strings.Contains(s, `"error"`) {
		t.Error("error field should be omitted when nil")
	}
}

func TestInstallJSONOutput_WithError(t *testing.T) {
	out := InstallJSONOutput{
		Target: "claude",
		Mode:   "project",
		Error:  NewJSONError("install_failed", &testError{"boom"}),
	}
	data, _ := json.Marshal(out)
	s := string(data)
	if !strings.Contains(s, `"error":{"code":"install_failed","message":"boom"}`) {
		t.Errorf("expected structured error, got: %s", s)
	}
}

func TestNewJSONError_NilReturnsNil(t *testing.T) {
	if NewJSONError("x", nil) != nil {
		t.Error("NewJSONError(_, nil) must return nil so omitempty works")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
