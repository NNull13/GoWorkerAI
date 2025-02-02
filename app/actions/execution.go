package actions

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"GoEngineerAI/app/models"
	"GoEngineerAI/app/utils"
)

func ExecuteAction(action *models.Action, folder string) (result string, err error) {
	if action != nil {
		switch action.Action {
		case write_file:
			err = writeToFile(folder, action.Filename, action.Content)
			result = "Successfully wrote file " + action.Filename
		case read_file:
			result, err = readFile(folder, action.Filename)
		case edit_file:
			err = editFile(folder, action.Filename, action.Content)
			result = "Successfully edited file " + action.Filename
		case delete_file:
			err = deleteFile(folder, action.Filename)
			result = "Successfully deleted file " + action.Filename
		case list_files:
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
	_, err := readFile(baseDir, filename)
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeToFile(baseDir, filename, newContent)
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

func listFiles(baseDir string) (tree string, err error) {
	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	baseDir = filepath.Clean(baseDir)

	if tree, err = utils.BuildTree(baseDir, nil, nil); err != nil {
		log.Printf("Error building tree for directory %s: %v\n", baseDir, err)
		return "", err
	}

	return tree, nil
}
