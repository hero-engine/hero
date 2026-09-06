package projection

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/mailquery"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/projectregistry"
)

// RegistryMailSource adapts the machine-global project registry to Mail's
// owning services. Each Mail service remains authoritative for its peer ID;
// this facade only aggregates their unread views and routes actions back to the
// service that owns the addressed envelope.
//
// The machine-global registry (~/.hero/projects.json) can hold every project
// ever registered on this machine, including projects that are merely peers
// of one another and have nothing to do with the project this source was
// built for. projectPeerID pins Attention's own view of Mail to that one
// project's own mailbox, mirroring `hero mail inbox` (no --project) — it must
// never widen to the registry's full "browse every project" behavior that
// `mailquery.Service` otherwise offers unscoped queries for.
type RegistryMailSource struct {
	services      []*mail.Service
	query         *mailquery.Service
	projectPeerID string
}

func NewRegistryMailSource(stateRoot string, registry *projectregistry.Registry, projectPeerID string) (*RegistryMailSource, error) {
	if registry == nil {
		return nil, errors.New("project registry is unavailable")
	}
	if projectPeerID == "" {
		return nil, errors.New("project peer ID is required")
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
	source := &RegistryMailSource{projectPeerID: projectPeerID}
	query, err := mailquery.NewService(stateRoot, registry)
	if err != nil {
		return nil, err
	}
	source.query = query
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

// Threads always scopes to this source's own project, regardless of what the
// caller requests: Attention's Mail view must never leak another registered
// project's mailbox, even a mailbox between two other, unrelated peers.
func (s *RegistryMailSource) Threads(request mailthread.ThreadListRequest) mailthread.ThreadListResponse {
	request.ProjectPeerID = s.projectPeerID
	return s.query.Threads(request)
}

func (s *RegistryMailSource) ThreadAction(request mailthread.ActionRequest) (mailthread.ThreadView, error) {
	for _, service := range s.services {
		if _, _, err := service.Thread(request.Identity.ProjectPeerID, request.Identity.ThreadID); err == nil {
			return service.ThreadAction(request)
		} else if !errors.Is(err, mail.ErrRecipientMismatch) && !errors.Is(err, mail.ErrNotFound) {
			return mailthread.ThreadView{}, err
		}
	}
	return mailthread.ThreadView{}, mail.ErrNotFound
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
