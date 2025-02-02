package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xlab/treeprint"
)

func RemoveSubstring(input string, start, end int) string {
	if start < 0 || end > len(input) || start >= end {
		return input
	}
	return input[:start] + input[end:]
}

func containsEscapeSequence(s string) bool {
	if len(s) < 2 {
		return false
	}
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\\' && strings.ContainsRune("ntr\"\\", rune(s[i+1])) {
			return true
		}
	}
	return false
}

func UnescapeIfNeeded(s string) string {
	s = strings.TrimSpace(s)
	if containsEscapeSequence(s) {
		if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
			s = fmt.Sprintf("\"%s\"", s)
		}
		unescaped, err := strconv.Unquote(s)
		if err != nil {
			log.Printf("Error unquoting string: %v; text: %s", err, s)
			return s
		}
		return unescaped
	}
	return s
}

func BuildTree(dir string, tree treeprint.Tree, skipDirs map[string]bool) (string, error) {
	if tree == nil {
		tree = treeprint.New()
		tree.SetValue(filepath.Base(dir))
	}
	if skipDirs == nil {
		skipDirs = map[string]bool{
			".git":  true,
			".idea": true,
			"logs":  true,
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
