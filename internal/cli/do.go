package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var doCmd = &cobra.Command{
	Use:   "do <natural language request...>",
	Short: "Route a natural-language request to the right Hero workflow",
	Long: `Analyzes your request and suggests the appropriate Hero command or slash command.

This is the "I don't know which command to use" entry point. Describe what you
want to do in plain English, and Hero will tell you which workflow to run.

Examples:
  hero do fix the login bug
  hero do design a new auth system
  hero do check if my specs are healthy
  hero do scan this project
  hero do capture my thoughts on caching
  hero do what conventions do we follow`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDo,
}

// route represents a matched command route.
type route struct {
	Command     string // CLI command or slash command
	Description string
	Score       int // match quality (higher = better)
}

// routeRule defines a keyword-based routing rule.
type routeRule struct {
	keywords    []string // any of these keywords trigger the rule
	phrases     []string // multi-word phrases (higher weight)
	command     string
	description string
}

var routeRules = []routeRule{
	// Bug/diagnosis workflows
	{
		keywords:    []string{"bug", "fix", "broken", "error", "crash", "fail", "wrong", "issue", "debug", "investigate", "diagnose", "regression", "exception"},
		phrases:     []string{"root cause", "not working", "doesn't work", "stack trace"},
		command:     "/diagnose",
		description: "Investigate the bug, find the root cause, and produce a fix spec",
	},
	// Design/feature workflows
	{
		keywords:    []string{"design", "feature", "build", "create", "implement", "add", "new feature", "enhance", "improve"},
		phrases:     []string{"new feature", "design a", "build a", "add a", "implement a"},
		command:     "/design",
		description: "Produce a spec for the feature with goal, design, and acceptance criteria",
	},
	// Delivery workflows
	{
		keywords:    []string{"deliver", "execute", "ship", "code", "implement"},
		phrases:     []string{"write the code", "implement the", "deliver the", "execute the spec", "ship it"},
		command:     "/deliver",
		description: "Execute an approved spec — implement, test, and complete the work",
	},
	// Review workflows
	{
		keywords:    []string{"review", "pr", "pull request", "code review"},
		phrases:     []string{"review the", "look at the pr", "code review", "pull request"},
		command:     "/review",
		description: "Review code, PRs, security posture, or architecture decisions",
	},
	// Convention workflows
	{
		keywords:    []string{"convention", "pattern", "standard", "guideline", "style"},
		phrases:     []string{"coding convention", "what conventions", "style guide", "coding standard"},
		command:     "/convention",
		description: "Document a codebase pattern as a convention spec",
	},
	// Decision workflows
	{
		keywords:    []string{"decide", "decision", "tradeoff", "evaluate", "compare", "choose"},
		phrases:     []string{"should we", "versus", "tradeoff", "compare options", "evaluate alternatives"},
		command:     "/decide",
		description: "Evaluate an architectural decision with structured analysis",
	},
	// Discovery/ideation workflows
	{
		keywords:    []string{"discover", "brainstorm", "explore", "ideate", "roadmap", "prioritize"},
		phrases:     []string{"product direction", "feature ideas", "what should we build"},
		command:     "/discover",
		description: "Explore product direction, brainstorm features, and prioritize",
	},
	// Documentation workflows
	{
		keywords:    []string{"document", "docs", "readme", "documentation", "explain"},
		phrases:     []string{"write docs", "update docs", "document the", "create documentation"},
		command:     "/docs",
		description: "Create or update technical documentation and project context",
	},
	// Release workflows
	{
		keywords:    []string{"release", "deploy", "version", "changelog", "tag"},
		phrases:     []string{"cut a release", "release readiness", "prepare release", "deployment plan"},
		command:     "/release",
		description: "Assess release readiness, deployment concerns, and operational risk",
	},
	// Compose/planning workflows
	{
		keywords:    []string{"compose", "plan", "break down", "decompose", "epic", "initiative"},
		phrases:     []string{"break down", "split into", "compose a plan", "break this into"},
		command:     "/compose",
		description: "Break a large initiative into sequenced specs with a delivery plan",
	},
	// Retro workflows
	{
		keywords:    []string{"retro", "retrospective", "postmortem", "reflection", "lessons"},
		phrases:     []string{"what went well", "lessons learned", "post-delivery", "how did it go"},
		command:     "/retro",
		description: "Run a post-delivery retrospective comparing spec vs actual",
	},
	// Note workflows
	{
		keywords:    []string{"note", "capture", "thought", "jot", "remember", "save"},
		phrases:     []string{"capture this", "save this", "jot down", "take a note", "write down"},
		command:     "/note (or: hero note)",
		description: "Capture the current thinking as a note in the knowledge base",
	},
	// Scan workflows
	{
		keywords:    []string{"scan", "detect", "stack", "onboard", "analyze"},
		phrases:     []string{"scan the project", "detect the stack", "what tech", "technology stack", "analyze the codebase"},
		command:     "hero scan",
		description: "Analyze the codebase and generate initial knowledge base entries",
	},
	// Check/health workflows
	{
		keywords:    []string{"check", "health", "stale", "hygiene", "validate", "lint"},
		phrases:     []string{"health check", "are my specs", "stale specs", "workspace health"},
		command:     "hero check",
		description: "Run a workspace health check — conventions, stale specs, hygiene",
	},
	// Search workflows
	{
		keywords:    []string{"search", "find", "look up", "query", "where"},
		phrases:     []string{"find the spec", "search for", "look up", "where is"},
		command:     "hero search <query>",
		description: "Search the spec corpus for matching specs",
	},
	// Status/dashboard workflows
	{
		keywords:    []string{"status", "dashboard", "overview", "summary", "progress"},
		phrases:     []string{"what's the status", "show me the dashboard", "project overview", "how are we doing"},
		command:     "hero dashboard",
		description: "Show a rich terminal summary of the workspace state",
	},
	// Knowledge workflows
	{
		keywords:    []string{"knowledge", "conventions", "decisions", "rules", "context"},
		phrases:     []string{"knowledge base", "what do we know", "list conventions", "list decisions"},
		command:     "hero knowledge",
		description: "List and browse the knowledge base entries",
	},
	// Mock/prototype workflows
	{
		keywords:    []string{"mock", "mockup", "prototype", "wireframe", "ui", "ux", "visual", "layout", "screen"},
		phrases:     []string{"mock up", "create a mock", "design the ui", "visual prototype", "what it looks like"},
		command:     "/mock",
		description: "Generate a visual HTML mockup from a spec or description",
	},
	// Split/decompose workflows
	{
		keywords:    []string{"split", "decompose", "break", "smaller", "child", "sub-spec"},
		phrases:     []string{"break down", "split into", "too big", "break this spec", "decompose this"},
		command:     "/split",
		description: "Break a large spec into smaller, independently deliverable child specs",
	},
	// Sprint planning workflows
	{
		keywords:    []string{"sprint", "iteration", "batch", "next sprint", "capacity"},
		phrases:     []string{"plan a sprint", "next sprint", "sprint planning", "what should we work on"},
		command:     "/sprint",
		description: "Plan a sprint by selecting and sequencing specs from the backlog",
	},
	// Cost/effort estimation workflows
	{
		keywords:    []string{"cost", "estimate", "effort", "size", "complexity", "points", "how big", "how long"},
		phrases:     []string{"how big is", "how much effort", "estimate the", "how long will", "effort estimate", "size this"},
		command:     "hero cost",
		description: "Estimate effort for a spec based on complexity signals",
	},
	// Replay/post-mortem workflows
	{
		keywords:    []string{"replay", "postmortem", "aftermath", "compare", "accuracy"},
		phrases:     []string{"what actually happened", "spec vs actual", "how accurate", "post-mortem replay"},
		command:     "hero replay",
		description: "Compare spec plan vs actual outcome for post-delivery analysis",
	},
	// Knowledge capture workflows
	{
		keywords:    []string{"capture", "learned", "learning", "knowledge", "reflect", "takeaway", "insight"},
		phrases:     []string{"what did we learn", "capture learnings", "save knowledge", "what did I learn", "extract knowledge"},
		command:     "/capture",
		description: "Extract and persist learnings from the current session into the knowledge base",
	},
}

