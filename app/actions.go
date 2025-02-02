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

func ExecuteAction(action *Action, folder string) (result string, err error) {
	if action != nil {
		switch action.Action {
		case "write_file":
			err = writeToFile(folder, action.Filename, action.Content)
			result = "Successfully wrote file " + action.Filename
		case "read_file":
			result, err = readFile(folder, action.Filename)
		case "edit_file":
			err = editFile(folder, action.Filename, action.Content)
			result = "Successfully edited file " + action.Filename
		case "delete_file":
			err = deleteFile(folder, action.Filename)
			result = "Successfully deleted file " + action.Filename
		case "list_files":
			result, err = listFiles(action.Filename)
		}
	}
	return result, err
}

func writeToFile(baseDir, filename, content string) error {
	path := filepath.Join(baseDir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s written successfully.\n", path)
	}
	return err
}

func readFile(baseDir, filename string) (string, error) {
	path := filepath.Join(baseDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", path, err)
		return "", err
	}
	return string(content), nil
}

func editFile(baseDir, filename, newContent string) error {
	existingContent, err := readFile(baseDir, filename)
	if err != nil {
		return err
	}
	mergedContent := existingContent + "\n" + newContent
	return writeToFile(baseDir, filename, mergedContent)
}

func deleteFile(baseDir, filename string) error {
	path := filepath.Join(baseDir, filename)
	err := os.Remove(path)
	if err != nil {
		log.Printf("Error deleting file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s deleted successfully.\n", path)
	}
	return err
}

func listFiles(baseDir string) (string, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

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

	return files, err
}
