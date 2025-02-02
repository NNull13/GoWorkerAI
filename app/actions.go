package app

import (
	"log"
	"os"
	"path/filepath"
)

var defaultActions = map[string]string{
	"write_file":  "Use this action to create or overwrite a file with the specified content.",
	"read_file":   "Use this action when you need to read the content of an existing file.",
	"edit_file":   "Use this action to modify an existing file, keeping necessary content intact.",
	"delete_file": "Use this action when a file needs to be deleted for cleanup or replacement.",
	"list_files":  "Use this action to list all files within a directory.",
}

func ActionSwitch(action *Action, folder string) string {
	var generatedCode string
	if action != nil {
		switch action.Action {
		case "write_file":
			writeToFile(folder, action.Filename, action.Content)
			generatedCode = action.Content
		case "read_file":
			generatedCode = readFile(folder, action.Filename)
		case "edit_file":
			editFile(folder, action.Filename, action.Content)
			generatedCode = readFile(folder, action.Filename)
		case "delete_file":
			deleteFile(folder, action.Filename)
			generatedCode = "Successfully deleted file " + action.Filename
		case "list_files":
			generatedCode = listFiles(action.Filename)
		}
	}
	return generatedCode
}

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

func listFiles(baseDir string) string {
	var files string
	baseDir = filepath.Clean(baseDir)
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v\n", path, err)
			return nil
		}
		if !d.IsDir() {
			files += path + "\n"
		}
		return nil
	})
	if err != nil {
		log.Printf("Error walking through directory %s: %v\n", baseDir, err)
	}

	return files
}
