package projection

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/projectregistry"
)

// RegistryMailSource adapts the machine-global project registry to Mail's
// owning services. Each Mail service remains authoritative for its peer ID;
// this facade only aggregates their unread views and routes actions back to the
// service that owns the addressed envelope.
type RegistryMailSource struct {
	services []*mail.Service
}

func NewRegistryMailSource(stateRoot string, registry *projectregistry.Registry) (*RegistryMailSource, error) {
	if registry == nil {
		return nil, errors.New("project registry is unavailable")
	}
	store, err := mail.NewStore(stateRoot)
	if err != nil {
		return nil, err
	}
	entries := registry.List()
	slugs := make([]string, 0, len(entries))
	for slug := range entries {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	seen := make(map[string]bool)
	source := &RegistryMailSource{}
	for _, slug := range slugs {
		entry := entries[slug]
		cfg, err := config.Load(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("load registered project %q for attention mail: %w", slug, err)
		}
		if cfg.PeerID == "" || seen[cfg.PeerID] {
			continue
		}
		seen[cfg.PeerID] = true
		source.services = append(source.services, mail.NewService(store, entry.Path, cfg))
	}
	return source, nil
}

func (s *RegistryMailSource) Inbox(_ string, unread bool) ([]mail.ListedMessage, error) {
	result := make([]mail.ListedMessage, 0)
	for _, service := range s.services {
		items, err := service.Inbox("", unread)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

func (s *RegistryMailSource) Action(request mail.ActionRequest) (mail.ActionResult, error) {
	for _, service := range s.services {
		if _, err := service.Show(request.MessageID, false); err == nil {
			return service.Action(request)
		} else if !errors.Is(err, mail.ErrNotFound) {
			return mail.ActionResult{}, err
		}
	}
	return mail.ActionResult{}, mail.ErrNotFound
}
