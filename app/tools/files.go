package tools

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"GoWorkerAI/app/utils"
)

func executeFileAction(action ToolTask) (string, error) {
	switch action.Key {
	case write_file:
		fa, err := utils.CastAny[FileAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing write_file action: %v\n", err)
			return "", err
		}
		return writeToFile("", fa.FilePath, fa.Content)
	case read_file:
		fa, err := utils.CastAny[FileAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing read_file action: %v\n", err)
			return "", err
		}
		return readFile("", fa.FilePath)
	case delete_file:
		fa, err := utils.CastAny[FileAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing delete_file action: %v\n", err)
			return "", err
		}
		return deleteFile("", fa.FilePath)
	case list_files:
		fa, err := utils.CastAny[FileAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing list_files action: %v\n", err)
			return "", err
		}
		return listFiles(fa.Directory)
	case copy_file:
		ca, err := utils.CastAny[CopyAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing copy_file action: %v\n", err)
			return "", err
		}
		return copyFileDirectoryInternal(ca.Source, ca.Destination)
	case move_file:
		ma, err := utils.CastAny[MoveAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing move_file action: %v\n", err)
			return "", err
		}
		return moveFile("", ma.Source, ma.Destination)
	case append_file:
		aa, err := utils.CastAny[AppendAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing append_file action: %v\n", err)
			return "", err
		}
		return appendToFile("", aa.FilePath, aa.Content)
	case search_file:
		sa, err := utils.CastAny[SearchAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing search_file action: %v\n", err)
			return "", err
		}
		return searchInFileOrDir("", sa.FilePath, sa.Pattern, sa.Recursive)
	case create_directory:
		cda, err := utils.CastAny[CreateDirectoryAction](action.Parameters)
		if err != nil {
			log.Printf("❌ Error parsing create_directory action: %v\n", err)
			return "", err
		}
		return createDirectory("", cda.DirectoryPath)
	default:
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
}

func writeToFile(baseDir, filename, content string) (string, error) {
	path := filepath.Join(baseDir, filename)
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		log.Printf("❌ Error creating directory for %s: %v\n", path, err)
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("❌ Error opening file %s: %v\n", path, err)
		return "", err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		log.Printf("❌ Error writing to file %s: %v\n", path, err)
		return "", err
	}
	log.Printf("✅ File %s written successfully.\n", path)
	return "Successfully wrote file " + path, nil
}

func readFile(baseDir, filename string) (string, error) {
	path := filepath.Join(baseDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠️ File %s does not exist.\n", path)
			return "[ File " + filename + " was not found in path " + path + " ]", nil
		}
		log.Printf("❌ Error reading file %s: %v\n", path, err)
		return "", err
	}
	log.Printf("✅ File %s read successfully.\n", path)
	return string(content), nil
}

func deleteFile(baseDir, filename string) (string, error) {
	path := filepath.Join(baseDir, filename)
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠️ File %s does not exist, nothing to delete.\n", path)
			return "File " + path + " does not exist, nothing to delete", nil
		}
		log.Printf("❌ Error deleting file %s: %v\n", path, err)
		return "", err
	}
	log.Printf("✅ File %s deleted successfully.\n", path)
	return "Successfully deleted file " + path, nil
}

func listFiles(baseDir string) (string, error) {
	var note, tree string
	cwd, err := os.Getwd()
	if baseDir == "" || baseDir == "." {
		baseDir = cwd
	}
	baseDir = filepath.Clean(baseDir)
	_, statErr := os.Stat(baseDir)
	if os.IsNotExist(statErr) {
		cwd, err = os.Getwd()
		if err != nil {
			log.Printf("❌ Error getting current directory: %v\n", err)
			return "", err
		}
		baseDir = filepath.Clean(cwd)
		note = fmt.Sprintf("⚠️ Could not find that directory. Used '.' instead.\n")
	}
	tree, err = utils.BuildTree(baseDir, nil, nil)
	if err != nil {
		log.Printf("❌ Error building tree for directory %s: %v\n", baseDir, err)
		return "", err
	}
	log.Printf("✅ Directory listing generated for %s.\n", baseDir)
	return note + tree, nil
}

func copyFileDirectoryInternal(source, destination string) (string, error) {
	if source == "" || destination == "" {
		log.Printf("❌ Missing source or destination.\n")
		return "", errors.New("both source and destination parameters are required")
	}
	info, err := os.Stat(source)
	if err != nil {
		log.Printf("❌ Source %s does not exist: %v\n", source, err)
		return "", fmt.Errorf("source does not exist: %w", err)
	}
	if info.IsDir() {
		err = copyDir(source, destination)
	} else {
		err = copyFile(source, destination)
	}
	if err != nil {
		log.Printf("❌ Copy operation failed: %v\n", err)
		return "", fmt.Errorf("copy operation failed: %w", err)
	}
	log.Printf("✅ File %s copied to %s.\n", source, destination)
	return "Successfully copied " + source + " to " + destination, nil
}

