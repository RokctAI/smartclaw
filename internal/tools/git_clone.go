package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitCloneTool provides a deterministic, hallucination-free way to clone
// remote repositories into the GoClaw workspace on a VPS.
type GitCloneTool struct {
	workspace string
}

func NewGitCloneTool(workspace string) *GitCloneTool {
	return &GitCloneTool{workspace: workspace}
}

func (t *GitCloneTool) Name() string { return "git_clone" }
func (t *GitCloneTool) Description() string {
	return "Clone a remote git repository safely into the workspace. Use this when starting a new session on a project that does not exist locally."
}

func (t *GitCloneTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repository": map[string]any{
				"type":        "string",
				"description": "The repository identifier (e.g., 'owner/repo' or full 'https://github.com/...' URL)",
			},
			"branch": map[string]any{
				"type":        "string",
				"description": "Optional: A specific branch to check out after cloning",
			},
			"directory": map[string]any{
				"type":        "string",
				"description": "Optional: A specific sub-directory name to clone into within the workspace",
			},
		},
		"required": []string{"repository"},
	}
}

func (t *GitCloneTool) Execute(ctx context.Context, args map[string]any) *Result {
	repoRaw, _ := args["repository"].(string)
	branch, _ := args["branch"].(string)
	dir, _ := args["directory"].(string)

	if repoRaw == "" {
		return ErrorResult("repository is required")
	}

	// Smart URL construction so the LLM doesn't have to guess SSH vs HTTPS
	cloneURL := repoRaw
	if !strings.HasPrefix(repoRaw, "http") && !strings.HasPrefix(repoRaw, "git@") {
		// Default to HTTPS for public Github, or can be configured for SSH.
		cloneURL = fmt.Sprintf("https://github.com/%s.git", repoRaw)
	}

	// Determine absolute target directory
	targetDir := t.workspace
	if dir != "" {
		targetDir = filepath.Join(t.workspace, filepath.Clean(dir))
	} else if !strings.Contains(repoRaw, "://") && !strings.Contains(repoRaw, "git@") {
		// If owner/repo, use the repo name as the directory
		parts := strings.Split(repoRaw, "/")
		if len(parts) == 2 {
			targetDir = filepath.Join(t.workspace, parts[1])
		}
	}

	// Ensure target directory doesn't escape workspace
	absWorkspace, _ := filepath.Abs(t.workspace)
	absTarget, _ := filepath.Abs(targetDir)
	if !strings.HasPrefix(absTarget, absWorkspace) {
		return ErrorResult("Cannot clone repository outside the safe workspace boundary")
	}

	// Prepare git command
	gitArgs := []string{"clone"}
	if branch != "" {
		gitArgs = append(gitArgs, "-b", branch)
	}
	gitArgs = append(gitArgs, cloneURL, absTarget)

	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	out, err := cmd.CombinedOutput()
	outStr := string(out)

	if err != nil {
		if strings.Contains(outStr, "already exists and is not an empty directory") {
			return SilentResult(fmt.Sprintf("Repository directory already exists at %s. You can proceed with mapping the workspace.", absTarget))
		}
		return ErrorResult(fmt.Sprintf("Git clone failed: %v\nOutput:\n%s", err, outStr))
	}

	successMsg := fmt.Sprintf("Successfully cloned %s into %s.\n\n[System Rule]: ALWAYS use the 'analyze_architecture' tool next to map to the new repository before doing anything else.", cloneURL, absTarget)
	return SilentResult(successMsg)
}
