package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	getenv = os.Getenv
}

// RunConfig configures a headless agent run.
type RunConfig struct {
	ProjectRoot string
	HeroDir     string
	Provider    string
	Model       string
	APIKey      string
	Command     string // e.g. "deliver", "diagnose"
	Args        string // e.g. spec slug or natural language
	MaxTurns    int
	Budget      float64 // max cost in dollars
	DryRun      bool

	// InlinePropose flips the agent into propose-mode: the system
	// prompt instructs the agent to emit HERO-PROPOSAL: NDJSON lines
	// on stdout instead of writing to disk. The wrapping shim
	// (`hero agent propose-shim`) tails stdout and forwards each
	// proposal to the daemon. See docs/contracts/inline-propose-v1.md.
	InlinePropose bool
}

// JobRecord tracks a completed or in-progress run.
type JobRecord struct {
	ID          string    `json:"id"`
	Command     string    `json:"command"`
	Args        string    `json:"args"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Status      string    `json:"status"` // running, completed, failed, budget_exceeded, turn_limit
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Turns       int       `json:"turns"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	EstCost     float64   `json:"est_cost"`
	Error       string    `json:"error,omitempty"`
}

// Run executes the agent loop headlessly.
func Run(cfg RunConfig) (*JobRecord, error) {
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 100
	}

	// Resolve provider
	providerName := cfg.Provider
	if providerName == "" {
		providerName = DetectProvider(cfg.Model)
	}

	// Build system prompt
	systemPrompt := buildSystemPrompt(cfg)

	// Set up tools
	executor := NewToolExecutor(cfg.ProjectRoot, cfg.HeroDir)
	tools := executor.ToolDefinitions()

	// Job record
	job := &JobRecord{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Command:   cfg.Command,
		Args:      cfg.Args,
		Provider:  providerName,
		Model:     cfg.Model,
		Status:    "running",
		StartedAt: time.Now(),
	}

	if cfg.DryRun {
		job.Status = "dry_run"
		fmt.Printf("Dry run: %s %s\n", cfg.Command, cfg.Args)
		fmt.Printf("Provider: %s, Model: %s\n", providerName, cfg.Model)
		fmt.Printf("Max turns: %d, Budget: $%.2f\n", cfg.MaxTurns, cfg.Budget)
		fmt.Printf("System prompt: %d chars\n", len(systemPrompt))
		fmt.Printf("Tools: %d registered\n", len(tools))
		return job, nil
	}

	apiKey := ResolveAPIKey(providerName, cfg.APIKey)
	if apiKey == "" {
		envVar := map[string]string{"anthropic": "ANTHROPIC_API_KEY", "openai": "OPENAI_API_KEY", "azure": "AZURE_OPENAI_KEY"}[providerName]
		return nil, fmt.Errorf("no API key found for provider %q — set %s or use --api-key", providerName, envVar)
	}

	provider, err := GetProvider(providerName, apiKey, nil)
	if err != nil {
		return nil, err
	}

	// Build initial message
	userMessage := buildUserMessage(cfg)

	fmt.Printf("Running: %s %s\n", cfg.Command, cfg.Args)
	fmt.Printf("Provider: %s, Model: %s\n", providerName, cfg.Model)
	fmt.Printf("Max turns: %d, Budget: $%.2f\n", cfg.MaxTurns, cfg.Budget)
	fmt.Println()

	// Agent loop
	messages := []Message{{Role: "user", Content: userMessage}}
	ctx := context.Background()

	for turn := 0; turn < cfg.MaxTurns; turn++ {
		job.Turns = turn + 1

		resp, err := provider.Chat(ctx, ChatRequest{
			Model:     cfg.Model,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8192,
		})
		if err != nil {
			job.Status = "failed"
			job.Error = err.Error()
			job.CompletedAt = time.Now()
			saveJob(cfg.HeroDir, job)
			return job, fmt.Errorf("turn %d: %w", turn+1, err)
		}

		job.InputTokens += resp.InputTokens
		job.OutputTokens += resp.OutputTokens
		job.EstCost = estimateCost(providerName, job.InputTokens, job.OutputTokens)

		// Print assistant text
		if resp.Text != "" {
			fmt.Printf("[turn %d] %s\n", turn+1, truncateLog(resp.Text, 200))
		}

		// Check if done
		if resp.StopReason == "end_turn" || resp.StopReason == "stop" {
			if len(resp.ToolCalls) == 0 {
				job.Status = "completed"
				break
			}
		}

		// Execute tool calls
		if len(resp.ToolCalls) > 0 {
			// Add assistant message with tool calls
			messages = append(messages, Message{Role: "assistant", Content: resp.Text})

			for _, tc := range resp.ToolCalls {
				fmt.Printf("[turn %d] tool: %s\n", turn+1, tc.Name)
				result := executor.Execute(tc)

				// Add tool result as user message (Anthropic format)
				messages = append(messages, Message{
					Role: "user",
					Content: []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": tc.ID,
							"content":     truncateLog(result, 5000),
						},
					},
				})
			}
		} else {
			messages = append(messages, Message{Role: "assistant", Content: resp.Text})
			job.Status = "completed"
			break
		}

		// Budget check
		if cfg.Budget > 0 && job.EstCost > cfg.Budget {
			fmt.Printf("\nBudget exceeded: $%.2f > $%.2f\n", job.EstCost, cfg.Budget)
			job.Status = "budget_exceeded"
			break
		}
	}

	if job.Status == "running" {
		job.Status = "turn_limit"
	}

	job.CompletedAt = time.Now()
	saveJob(cfg.HeroDir, job)

	fmt.Printf("\n✓ %s in %d turns ($%.2f)\n", job.Status, job.Turns, job.EstCost)
	return job, nil
}

