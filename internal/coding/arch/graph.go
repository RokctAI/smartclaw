package arch

import (
	"fmt"
	"strings"
)

// GenerateMermaidGraph takes a mapped workspace and converts it into a Markdown Mermaid
// dependency graph. This provides the LLM with a structural visualization of how
// the monolithic application is linked together without reading 1000s of files.
func GenerateMermaidGraph(root *ProjectArchNode) string {
	if root == nil || len(root.Children) == 0 {
		return "graph TD;\n  Root(Empty Workspace)"
	}

	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph TD;\n")

	// Walk the tree to build connections
	var walk func(node *ProjectArchNode, parentID string)
	nodeCount := 0

	walk = func(node *ProjectArchNode, parentID string) {
		currentID := fmt.Sprintf("N%d", nodeCount)
		nodeCount++

		label := node.Name
		if node.Type != "" {
			label = fmt.Sprintf("%s [%s]", node.Name, node.Type)
		}

		if node.IsRoot {
			sb.WriteString(fmt.Sprintf("  %s{\"%s\"};\n", currentID, label))
		} else {
			sb.WriteString(fmt.Sprintf("  %s(\"%s\");\n", currentID, label))
		}

		if parentID != "" {
			sb.WriteString(fmt.Sprintf("  %s --> %s;\n", parentID, currentID))
		}

		// Only map interesting directories to prevent massive graph bloat
		for _, child := range node.Children {
			// Skip hidden folders or generic noise like node_modules
			if strings.HasPrefix(child.Name, ".") || child.Name == "node_modules" {
				continue
			}
			walk(child, currentID)
		}
	}

	walk(root, "")
	sb.WriteString("```")
	return sb.String()
}
