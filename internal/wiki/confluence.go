package wiki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// confluenceWiki syncs specs to Confluence via the REST API v2.
// Confluence Cloud uses basic auth: email + API token.
// Confluence Server/Data Center uses Bearer token (PAT).
type confluenceWiki struct {
	baseURL         string
	spaceKey        string
	token           string
	userEmail       string // required for Confluence Cloud (basic auth)
	parentPageTitle string
	labelPrefix     string
	client          *http.Client
}

func newConfluenceWiki(cfg *config.ConfluenceConfig) (*confluenceWiki, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("confluence base_url is required (e.g. https://mycompany.atlassian.net/wiki)")
	}
	if cfg.SpaceKey == "" {
		return nil, fmt.Errorf("confluence space_key is required (e.g. ENG)")
	}

	token, err := cfg.ResolveToken()
	if err != nil {
		return nil, fmt.Errorf("confluence sync requires a token: %w", err)
	}

	parentTitle := cfg.ParentPageTitle
	if parentTitle == "" {
		parentTitle = "Hero Specs"
	}
	labelPrefix := cfg.LabelPrefix
	if labelPrefix == "" {
		labelPrefix = "hero-"
	}

	return &confluenceWiki{
		baseURL:         strings.TrimRight(cfg.BaseURL, "/"),
		spaceKey:        cfg.SpaceKey,
		token:           token,
		userEmail:       cfg.UserEmail,
		parentPageTitle: parentTitle,
		labelPrefix:     labelPrefix,
		client:          &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *confluenceWiki) Name() string { return "confluence" }

// SyncSpec pushes a single spec to Confluence. Creates or updates a page.
func (c *confluenceWiki) SyncSpec(s *spec.Spec, content string) (string, error) {
	pageName := specToPageName(s)
	body := specToStorageFormat(s, content)

	parentID, err := c.ensureParentPage()
	if err != nil {
		return "", fmt.Errorf("ensuring parent page: %w", err)
	}

	existingPage, err := c.findPage(pageName)
	if err != nil {
		return "", fmt.Errorf("searching for page %q: %w", pageName, err)
	}

	if existingPage != nil {
		if err := c.updatePage(existingPage.ID, pageName, body, existingPage.Version+1); err != nil {
			return "", fmt.Errorf("updating page %q: %w", pageName, err)
		}
		_ = c.applyLabels(existingPage.ID, s)
		return pageName, nil
	}

	pageID, err := c.createPage(parentID, pageName, body)
	if err != nil {
		return "", fmt.Errorf("creating page %q: %w", pageName, err)
	}
	_ = c.applyLabels(pageID, s)
	return pageName, nil
}

// SyncAll pushes all specs to Confluence.
func (c *confluenceWiki) SyncAll(specs []*spec.Spec) ([]string, error) {
	var pages []string
	var errs []string

	for _, s := range specs {
		content, err := os.ReadFile(s.Path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.Slug, err))
			continue
		}

		pageName, err := c.SyncSpec(s, string(content))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.Slug, err))
			continue
		}
		pages = append(pages, pageName)
	}

	if len(errs) > 0 && len(pages) == 0 {
		return nil, fmt.Errorf("all pages failed: %s", strings.Join(errs, "; "))
	}

	return pages, nil
}

// ---------------------------------------------------------------------------
// Confluence REST API helpers
// ---------------------------------------------------------------------------

// confluencePage represents a minimal Confluence page object.
type confluencePage struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version int    `json:"version"`
}

