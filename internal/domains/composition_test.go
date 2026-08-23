package domains

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveCompositionCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		declared *Composition
		legacy   DomainID
		want     ResolvedComposition
	}{
		{name: "missing defaults to engineering", want: ResolvedComposition{Primary: DomainEngineering}},
		{name: "legacy scalar", legacy: DomainPM, want: ResolvedComposition{Primary: DomainPM}},
		{
			name: "canonical ordered extensions are deduplicated",
			declared: &Composition{
				Primary:    DomainEngineering,
				Extensions: []DomainID{DomainQA, DomainPM, DomainQA},
			},
			want: ResolvedComposition{Primary: DomainEngineering, Extensions: []DomainID{DomainQA, DomainPM}},
		},
		{
			name:     "matching legacy and canonical",
			legacy:   DomainQA,
			declared: &Composition{Primary: DomainQA},
			want:     ResolvedComposition{Primary: DomainQA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveComposition(tt.declared, tt.legacy)
			if err != nil {
				t.Fatalf("ResolveComposition() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveComposition() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveCompositionRejectsInvalidRoles(t *testing.T) {
	tests := []struct {
		name     string
		declared *Composition
		legacy   DomainID
		contains string
	}{
		{name: "missing primary", declared: &Composition{}, contains: "primary is required"},
		{name: "unknown primary", declared: &Composition{Primary: "unknown"}, contains: "not available"},
		{name: "unsupported extension", declared: &Composition{Primary: DomainPM, Extensions: []DomainID{DomainSales}}, contains: "does not support"},
		{name: "primary repeated as extension", declared: &Composition{Primary: DomainQA, Extensions: []DomainID{DomainQA}}, contains: "both primary and an extension"},
		{name: "conflicting legacy", legacy: DomainPM, declared: &Composition{Primary: DomainQA}, contains: "conflicting domain configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveComposition(tt.declared, tt.legacy)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("ResolveComposition() error = %v, want containing %q", err, tt.contains)
			}
		})
	}
}

func TestResolveCompositionRejectsExtensionsForUnrelatedPrimary(t *testing.T) {
	_, err := ResolveComposition(&Composition{
		Primary: DomainSales, Extensions: []DomainID{DomainPM},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "does not support capability-pack extensions") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolvedCompositionStackAndContains(t *testing.T) {
	composition := ResolvedComposition{Primary: DomainEngineering, Extensions: []DomainID{DomainPM, DomainQA}}
	if got, want := composition.Stack(), []DomainID{DomainCore, DomainEngineering, DomainPM, DomainQA}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Stack() = %v, want %v", got, want)
	}
	for _, id := range composition.Stack() {
		if !composition.Contains(id) {
			t.Errorf("Contains(%q) = false", id)
		}
	}
	if composition.Contains(DomainSales) {
		t.Error("Contains(sales) = true")
	}
}
