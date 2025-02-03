package actions

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/utils"
)

func ExecuteFileAction(action *models.ActionTask, folder string) (result string, err error) {
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
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		log.Printf("❌ Error creating directory for %s: %v\n", path, err)
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("❌ Error opening file %s: %v\n", path, err)
		return err
	}

	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		log.Printf("❌ Error writing to file %s: %v\n", path, err)
		return err
	}

	log.Printf("✅ File %s written successfully.\n", path)
	return nil
}

func readFile(baseDir, filename string) (string, error) {
	path := filepath.Join(baseDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠️ File %s does not exist.\n", path)
			return fmt.Sprintf("[ File %s was not found in path %s ]", filename, path), nil
		}
		log.Printf("❌ Error reading file %s: %v\n", path, err)
		return "", err
	}
	return string(content), nil
}

func editFile(baseDir, filename, newContent string) error {
	path := filepath.Join(baseDir, filename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("❌ Error opening file %s for editing: %v\n", path, err)
		return err
	}

	defer file.Close()
	if _, err := file.WriteString(newContent); err != nil {
		log.Printf("❌ Error writing new content to file %s: %v\n", path, err)
		return err
	}

	log.Printf("✅ File %s edited successfully.\n", path)
	return nil
}

func deleteFile(baseDir, filename string) error {
	path := filepath.Join(baseDir, filename)
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠️ File %s does not exist, nothing to delete.\n", path)
		} else {
			log.Printf("❌ Error deleting file %s: %v\n", path, err)
		}
		return err
	}

	log.Printf("✅ File %s deleted successfully.\n", path)
	return nil
}

func listFiles(baseDir string) (string, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	baseDir = filepath.Clean(baseDir)
	tree, err := utils.BuildTree(baseDir, nil, nil)
	if err != nil {
		log.Printf("❌ Error building tree for directory %s: %v\n", baseDir, err)
		return "", err
	}

	return tree, nil
}
