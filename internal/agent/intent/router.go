package intent

import (
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/tools"
)

// Intent represents the high-level goal of the user's prompt.
type Intent string

const (
	IntentProjectCreation Intent = "PROJECT_CREATION"
	IntentDebugging       Intent = "DEBUGGING"
	IntentRefactoring     Intent = "REFACTORING"
	IntentMapping         Intent = "ARCHITECTURAL_MAPPING"
	IntentGeneral         Intent = "GENERAL"
)

// ParsedIntent contains the pre-flight classification results and the 
// specific tools the agent should be restricted to for this intent.
type ParsedIntent struct {
	PrimaryIntent Intent
	Entities      map[string]string // e.g. "framework": "flutter", "repo": "rokctai/checkin"
	AllowedTools  []string          // specific tool constraints
}

// Router provides native heuristic and (future ML/Transformers) pre-flight 
// intent classification before the main GoClaw LLM boots up.
type Router struct{}

func NewRouter() *Router {
	return &Router{}
}

// Classify takes the raw user prompt and returns a pure ParsedIntent.
// This acts as GoClaw's native "Pre-Flight" routing engine to prevent tool bloat.
func (r *Router) Classify(prompt string) ParsedIntent {
	lowerPrompt := strings.ToLower(prompt)
	parsed := ParsedIntent{
		PrimaryIntent: IntentGeneral,
		Entities:      make(map[string]string),
		AllowedTools:  []string{},
	}

	// 1. Detect Framework Entities
	if strings.Contains(lowerPrompt, "flutter") {
		parsed.Entities["framework"] = "flutter"
	} else if strings.Contains(lowerPrompt, "python") || strings.Contains(lowerPrompt, "frappe") {
		parsed.Entities["framework"] = "frappe"
	} else if strings.Contains(lowerPrompt, "go ") || strings.Contains(lowerPrompt, "golang") {
		parsed.Entities["framework"] = "golang"
	}

	// 2. Detect Intent & Assign Custom Modes
	if strings.Contains(lowerPrompt, "create") || strings.Contains(lowerPrompt, "clone") || strings.Contains(lowerPrompt, "initialize") {
		parsed.PrimaryIntent = IntentProjectCreation
		
		// For project creation, restrict the LLM to architect tools to prevent random file editing.
		parsed.AllowedTools = []string{
			"git_clone",
			"exec",
			"analyze_architecture",
			"browser",
			"tool_search",
		}
		
	} else if strings.Contains(lowerPrompt, "fix") || strings.Contains(lowerPrompt, "bug") || strings.Contains(lowerPrompt, "error") {
		parsed.PrimaryIntent = IntentDebugging
		
		// For debugging, restrict to Search, Read, and Structural Edit
		parsed.AllowedTools = []string{
			"search_code",
			"read_file",
			"edit_structural",
			"exec",
		}
		
	} else if strings.Contains(lowerPrompt, "map") || strings.Contains(lowerPrompt, "audit") || strings.Contains(lowerPrompt, "analyze") {
		parsed.PrimaryIntent = IntentMapping
		
		parsed.AllowedTools = []string{
			"analyze_architecture",
			"read_file",
			"search_code",
		}
	} else {
		// General fallback: Full access (or restricted based on default config)
		parsed.PrimaryIntent = IntentGeneral
	}

	return parsed
}

// ApplyConstraints forces the GoClaw tool registry to ONLY expose the AllowedTools.
func (p *ParsedIntent) ApplyConstraints(registry *tools.Registry) {
	if len(p.AllowedTools) == 0 {
		return // No constraints, allow all fallback tools
	}

	// In a real execution, we would iterate through the master registry and 
	// disable any tool not in p.AllowedTools, ensuring the LLM is perfectly
	// locked into its Custom Mode for token optimization.
}