func runDo(cmd *cobra.Command, args []string) error {
	input := strings.ToLower(strings.Join(args, " "))

	matches := matchRoutes(input)

	if len(matches) == 0 {
		fmt.Println("No matching workflow found for your request.")
		fmt.Println()
		fmt.Println("Try being more specific, or use one of these commands:")
		fmt.Println("  hero dashboard    — workspace overview")
		fmt.Println("  hero search       — find specs")
		fmt.Println("  hero check        — workspace health")
		fmt.Println()
		fmt.Println("For agent-powered workflows, use these slash commands in your AI tool:")
		fmt.Println("  /design           — design a feature")
		fmt.Println("  /diagnose         — investigate a bug")
		fmt.Println("  /deliver          — implement a spec")
		fmt.Println("  /review           — review code or PRs")
		return nil
	}

	fmt.Printf("Based on: %q\n\n", strings.Join(args, " "))

	if len(matches) == 1 {
		m := matches[0]
		fmt.Printf("Recommended:  %s\n", m.Command)
		fmt.Printf("              %s\n", m.Description)
	} else {
		fmt.Println("Matching workflows (best match first):")
		fmt.Println()
		for i, m := range matches {
			marker := "  "
			if i == 0 {
				marker = "→ "
			}
			fmt.Printf("%s%-28s %s\n", marker, m.Command, m.Description)
		}
	}

	return nil
}

// matchRoutes finds routes that match the input, sorted by score descending.
func matchRoutes(input string) []route {
	var matches []route

	words := strings.Fields(input)

	for _, rule := range routeRules {
		score := 0

		// Check phrase matches (higher weight)
		for _, phrase := range rule.phrases {
			if strings.Contains(input, phrase) {
				score += 3
			}
		}

		// Check keyword matches
		for _, kw := range rule.keywords {
			for _, word := range words {
				if word == kw {
					score += 1
				}
			}
			// Also check if multi-word keyword is a substring
			if strings.Contains(kw, " ") && strings.Contains(input, kw) {
				score += 2
			}
		}

		if score > 0 {
			matches = append(matches, route{
				Command:     rule.command,
				Description: rule.description,
				Score:       score,
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Limit to top 3
	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}
