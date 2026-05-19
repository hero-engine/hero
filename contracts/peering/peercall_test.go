package peering

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestApproxInt_UnmarshalYAML(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		wantOK bool
		want   ApproxInt
	}{
		{"plain int", "v: 22000\n", true, 22000},
		{"tilde", "v: ~22000\n", true, 22000},
		{"float truncates", "v: 22000.7\n", true, 22000},
		{"quoted int", `v: "22000"` + "\n", true, 22000},
		{"quoted tilde", `v: "~22000"` + "\n", true, 22000},
		{"empty string", `v: ""` + "\n", true, 0},
		{"zero", "v: 0\n", true, 0},
		{"negative rejected", "v: -1\n", false, 0},
		{"garbage rejected", "v: lots\n", false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out struct {
				V ApproxInt `yaml:"v"`
			}
			err := yaml.Unmarshal([]byte(tc.input), &out)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if out.V != tc.want {
					t.Errorf("got %d, want %d", out.V, tc.want)
				}
			} else if err == nil {
				t.Fatalf("expected error, got v=%d", out.V)
			}
		})
	}
}

func TestApproxInt_MarshalYAMLRoundtrip(t *testing.T) {
	in := struct {
		V ApproxInt `yaml:"v"`
	}{V: 22000}
	out, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "v: 22000" {
		t.Errorf("canonical int marshal: got %q, want %q", got, "v: 22000")
	}
}

func TestApproxInt_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		wantOK bool
		want   ApproxInt
	}{
		{"plain int", `{"v":22000}`, true, 22000},
		{"string int", `{"v":"22000"}`, true, 22000},
		{"string tilde", `{"v":"~22000"}`, true, 22000},
		{"null zero", `{"v":null}`, true, 0},
		{"negative rejected", `{"v":-1}`, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out struct {
				V ApproxInt `json:"v"`
			}
			err := json.Unmarshal([]byte(tc.input), &out)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if out.V != tc.want {
					t.Errorf("got %d, want %d", out.V, tc.want)
				}
			} else if err == nil {
				t.Fatalf("expected error, got v=%d", out.V)
			}
		})
	}
}

func TestApproxInt_MarshalJSONCanonical(t *testing.T) {
	in := struct {
		V ApproxInt `json:"v"`
	}{V: 22000}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `{"v":22000}` {
		t.Errorf("got %s, want {\"v\":22000}", string(out))
	}
}
