package app

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/xlab/treeprint"
)

var defaultActions = map[string]string{
	"write_file": "Use this action to create a new file or overwrite an existing file with the provided content. " +
		"Ensure that you specify the complete file path and include all necessary content. Ideal for saving generated code or configuration files. " +
		"Double-check that the content does not contain unwanted formatting or escape sequences.",

	"read_file": "Use this action to retrieve the content of an existing file. " +
		"Provide the correct relative or absolute path to the file. Ensure that the file exists and is accessible. " +
		"This action is useful for verifying file content or for processing data from external sources.",

	"edit_file": "Use this action to modify an existing file while preserving its original content. " +
		"Specify the target file path and the additional content or changes to be applied. " +
		"This action should merge the new content with the existing content, ensuring that no essential data is lost.",

	"delete_file": "Use this action to remove an existing file from the system. " +
		"Provide the exact file path to be deleted. Confirm that the file is no longer required before using this action, " +
		"as deletion is irreversible.",

	"list_files": "Use this action to generate a detailed list of files within a specified directory. " +
		"Supply the folder path as the 'filename' parameter. The output should include a hierarchical tree structure " +
		"of the directory, and optionally, the contents of each file. This action is ideal for auditing or processing " +
		"the repository structure to identify relevant files for further operations.",
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

func listFiles(baseDir string) (string, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	baseDir = filepath.Clean(baseDir)

	tree := treeprint.New()
	tree.SetValue(filepath.Base(baseDir))

	skipDirs := map[string]bool{
		".git":  true,
		".idea": true,
		"logs":  true,
	}

	if err := buildTree(baseDir, tree, skipDirs); err != nil {
		log.Printf("Error building tree for directory %s: %v\n", baseDir, err)
		return "", err
	}

	return tree.String(), nil
}

func buildTree(dir string, tree treeprint.Tree, skipDirs map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				continue
			}
			branch := tree.AddBranch(entry.Name())
			err = buildTree(filepath.Join(dir, entry.Name()), branch, skipDirs)
			if err != nil {
				return err
			}
		} else {
			tree.AddNode(entry.Name())
		}
	}
	return nil
}
