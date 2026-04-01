package arch

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectType defines the type of project detected in a directory.
type ProjectType string

const (
	ProjectGo        ProjectType = "go"
	ProjectNode      ProjectType = "node"
	ProjectFlutter   ProjectType = "flutter"
	ProjectFrappe    ProjectType = "frappe"
	ProjectPython    ProjectType = "python"
	ProjectUnknown   ProjectType = "unknown"
)

// ProjectNode represents a node in the architecture tree.
type ProjectNode struct {
	Path        string        `json:"path"`
	Name        string        `json:"name"`
	Type        ProjectType   `json:"type,omitempty"`
	IsRoot      bool          `json:"is_root,omitempty"`
	Children    []*ProjectNode `json:"children,omitempty"`
	EntryPoints []string      `json:"entry_points,omitempty"`
	Configs     []string      `json:"configs,omitempty"`
}

// MapWorkspace scans the given path and identifies all project roots.
func MapWorkspace(rootPath string) (*ProjectNode, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	root := &ProjectNode{
		Path: absRoot,
		Name: filepath.Base(absRoot),
	}

	// Identify all directories containing a project root file.
	var roots []string
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		
		name := info.Name()
		if name == ".git" || name == "node_modules" || name == ".next" || name == "build" || name == "dist" || name == "vendor" {
			return filepath.SkipDir
		}

		if pt := detectProjectType(path); pt != ProjectUnknown {
			roots = append(roots, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Map each found root to its properties
	for _, rp := range roots {
		rel, _ := filepath.Rel(absRoot, rp)
		node := &ProjectNode{
			Path:   rel,
			Name:   filepath.Base(rp),
			Type:   detectProjectType(rp),
			IsRoot: true,
		}

		// Look for entry points and configs within this root (shallow search)
		files, _ := os.ReadDir(rp)
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			fname := f.Name()
			if isEntryPoint(fname, node.Type) {
				node.EntryPoints = append(node.EntryPoints, fname)
			}
			if isConfig(fname) {
				node.Configs = append(node.Configs, fname)
			}
		}

		root.Children = append(root.Children, node)
	}

	return root, nil
}

// isEntryPoint checks if a filename is a common entry point.
func isEntryPoint(filename string, pt ProjectType) bool {
	switch pt {
	case ProjectGo:
		return filename == "main.go"
	case ProjectNode:
		return filename == "index.js" || filename == "index.ts" || filename == "app.js"
	case ProjectFlutter:
		return filename == "main.dart"
	case ProjectPython:
		return filename == "app.py" || filename == "manage.py"
	default:
		return false
	}
}

// isConfig checks if a filename is a configuration file.
func isConfig(filename string) bool {
	return strings.HasSuffix(filename, ".env") || 
		strings.HasSuffix(filename, ".yml") || 
		strings.HasSuffix(filename, ".yaml") || 
		strings.HasSuffix(filename, ".json") || 
		filename == "Makefile" || 
		filename == "docker-compose.yml"
}

// detectProjectType identifies the language/framework based on file presence.
func detectProjectType(dir string) ProjectType {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return ProjectGo
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return ProjectNode
	}
	if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
		return ProjectFlutter
	}
	if _, err := os.Stat(filepath.Join(dir, "frappe-app.txt")); err == nil {
		return ProjectFrappe
	}
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return ProjectPython
	}
	return ProjectUnknown
}