// findPage searches for a page by title in the configured space.
func (c *confluenceWiki) findPage(title string) (*confluencePage, error) {
	url := fmt.Sprintf("%s/rest/api/content?spaceKey=%s&title=%s&expand=version",
		c.baseURL, c.spaceKey, urlEncode(title))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("confluence API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Version struct {
				Number int `json:"number"`
			} `json:"version"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &confluencePage{
		ID:      r.ID,
		Title:   r.Title,
		Version: r.Version.Number,
	}, nil
}

// ensureParentPage finds or creates the parent page for Hero specs.
func (c *confluenceWiki) ensureParentPage() (string, error) {
	page, err := c.findPage(c.parentPageTitle)
	if err != nil {
		return "", err
	}
	if page != nil {
		return page.ID, nil
	}

	// Create root parent page
	rootBody := fmt.Sprintf("<p>Hero-managed spec pages for the %s space.</p>", c.spaceKey)
	return c.createPage("", c.parentPageTitle, rootBody)
}

// createPage creates a new Confluence page. parentID may be empty for a root-level page.
func (c *confluenceWiki) createPage(parentID, title, storageBody string) (string, error) {
	type ancestor struct {
		ID string `json:"id"`
	}
	type storage struct {
		Value          string `json:"value"`
		Representation string `json:"representation"`
	}
	type body struct {
		Storage storage `json:"storage"`
	}
	type space struct {
		Key string `json:"key"`
	}

	payload := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": space{Key: c.spaceKey},
		"body": body{
			Storage: storage{Value: storageBody, Representation: "storage"},
		},
	}
	if parentID != "" {
		payload["ancestors"] = []ancestor{{ID: parentID}}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/rest/api/content", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("confluence API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// updatePage updates an existing Confluence page.
func (c *confluenceWiki) updatePage(pageID, title, storageBody string, version int) error {
	type storage struct {
		Value          string `json:"value"`
		Representation string `json:"representation"`
	}
	type body struct {
		Storage storage `json:"storage"`
	}
	type ver struct {
		Number int `json:"number"`
	}

	payload := map[string]interface{}{
		"version": ver{Number: version},
		"title":   title,
		"type":    "page",
		"body": body{
			Storage: storage{Value: storageBody, Representation: "storage"},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/rest/api/content/%s", c.baseURL, pageID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("confluence API returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// applyLabels adds Hero labels to a Confluence page.
func (c *confluenceWiki) applyLabels(pageID string, s *spec.Spec) error {
	labels := []map[string]string{
		{"prefix": "global", "name": c.labelPrefix + string(s.Type)},
		{"prefix": "global", "name": c.labelPrefix + "managed"},
	}
	for _, tag := range s.Tags {
		if tag != "" {
			labels = append(labels, map[string]string{"prefix": "global", "name": c.labelPrefix + tag})
		}
	}

	data, err := json.Marshal(labels)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/rest/api/content/%s/label", c.baseURL, pageID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *confluenceWiki) setHeaders(req *http.Request) {
	if c.userEmail != "" {
		// Confluence Cloud: basic auth with email + token
		req.SetBasicAuth(c.userEmail, c.token)
	} else {
		// Confluence Server/DC: Bearer token (PAT)
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// ---------------------------------------------------------------------------
// Content conversion helpers
// ---------------------------------------------------------------------------

// specToStorageFormat converts a Hero spec to Confluence storage format (XHTML).
func specToStorageFormat(s *spec.Spec, rawContent string) string {
	body := stripFrontmatter(rawContent)

	var sb strings.Builder

	// Managed notice
	sb.WriteString(fmt.Sprintf(
		`<ac:structured-macro ac:name="info"><ac:rich-text-body><p>Managed by Hero — spec: %s | type: %s | status: %s</p></ac:rich-text-body></ac:structured-macro>`,
		s.Slug, s.Type, s.Status))

	if s.ClaimedBy != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>Assigned:</strong> %s</p>", htmlEscape(s.ClaimedBy)))
	}
	if len(s.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("<p><strong>Tags:</strong> %s</p>", htmlEscape(strings.Join(s.Tags, ", "))))
	}

	// Convert markdown body to a simple XHTML representation.
	// This is not a full markdown parser — it handles the most common Hero spec patterns.
	sb.WriteString(markdownToStorage(body))

	return sb.String()
}

// markdownToStorage converts basic markdown to Confluence storage XHTML.
// Handles: headings (##/###), bold, code blocks, bullet lists, paragraphs.
func markdownToStorage(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	inCode := false
	inList := false

	for _, line := range lines {
		// Code fence
		if strings.HasPrefix(line, "```") {
			if !inCode {
				lang := strings.TrimPrefix(line, "```")
				if lang == "" {
					lang = "none"
				}
				if inList {
					sb.WriteString("</ul>")
					inList = false
				}
				sb.WriteString(fmt.Sprintf(`<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">%s</ac:parameter><ac:plain-text-body><![CDATA[`, htmlEscape(lang)))
				inCode = true
			} else {
				sb.WriteString("]]></ac:plain-text-body></ac:structured-macro>")
				inCode = false
			}
			continue
		}

		if inCode {
			sb.WriteString(line + "\n")
			continue
		}

		// Close list if needed
		if inList && !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			sb.WriteString("</ul>")
			inList = false
		}

		switch {
		case strings.HasPrefix(line, "#### "):
			sb.WriteString(fmt.Sprintf("<h4>%s</h4>", htmlEscape(strings.TrimPrefix(line, "#### "))))
		case strings.HasPrefix(line, "### "):
			sb.WriteString(fmt.Sprintf("<h3>%s</h3>", htmlEscape(strings.TrimPrefix(line, "### "))))
		case strings.HasPrefix(line, "## "):
			sb.WriteString(fmt.Sprintf("<h2>%s</h2>", htmlEscape(strings.TrimPrefix(line, "## "))))
		case strings.HasPrefix(line, "# "):
			sb.WriteString(fmt.Sprintf("<h1>%s</h1>", htmlEscape(strings.TrimPrefix(line, "# "))))
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			if !inList {
				sb.WriteString("<ul>")
				inList = true
			}
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			sb.WriteString(fmt.Sprintf("<li>%s</li>", inlineMarkdown(item)))
		case strings.TrimSpace(line) == "---" || strings.TrimSpace(line) == "***":
			sb.WriteString("<hr/>")
		case strings.TrimSpace(line) == "":
			// paragraph break — skip empty lines
		default:
			sb.WriteString(fmt.Sprintf("<p>%s</p>", inlineMarkdown(line)))
		}
	}

	if inList {
		sb.WriteString("</ul>")
	}

	return sb.String()
}

// inlineMarkdown converts inline markdown (bold, code, italic) to XHTML.
func inlineMarkdown(s string) string {
	// Bold: **text**
	s = replaceInline(s, "**", "<strong>", "</strong>")
	// Italic: *text* (single)
	s = replaceInline(s, "*", "<em>", "</em>")
	// Inline code: `text`
	s = replaceInline(s, "`", "<code>", "</code>")
	return s
}

// replaceInline replaces pairs of delimiters with open/close tags.
func replaceInline(s, delim, open, close string) string {
	parts := strings.Split(s, delim)
	if len(parts) < 3 {
		return s
	}
	var sb strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			sb.WriteString(open)
			sb.WriteString(htmlEscape(part))
			sb.WriteString(close)
		} else {
			sb.WriteString(part)
		}
	}
	return sb.String()
}

// htmlEscape escapes HTML special characters.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// urlEncode performs minimal percent-encoding for query string values.
func urlEncode(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '~':
			sb.WriteRune(r)
		case r == ' ':
			sb.WriteString("+")
		default:
			sb.WriteString(fmt.Sprintf("%%%02X", r))
		}
	}
	return sb.String()
}
