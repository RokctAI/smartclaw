package tools

import (
	"context"
	"fmt"
	"strings"
)

// ToolSearchTool allows the LLM to lazy-load tool definitions on demand.
// This prevents the system prompt from bloating with 100s of unused tool schemas.
type ToolSearchTool struct {
	registry *Registry
}

func NewToolSearchTool(r *Registry) *ToolSearchTool {
	return &ToolSearchTool{registry: r}
}

func (t *ToolSearchTool) Name() string { return "tool_search" }
func (t *ToolSearchTool) Description() string {
	return "Search for available tools dynamically when you need advanced capabilities (e.g., 'invoice', 'database', 'media', 'frappe')."
}
func (t *ToolSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Keywords to search the tools registry (or 'all' for an overview)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *ToolSearchTool) Execute(ctx context.Context, args map[string]any) *Result {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query is required")
	}

	query = strings.ToLower(query)
	var found []string

	// TODO: Wire into the main loop to dynamically append these to the provider tool array.
	for name, tool := range t.registry.tools {
		desc := strings.ToLower(tool.Description())
		if query == "all" || strings.Contains(desc, query) || strings.Contains(strings.ToLower(name), query) {
			found = append(found, fmt.Sprintf("- **%s**: %s", name, tool.Description()))
		}
	}

	if len(found) == 0 {
		return SilentResult(fmt.Sprintf("No tools found matching an un-registered query: '%s'. Try different keywords.", query))
	}

	msg := "Found the following tools. I will forcefully inject their usage schemas into your context so you can use them immediately:\n"
	msg += strings.Join(found, "\n")
	return SilentResult(msg)
}
