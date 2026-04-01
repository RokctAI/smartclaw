package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// SearchCodeTool allows the agent to search large codebases without reading full files.
// It uses ripgrep (if available) or standard grep/findstr under the hood.
type SearchCodeTool struct {
	workspace string
}

func NewSearchCodeTool(workspace string) *SearchCodeTool {
	return &SearchCodeTool{workspace: workspace}
}

func (t *SearchCodeTool) Name() string { return "search_code" }
func (t *SearchCodeTool) Description() string {
	return "Search the codebase for specific keywords, variable names, or function definitions without reading full files. Use this BEFORE reading files to find exact line numbers and save tokens."
}
func (t *SearchCodeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The exact string or regex pattern to search for (e.g., 'def create_invoice')",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional: directory or file to constrain the search (default is workspace root)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchCodeTool) Execute(ctx context.Context, args map[string]any) *Result {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query is required")
	}

	searchPath := t.workspace
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = filepath.Clean(filepath.Join(t.workspace, p))
	}

	// Verify path is within workspace
	absWorkspace, _ := filepath.Abs(t.workspace)
	absSearchPath, _ := filepath.Abs(searchPath)
	if !strings.HasPrefix(absSearchPath, absWorkspace) {
		return ErrorResult("search path cannot escape workspace")
	}

	if _, err := os.Stat(absSearchPath); os.IsNotExist(err) {
		return ErrorResult(fmt.Sprintf("path does not exist: %s", searchPath))
	}

	// Simple fallback search using grep (Linux/Mac) or findstr (Windows).
	// In a production setup, ripgrep (rg) is preferred.
	var cmd *exec.Cmd
	// Windows fallback (since user relies on powershell)
	cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", 
		fmt.Sprintf("Select-String -Path '%s\\*' -Pattern '%s' -Recurse -CaseSensitive:$false | Select-Object LineNumber, Filename, Line | ConvertTo-Json -Compress", absSearchPath, query))
	
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		if strings.Contains(outStr, "ObjectNotFound") || len(strings.TrimSpace(outStr)) == 0 {
			return SilentResult(fmt.Sprintf("No matches found for '%s' in %s", query, searchPath))
		}
		return ErrorResult(fmt.Sprintf("Search failed: %v\nOutput: %s", err, cutAtLastNewline(outStr[:min(len(outStr), 1000)])))
	}

	// Cap output to 8,000 characters to prevent token explosion on generic queries like 'import'.
	if utf8.RuneCountInString(outStr) > 8000 {
		outStr = cutAtLastNewline(outStr[:8000]) + "\n\n[Warning: Results truncated. Please refine your query to be more specific.]"
	}

	return SilentResult(fmt.Sprintf("Search Results:\n%s", outStr))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