func copyFile(source, dest string) error {
	inputFile, err := os.Open(source)
	if err != nil {
		log.Printf("❌ Error opening source file %s: %v\n", source, err)
		return err
	}
	defer inputFile.Close()
	destDir := filepath.Dir(dest)
	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		log.Printf("❌ Error creating directory for %s: %v\n", dest, err)
		return err
	}
	outputFile, err := os.Create(dest)
	if err != nil {
		log.Printf("❌ Error creating destination file %s: %v\n", dest, err)
		return err
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		log.Printf("❌ Error copying from %s to %s: %v\n", source, dest, err)
		return err
	}
	info, err := os.Stat(source)
	if err != nil {
		log.Printf("❌ Error reading info of source file %s: %v\n", source, err)
		return err
	}
	err = os.Chmod(dest, info.Mode())
	if err != nil {
		log.Printf("❌ Error setting permissions on %s: %v\n", dest, err)
		return err
	}
	log.Printf("✅ File %s copied to %s.\n", source, dest)
	return nil
}

func copyDir(source, dest string) error {
	info, err := os.Stat(source)
	if err != nil {
		log.Printf("❌ Error reading directory info for %s: %v\n", source, err)
		return err
	}
	err = os.MkdirAll(dest, info.Mode())
	if err != nil {
		log.Printf("❌ Error creating directory %s: %v\n", dest, err)
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		log.Printf("❌ Error reading directory %s: %v\n", source, err)
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			err = copyDir(sourcePath, destPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(sourcePath, destPath)
			if err != nil {
				return err
			}
		}
	}
	log.Printf("✅ Directory %s copied to %s.\n", source, dest)
	return nil
}

func moveFile(baseDir, source, destination string) (string, error) {
	src := filepath.Join(baseDir, source)
	dst := filepath.Join(baseDir, destination)
	err := os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		log.Printf("❌ Error creating directory for %s: %v\n", dst, err)
		return "", err
	}
	err = os.Rename(src, dst)
	if err != nil {
		log.Printf("❌ Error moving from %s to %s: %v\n", src, dst, err)
		return "", err
	}
	log.Printf("✅ Moved %s to %s.\n", src, dst)
	return "Successfully moved " + src + " to " + dst, nil
}

func appendToFile(baseDir, filename, content string) (string, error) {
	path := filepath.Join(baseDir, filename)
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		log.Printf("❌ Error creating directory for %s: %v\n", path, err)
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("❌ Error opening file %s for append: %v\n", path, err)
		return "", err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		log.Printf("❌ Error appending to file %s: %v\n", path, err)
		return "", err
	}
	log.Printf("✅ Content appended to file %s.\n", path)
	return "Successfully appended content to " + path, nil
}

func searchInFileOrDir(baseDir, pathParam, pattern string, recursive bool) (string, error) {
	absPath := filepath.Join(baseDir, pathParam)
	stat, err := os.Stat(absPath)
	if err != nil {
		log.Printf("❌ Could not access path %s: %v\n", absPath, err)
		return "", fmt.Errorf("could not access path: %w", err)
	}
	if stat.IsDir() {
		if recursive {
			results, e := searchDirectoryRecursive(absPath, pattern)
			if e != nil {
				log.Printf("❌ Error searching directory recursively %s: %v\n", absPath, e)
				return "", e
			}
			log.Printf("✅ Recursive search completed in %s.\n", absPath)
			return strings.Join(results, "\n"), nil
		}
		results, e := searchDirectory(absPath, pattern)
		if e != nil {
			log.Printf("❌ Error searching directory %s: %v\n", absPath, e)
			return "", e
		}
		log.Printf("✅ Search completed in %s.\n", absPath)
		return strings.Join(results, "\n"), nil
	}
	results, e := searchFile(absPath, pattern)
	if e != nil {
		log.Printf("❌ Error searching file %s: %v\n", absPath, e)
		return "", e
	}
	log.Printf("✅ Search completed in file %s.\n", absPath)
	return strings.Join(results, "\n"), nil
}

func searchDirectoryRecursive(dir, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !info.IsDir() {
			found, fe := searchFile(path, pattern)
			if fe != nil {
				return nil
			}
			matches = append(matches, found...)
		}
		return nil
	})
	if err != nil {
		log.Printf("❌ Error walking through directory %s: %v\n", dir, err)
		return nil, err
	}
	return matches, nil
}

func searchDirectory(dir, pattern string) ([]string, error) {
	var matches []string
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("❌ Error reading directory %s: %v\n", dir, err)
		return nil, err
	}
	for _, fileEntry := range files {
		if !fileEntry.IsDir() {
			path := filepath.Join(dir, fileEntry.Name())
			found, e := searchFile(path, pattern)
			if e != nil {
				log.Printf("⚠️ Error searching file %s: %v\n", path, e)
				continue
			}
			matches = append(matches, found...)
		}
	}
	return matches, nil
}

func searchFile(path, pattern string) ([]string, error) {
	var matches []string
	file, err := os.Open(path)
	if err != nil {
		log.Printf("❌ Error opening file %s for search: %v\n", path, err)
		return nil, err
	}
	defer file.Close()
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("❌ Error compiling regex: %v\n", err)
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	lineNumber := 1
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d:%s", path, lineNumber, line))
		}
		lineNumber++
	}
	if err := scanner.Err(); err != nil {
		log.Printf("❌ Error scanning file %s: %v\n", path, err)
		return nil, err
	}
	return matches, nil
}

func createDirectory(baseDir, directoryPath string) (string, error) {
	path := filepath.Join(baseDir, directoryPath)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Printf("❌ Error creating directory %s: %v\n", path, err)
		return "", err
	}
	log.Printf("✅ Directory %s created successfully.\n", path)
	return "Successfully created directory " + path, nil
}
