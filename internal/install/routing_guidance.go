package install

import (
	"fmt"
	"io/fs"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/managed"
)

const routingGuidanceSectionID = "install:natural-language-routing"

type routingGuidanceSection struct {
	body  string
	title string
	err   error
}

func newRoutingGuidanceSection(opts Options) managed.SectionContributor {
	srcFS := opts.sourceFS()
	if srcFS == nil {
		srcFS = hero.ContentFS()
	}
	data, err := fs.ReadFile(srcFS, "routing.md")
	if err != nil {
		fallback := hero.ContentFS()
		data, err = fs.ReadFile(fallback, "routing.md")
	}
	if err != nil {
		return routingGuidanceSection{err: fmt.Errorf("load canonical engineering routing: %w", err)}
	}
	body, title := splitPackAgentsMd(string(data))
	if title == "" {
		title = "Natural Language Routing"
	}
	return routingGuidanceSection{body: body, title: title}
}

func (routingGuidanceSection) SectionID() string { return routingGuidanceSectionID }

func (s routingGuidanceSection) SectionTitle() string { return s.title }

func (s routingGuidanceSection) Render(_ managed.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.body, nil
}
