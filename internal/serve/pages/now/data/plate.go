package data

import (
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// PlateInputs is the per-request input bundle for the On-your-plate
// section.
type PlateInputs struct {
	ProjectRoot string
	HeroDir     string
	UserName    string
}

// LoadPlate returns the top two specs claimed by the active user,
// ordered by last modification. Both pointers are nil when no claimed
// specs exist.
func LoadPlate(in PlateInputs) Plate {
	if in.HeroDir == "" {
		return Plate{}
	}
	specs, err := spec.Discover(in.HeroDir)
	if err != nil {
		return Plate{}
	}

	user := strings.TrimSpace(in.UserName)
	var mine []*spec.Spec
	for _, s := range specs {
		if !s.IsWorkSpec() {
			continue
		}
		// Match either by username or by "you" sentinel value some
		// claim flows write.
		if claimedByMatches(s.ClaimedBy, user) {
			mine = append(mine, s)
		}
	}

	sort.SliceStable(mine, func(i, j int) bool {
		return mine[i].ModifiedAt.After(mine[j].ModifiedAt)
	})

	p := Plate{Total: len(mine)}
	if len(mine) >= 1 {
		card := plateCardFor(mine[0], false)
		p.Primary = &card
	}
	if len(mine) >= 2 {
		card := plateCardFor(mine[1], true)
		p.Secondary = &card
	}
	return p
}

// claimedByMatches accepts a few common spellings of "claimed by me".
func claimedByMatches(claimedBy, user string) bool {
	if claimedBy == "" {
		return false
	}
	cb := strings.ToLower(strings.TrimSpace(claimedBy))
	u := strings.ToLower(strings.TrimSpace(user))
	if u == "" {
		return false
	}
	return cb == u || cb == "you" || cb == "me"
}

func plateCardFor(s *spec.Spec, secondary bool) PlateCard {
	status := strings.ToLower(string(s.Status))
	if status == "" {
		status = "planning"
	}

	criteria := s.AcceptanceCriteria()
	addressedPct, addressedFrac := acceptanceProgress(criteria)

	desc := strings.TrimSpace(firstSentence(s.Sections["context"]))
	if desc == "" {
		desc = strings.TrimSpace(firstSentence(s.Sections["goal"]))
	}
	if desc == "" {
		desc = s.Title
	}

	meta := []PlateMeta{
		{Label: "claimed by " + safeStr(s.ClaimedBy, "you")},
	}
	actions := []PlateAction{
		{Label: "Open spec", Href: "#"},
	}
	if !secondary {
		actions = append(actions,
			PlateAction{Label: "Continue session", Href: "#"},
			PlateAction{Label: "/deliver", Href: "#", Mono: true},
			PlateAction{Label: "View graph", Href: "#"},
		)
	} else {
		actions = append(actions, PlateAction{Label: "View PR →", Href: "#"})
	}

	return PlateCard{
		Slug:        s.Slug,
		Title:       s.Slug,
		Status:      status,
		StatusLabel: statusLabel(status),
		Description: desc,
		Criteria: ProgressBar{
			Label:   "Acceptance criteria",
			Pct:     addressedPct,
			Value:   addressedFrac,
			Variant: "",
		},
		Coverage: ProgressBar{
			Label:   "Contract coverage",
			Pct:     0,
			Value:   "—",
			Variant: "success",
		},
		Meta:        meta,
		Actions:     actions,
		IsSecondary: secondary,
	}
}

func acceptanceProgress(criteria []spec.Criterion) (pct int, frac string) {
	if len(criteria) == 0 {
		return 0, "—"
	}
	addressed := 0
	for _, c := range criteria {
		if len(c.VerifiedBy) > 0 {
			addressed++
		}
	}
	pct = (addressed * 100) / len(criteria)
	frac = ""
	if len(criteria) > 0 {
		frac = itoa(addressed) + " / " + itoa(len(criteria))
	}
	return
}

func statusLabel(status string) string {
	switch status {
	case "delivering":
		return "Delivering"
	case "review", "in-review":
		return "In review"
	case "completed":
		return "Completed"
	case "planning":
		return "Planning"
	default:
		return strings.Title(status) //nolint:staticcheck // ASCII status names
	}
}

func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	// Strip leading blank lines and pull the first paragraph.
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Cap at ~200 chars so the plate stays one line.
		if len(line) > 200 {
			line = line[:200] + "…"
		}
		return line
	}
	return ""
}

func safeStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func itoa(n int) string {
	// Tiny inlined formatter — avoids dragging strconv into a hot path.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
