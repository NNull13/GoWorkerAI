package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xlab/treeprint"
)

func BuildTree(dir string, tree treeprint.Tree, skipDirs map[string]bool) (string, error) {
	if tree == nil {
		tree = treeprint.New()
		tree.SetValue(filepath.Base(dir))
	}
	if skipDirs == nil {
		skipDirs = map[string]bool{
			".git":         true, // Version control (Git)
			".github":      true, // GitHub Actions/workflows
			".idea":        true, // IntelliJ/GoLand IDE settings
			".vscode":      true, // VSCode settings
			"node_modules": true, // JS/Node.js dependencies
			"vendor":       true, // Go vendor directory
			"dist":         true, // Build artifacts
			"build":        true, // Build artifacts
			"bin":          true, // Compiled binaries
			"obj":          true, // Object files
			".cache":       true, // General caching
			".DS_Store":    true, // macOS file metadata
			"logs":         true, // Log files or directories
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				continue
			}
			branch := tree.AddBranch(entry.Name())
			_, err = BuildTree(filepath.Join(dir, entry.Name()), branch, skipDirs)
			if err != nil {
				return "", err
			}
		} else {
			tree.AddNode(entry.Name())
		}
	}
	return tree.String(), nil
}

func ToJSON(input any) string {
	data, err := json.Marshal(input)
	if err != nil {
		log.Printf("⚠️ Error serializing iteration: %v", err)
		return "{}"
	}
	return string(data)
}

func ParseArguments(arguments string) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal([]byte(arguments), &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing arguments: %w", err)
	}
	return result, nil
}

func CastAny[T any](v any) (*T, error) {
	var result T
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("error serializing input to JSON: %w", err)
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &result, nil
}
