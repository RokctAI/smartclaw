package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrProtocolNotInitialized indicates the protocol submodule exists, but the local .rokct directory is missing.
var ErrProtocolNotInitialized = errors.New("protocol submodule exists, but .rokct directory is missing. Protocol initialization required")

// RuleSet represents a collection of protocol rules.
type RuleSet struct {
	Path    string
	Files   map[string]string
	Healthy bool
}

// LoadProtocol attempts to load the ROKCT protocol from the local workspace.
func LoadProtocol(workspace string) (*RuleSet, error) {
	rokctPath := filepath.Join(workspace, ".rokct")

	// 1. Check for immediate .rokct directory in the workspace root
	if info, err := os.Stat(rokctPath); err == nil && info.IsDir() {
		return loadFromDir(rokctPath)
	}

	// 2. Check for the protocol submodule directory
	submodulePath := filepath.Join(workspace, "The-Rokct-Protocol")
	if info, err := os.Stat(submodulePath); err == nil && info.IsDir() {
		// Submodule exists, but .rokct does not -> warrants init protocol
		return nil, ErrProtocolNotInitialized
	}

	return nil, os.ErrNotExist
}

func loadFromDir(dir string) (*RuleSet, error) {
	rules := &RuleSet{
		Path:  dir,
		Files: make(map[string]string),
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".txt") {
			content, err := os.ReadFile(path)
			if err == nil {
				rel, _ := filepath.Rel(dir, path)
				rules.Files[rel] = string(content)
			}
		}
		return nil
	})

	rules.Healthy = len(rules.Files) > 0
	return rules, err
}

// Format returns the RuleSet as a formatted markdown string for the system prompt.
func (rs *RuleSet) Format() string {
	if !rs.Healthy {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## GLOBAL WORKSPACE PROTOCOL (ROKCT)\n\n")
	sb.WriteString("> This context is inherited from the Monorepo root and governs all development.\n\n")

	// Prioritize README.md or core rule files if they exist
	for name, content := range rs.Files {
		sb.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", name, content))
	}

	return sb.String()
}