func buildSystemPrompt(cfg RunConfig) string {
	var sb strings.Builder

	// Load AGENTS.md if it exists
	agentsPath := filepath.Join(cfg.ProjectRoot, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	// Load the agent definition for the command
	agentFile := ""
	switch cfg.Command {
	case "deliver":
		agentFile = "feature-delivery-lead.md"
	case "diagnose":
		agentFile = "feature-delivery-lead.md"
	case "design":
		agentFile = "feature-delivery-lead.md"
	}
	if agentFile != "" {
		agentPath := filepath.Join(cfg.ProjectRoot, "agents", agentFile)
		if data, err := os.ReadFile(agentPath); err == nil {
			sb.WriteString("## Agent Role\n\n")
			sb.Write(data)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("You are running in headless mode via `hero run`. ")
	sb.WriteString("There is no interactive user — execute the task to completion. ")
	sb.WriteString("Use tools to read files, write code, run tests, and commit. ")
	sb.WriteString("Be thorough but efficient.\n")

	if cfg.InlinePropose {
		sb.WriteString("\n")
		sb.WriteString(InlineProposeAddendum)
		sb.WriteString("\n")
	}

	return sb.String()
}

// InlineProposeAddendum is the agent-prompt addendum that turns
// write-to-disk output into stdout HERO-PROPOSAL: NDJSON lines.
// Authored to match docs/contracts/inline-propose-v1.md §1–§2.
const InlineProposeAddendum = `## Output mode: inline-propose

You are running under --inline-propose. Do NOT write directly to spec
files for the artifact content you would normally author. Instead,
emit each unit of proposed content as a single line on stdout in
this exact shape:

    HERO-PROPOSAL: {"schema_version":"1.0","proposal_id":"p-<6-hex>","batch_id":"b-<6-hex>","session_id":"<session>","agent":"<your-slug>","target":{"spec_slug":"<slug>","anchor":{"kind":"section","value":"<section-id>","position":"append"}},"content":{"format":"markdown","body":"..."},"rationale":"..."}

Rules:
- One proposal per line, ASCII prefix HERO-PROPOSAL: (with a trailing space).
- The line MUST be valid JSON after the prefix. No multi-line JSON.
- proposal_id is unique within the session. batch_id groups proposals
  that should bulk-accept together (one batch_id per "draft N AC"
  invocation).
- target.anchor.kind is one of: frontmatter, section, heading,
  list_item, free. target.anchor.value identifies the anchor within
  the spec (e.g. the section heading slug or the frontmatter field
  name).
- content.format is markdown unless the target anchor is frontmatter
  (then yaml) or a free-text label (then text).
- session_id comes from the HERO_SESSION_ID environment variable. If
  unset, the wrapping shim will fill it in.
- After emitting your proposals, finish your turn. The dashboard
  surfaces accept / edit / reject controls; you do NOT apply the
  changes yourself. Disk writes for proposed content are reserved
  for the user's accept action.

Non-proposal stdout (status messages, progress, etc.) is passed
through to the user and is fine to emit.`

func buildUserMessage(cfg RunConfig) string {
	switch cfg.Command {
	case "deliver":
		return fmt.Sprintf("Deliver the spec: %s\n\nRead the spec, implement the changes, run tests, and commit. Use --autopilot mode.", cfg.Args)
	case "diagnose":
		return fmt.Sprintf("Diagnose this bug: %s\n\nInvestigate, identify root cause, and write the findings into the spec.", cfg.Args)
	case "design":
		return fmt.Sprintf("Design a spec for: %s\n\nWrite a complete spec with goal, design, changes, and acceptance criteria.", cfg.Args)
	default:
		return cfg.Args
	}
}

func estimateCost(provider string, inputTokens, outputTokens int) float64 {
	// Rough estimates per 1M tokens
	switch provider {
	case "anthropic":
		return float64(inputTokens)*3.0/1_000_000 + float64(outputTokens)*15.0/1_000_000
	case "openai":
		return float64(inputTokens)*2.5/1_000_000 + float64(outputTokens)*10.0/1_000_000
	default:
		return float64(inputTokens)*3.0/1_000_000 + float64(outputTokens)*15.0/1_000_000
	}
}

func saveJob(heroDir string, job *JobRecord) {
	jobsDir := filepath.Join(heroDir, "jobs")
	os.MkdirAll(jobsDir, 0o755)
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(jobsDir, job.ID+".json"), data, 0o644)
}

// ListJobs returns recent job records.
func ListJobs(heroDir string, limit int) ([]*JobRecord, error) {
	jobsDir := filepath.Join(heroDir, "jobs")
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var jobs []*JobRecord
	// Read in reverse order (most recent first)
	for i := len(entries) - 1; i >= 0 && len(jobs) < limit; i-- {
		e := entries[i]
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(jobsDir, e.Name()))
		if err != nil {
			continue
		}
		var job JobRecord
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

func truncateLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
