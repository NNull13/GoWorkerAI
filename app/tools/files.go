package tools

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"GoWorkerAI/app/utils"
)

type FileAction struct {
	Folder     string `json:"folder"`
	FilePath   string `json:"file_path"`
	Directory  string `json:"directory"`
	NewContent string `json:"new_content"`
	Content    string `json:"content"`
}

type GenericResult struct {
	Result string `json:"result"`
	Error  error  `json:"error,omitempty"`
}

func ExecuteFileAction(action ToolTask) (any, error) {
	var result GenericResult
	var fileAction *FileAction
	fileAction, _ = utils.CastAny[FileAction](action.Parameters)
	if fileAction == nil {
		return nil, errors.New("File Action Not Found")
	}
	switch action.Key {
	case write_file:
		result.Result = "Successfully wrote file " + fileAction.FilePath
		result.Error = writeToFile("", fileAction.FilePath, fileAction.Content)
	case read_file:
		result.Result, result.Error = readFile("", fileAction.FilePath)
	case edit_file:
		result.Result = "Successfully edited file " + fileAction.FilePath
		result.Error = editFile("", fileAction.FilePath, fileAction.NewContent)
	case delete_file:
		result.Result = "Successfully deleted file " + fileAction.FilePath
		result.Error = deleteFile("", fileAction.FilePath)
	case list_files:
		result.Result, result.Error = listFiles(fileAction.Directory)
	}

	return result, nil
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
	dir, err := os.Getwd()
	if baseDir == "" || baseDir == "." {
		baseDir = dir
	}

	var errNote error
	baseDir = filepath.Clean(baseDir)
	if _, err = os.Stat(baseDir); os.IsNotExist(err) {
		errNote = errors.New("could not find that directory. Used '.' instead; could try one of the listed directories")
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Clean(dir)
	} else if err != nil {
		return "", err
	}

	tree, err := utils.BuildTree(baseDir, nil, nil)
	if err != nil {
		log.Printf("❌ Error building tree for directory %s: %v\n", baseDir, err)
		return "", err
	}

	return tree, errNote
}
