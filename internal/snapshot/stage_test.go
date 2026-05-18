package snapshot

import (
	"testing"

	"github.com/hero-engine/hero/internal/spec"
)

func TestDeriveStage(t *testing.T) {
	tests := []struct {
		name string
		in   StageInputs
		want Stage
	}{
		{
			name: "no activity = concept",
			in:   StageInputs{SpecsByStatus: map[spec.Status]int{}},
			want: StageConcept,
		},
		{
			name: "delivering only, no release = building",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusDelivering: 2},
			},
			want: StageBuilding,
		},
		{
			name: "completed only, no release, no tag = scaffolded",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusCompleted: 1},
			},
			want: StageScaffolded,
		},
		{
			name: "completed only, shipped tag = maturing",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusCompleted: 3},
				HasShippedTag: true,
			},
			want: StageMaturing,
		},
		{
			name: "release scope >=50% in flight = shipping-v1",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusDelivering: 2, spec.StatusCompleted: 5},
				ReleaseDone:   5,
				ReleaseTotal:  9,
			},
			want: StageShippingV1,
		},
		{
			name: "release scope <50%, in flight = building",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusDelivering: 1, spec.StatusCompleted: 1},
				ReleaseDone:   1,
				ReleaseTotal:  10,
			},
			want: StageBuilding,
		},
		{
			name: "release scope all done, nothing in flight, no tag = shipped",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusCompleted: 5},
				ReleaseDone:   5,
				ReleaseTotal:  5,
			},
			want: StageShipped,
		},
		{
			name: "release scope all done, tagged = maturing",
			in: StageInputs{
				SpecsByStatus: map[spec.Status]int{spec.StatusCompleted: 5},
				ReleaseDone:   5,
				ReleaseTotal:  5,
				HasShippedTag: true,
			},
			want: StageMaturing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveStage(tt.in)
			if got != tt.want {
				t.Errorf("DeriveStage(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsValidStage(t *testing.T) {
	for _, s := range AllStages() {
		if !IsValidStage(s) {
			t.Errorf("AllStages produced invalid stage %q", s)
		}
	}
	if IsValidStage(Stage("nope")) {
		t.Error("nope is not a valid stage")
	}
}
