package app

import (
	"log"
	"os"
	"path/filepath"
)

func writeToFile(baseDir, filename, content string) {
	path := filepath.Join(baseDir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s written successfully.\n", path)
	}
}

func readFile(baseDir, filename string) string {
	path := filepath.Join(baseDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", path, err)
		return ""
	}
	return string(content)
}

func editFile(baseDir, filename, newContent string) {
	existingContent := readFile(baseDir, filename)
	mergedContent := existingContent + "\n" + newContent
	writeToFile(baseDir, filename, mergedContent)
}

func deleteFile(baseDir, filename string) {
	path := filepath.Join(baseDir, filename)
	err := os.Remove(path)
	if err != nil {
		log.Printf("Error deleting file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s deleted successfully.\n", path)
	}
}
