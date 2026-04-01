package arch

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nextlevelbuilder/goclaw/internal/tools"
)

// ArchTool allows agents to map the architecture of their workspace.
type ArchTool struct {
	workspace string
	restrict  bool
}

func NewArchTool(workspace string, restrict bool) *ArchTool {
	return &ArchTool{
		workspace: workspace,
		restrict:  restrict,
	}
}

func (t *ArchTool) Name() string {
	return "analyze_architecture"
}

func (t *ArchTool) Description() string {
	return "Scan the workspace to map the project architecture, identifying roots (Go, Node, Flutter, Frappe), entry points, and configuration files."
}

func (t *ArchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Optional relative path within the workspace to scan. Defaults to workspace root.",
			},
		},
	}
}

func (t *ArchTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path := "."
	if p, ok := args["path"].(string); ok {
		path = p
	}

	// Resolve the real workspace path from context if possible
	ws := t.workspace
	if ctxWs := tools.ToolWorkspaceFromCtx(ctx); ctxWs != "" {
		ws = ctxWs
	}

	target := filepath.Join(ws, path)
	
	// Map the workspace
	node, err := MapWorkspace(target)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to map architecture: %v", err))
	}

	// Format results for the LLM
	data, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return tools.ErrorResult("failed to format architecture results")
	}

	return &tools.Result{
		ForLLM: string(data),
		ForUser: fmt.Sprintf("Architectural map of %q generated.", path),
	}
}
