package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StructuralEditTool implements "MorphLLM" style edits. 
// Instead of matching a single string on 1 line (which is fragile to whitespace),
// it matches a block of lines and replaces them. This allows the agent to safely
// replace entire functions or structs without recreating the entire file.
type StructuralEditTool struct {
	workspace string
}

func NewStructuralEditTool(workspace string) *StructuralEditTool {
	return &StructuralEditTool{workspace: workspace}
}

func (t *StructuralEditTool) Name() string { return "edit_structural" }
func (t *StructuralEditTool) Description() string {
	return "Surgically edit code using 'Morph' Search and Replace blocks. Use this for multi-line edits (e.g., replacing a whole function). The 'search_block' must EXACTLY match the lines in the file, including indentation, to succeed."
}

func (t *StructuralEditTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path relative to the workspace",
			},
			"search_block": map[string]any{
				"type":        "string",
				"description": "The exact multi-line block of code you want to replace. Must include the exact indentation.",
			},
			"replace_block": map[string]any{
				"type":        "string",
				"description": "The new multi-line code block to inject in its place.",
			},
		},
		"required": []string{"path", "search_block", "replace_block"},
	}
}

func (t *StructuralEditTool) Execute(ctx context.Context, args map[string]any) *Result {
	path, _ := args["path"].(string)
	searchBlock, _ := args["search_block"].(string)
	replaceBlock, _ := args["replace_block"].(string)

	if path == "" || searchBlock == "" {
		return ErrorResult("path and search_block are required")
	}

	absPath, _ := filepath.Abs(filepath.Join(t.workspace, path))
	absWorkspace, _ := filepath.Abs(t.workspace)
	if !strings.HasPrefix(absPath, absWorkspace) {
		return ErrorResult("Cannot edit files outside the workspace boundary")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to read file: %v", err))
	}

	content := string(data)

	// In the future, this is where we inject the Tree-sitter AST parsing logic.
	// For Phase 1, we execute a strict string-block match.
	count := strings.Count(content, searchBlock)
	
	if count == 0 {
		return ErrorResult("Morph Match Failed: The search_block was not found. This is usually due to mismatched indentation or missing newlines. Use 'read_file' to view the exact characters and try again.")
	}

	if count > 1 {
		return ErrorResult(fmt.Sprintf("Morph Match Failed: The search_block was found %d times. Your block must be unique (add more surrounding context lines) so I know which one to replace.", count))
	}

	newContent := strings.Replace(content, searchBlock, replaceBlock, 1)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to create directory structure: %v", err))
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to write edited file: %v", err))
	}

	// Semantic Edit Verification (Pre-Flight Safety Check)
	// We must physically read the file back from the OS and mathematically prove 
	// the `replaceBlock` is present to ensure no silent truncation or IO failures occurred.
	verifyData, verifyErr := os.ReadFile(absPath)
	if verifyErr != nil {
		return ErrorResult(fmt.Sprintf("CRITICAL Verification Error: Failed to read file back from disk after writing: %v", verifyErr))
	}
	
	if !strings.Contains(string(verifyData), replaceBlock) {
		return ErrorResult("CRITICAL Verification Error: The write operation succeeded, but the new code is NOT present when reading the file back. The structural edit failed silently. You must re-evaluate your assumptions.")
	}

	return SilentResult(fmt.Sprintf("Structural Edit successful and VERIFIED: File %s updated properly.", path))
}
