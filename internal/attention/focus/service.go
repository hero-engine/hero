package focus

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hero-engine/hero/contracts/attention"
)

type CreateRequest struct {
	Title     string
	Prompt    string
	Lifecycle string
	Project   *attention.ProjectReference
	Origin    *attention.ProvenanceReference
	OriginKey string
}

type ListedItem struct {
	Item
	Availability string `json:"availability"`
}

type LaunchIntent struct {
	ID       string                     `json:"id"`
	Revision int64                      `json:"revision"`
	Prompt   string                     `json:"prompt"`
	Project  attention.ProjectReference `json:"project"`
	Path     string                     `json:"path"`
}

type MissingProjectError struct{ Project *attention.ProjectReference }

func (e *MissingProjectError) Error() string {
	if e.Project == nil {
		return "focus item has no project target"
	}
	return fmt.Sprintf("focus project %q is missing", e.Project.DisplayName)
}

type Service struct {
	store    *Store
	resolver ProjectResolver
	now      func() time.Time
	newID    func() (string, error)
}

func NewService(store *Store, resolver ProjectResolver) *Service {
	return &Service{store: store, resolver: resolver, now: time.Now, newID: randomID}
}

func (s *Service) Create(req CreateRequest) (Item, error) {
	if req.Origin != nil || req.OriginKey != "" {
		return Item{}, errors.New("source-derived focus requires CreateOrGet")
	}
	item, err := s.newItem(req)
	if err != nil {
		return Item{}, err
	}
	return s.store.Create(item)
}

func (s *Service) CreateOrGet(req CreateRequest) (Item, bool, error) {
	if req.Origin == nil || strings.TrimSpace(req.Origin.Kind) == "" || strings.TrimSpace(req.Origin.SourceID) == "" {
		return Item{}, false, errors.New("typed provenance is required")
	}
	if strings.TrimSpace(req.OriginKey) == "" {
		return Item{}, false, errors.New("origin key is required")
	}
	if !utf8.ValidString(req.OriginKey) || len(req.OriginKey) > 512 {
		return Item{}, false, errors.New("origin key must be valid UTF-8 and at most 512 bytes")
	}
	item, err := s.newItem(req)
	if err != nil {
		return Item{}, false, err
	}
	return s.store.CreateOrGet(item)
}

func (s *Service) Get(id string) (ListedItem, error) {
	if err := validateID(id); err != nil {
		return ListedItem{}, err
	}
	item, err := s.store.Get(id)
	if err != nil {
		return ListedItem{}, err
	}
	return s.present(item), nil
}

func (s *Service) List(state string) ([]ListedItem, error) {
	if state == "" {
		state = "active"
	}
	if state != "active" && state != "all" && !validLifecycle(state) {
		return nil, fmt.Errorf("invalid focus state %q", state)
	}
	items, err := s.store.List()
	if err != nil {
		return nil, err
	}
	listed := make([]ListedItem, 0, len(items))
	for _, item := range items {
		if state == "active" && item.Lifecycle == attention.FocusDone {
			continue
		}
		if state != "active" && state != "all" && item.Lifecycle != state {
			continue
		}
		listed = append(listed, s.present(item))
	}
	return listed, nil
}

func (s *Service) Move(id, lifecycle string, expected int64) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	if !validLifecycle(lifecycle) {
		return Item{}, fmt.Errorf("invalid focus lifecycle %q", lifecycle)
	}
	if expected < 1 {
		return Item{}, errors.New("revision must be positive")
	}
	return s.store.Replace(id, expected, func(item *Item) (bool, error) {
		if item.Lifecycle == lifecycle {
			return false, nil
		}
		item.Lifecycle = lifecycle
		item.UpdatedAt = utcNow(s.now())
		return true, nil
	})
}

func (s *Service) LaunchIntent(id string) (LaunchIntent, error) {
	listed, err := s.Get(id)
	if err != nil {
		return LaunchIntent{}, err
	}
	resolved := s.resolver.ResolveReference(listed.Project)
	if listed.Project == nil || resolved.Availability != ProjectAvailable || resolved.Path == "" || resolved.Reference == nil {
		return LaunchIntent{}, &MissingProjectError{Project: listed.Project}
	}
	return LaunchIntent{ID: listed.ID, Revision: listed.Revision, Prompt: listed.Prompt, Project: *resolved.Reference, Path: resolved.Path}, nil
}

// ProjectAvailability exposes the same registry-backed availability used by
// Focus presentation without exposing registry or filesystem state to
// projection consumers.
func (s *Service) ProjectAvailability(project *attention.ProjectReference) string {
	return s.resolver.ResolveReference(project).Availability
}

func (s *Service) newItem(req CreateRequest) (Item, error) {
	if strings.TrimSpace(req.Title) == "" {
		return Item{}, errors.New("title is required")
	}
	if !utf8.ValidString(req.Title) || utf8.RuneCountInString(req.Title) > attention.MaxSubjectCharacters {
		return Item{}, errors.New("title must be valid UTF-8 and at most 200 characters")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return Item{}, errors.New("prompt is required")
	}
	if !utf8.ValidString(req.Prompt) || len(req.Prompt) > attention.MaxFocusPromptBytes {
		return Item{}, errors.New("prompt must be valid UTF-8 and at most 65536 bytes")
	}
	if req.Lifecycle == "" {
		req.Lifecycle = attention.FocusInbox
	}
	if !validLifecycle(req.Lifecycle) {
		return Item{}, fmt.Errorf("invalid focus lifecycle %q", req.Lifecycle)
	}
	if req.Project != nil && (strings.TrimSpace(req.Project.PeerID) == "" || strings.TrimSpace(req.Project.DisplayName) == "") {
		return Item{}, errors.New("project peer_id and display_name are required")
	}
	id, err := s.newID()
	if err != nil {
		return Item{}, err
	}
	now := utcNow(s.now())
	return Item{SchemaVersion: attention.SchemaVersion, ID: id, Project: req.Project, Title: req.Title, Prompt: req.Prompt, Lifecycle: req.Lifecycle, CreatedAt: now, UpdatedAt: now, Origin: req.Origin, OriginKey: req.OriginKey}, nil
}

func (s *Service) present(item Item) ListedItem {
	availability := ProjectAvailable
	if item.Project != nil {
		availability = s.resolver.ResolveReference(item.Project).Availability
	}
	return ListedItem{Item: item, Availability: availability}
}

func validLifecycle(v string) bool {
	return v == attention.FocusInbox || v == attention.FocusToday || v == attention.FocusLater || v == attention.FocusDone
}

func validateID(id string) error {
	if !strings.HasPrefix(id, "focus_") || len(id) == len("focus_") {
		return errors.New("invalid focus id")
	}
	for _, r := range id[len("focus_"):] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return errors.New("invalid focus id")
		}
	}
	return nil
}

func randomID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate focus id: %w", err)
	}
	return "focus_" + hex.EncodeToString(b), nil
}
