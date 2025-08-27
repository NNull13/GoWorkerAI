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
	"sync"

	"GoWorkerAI/app/utils"
)

func executeFileAction(action ToolTask) (string, error) {
	h, ok := fileDispatch[action.Key]
	if !ok {
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
	return h(action.Parameters)
}

var fileDispatch = map[string]func(any) (string, error){
	write_file: func(p any) (string, error) {
		return withParsed[FileAction](p, write_file, func(fa FileAction) (string, error) {
			return writeToFile("", fa.FilePath, fa.Content)
		})
	},
	read_file: func(p any) (string, error) {
		return withParsed[FileAction](p, read_file, func(fa FileAction) (string, error) {
			return readFile("", fa.FilePath)
		})
	},
	delete_file: func(p any) (string, error) {
		return withParsed[FileAction](p, delete_file, func(fa FileAction) (string, error) {
			return deleteFile("", fa.FilePath)
		})
	},
	list_files: func(p any) (string, error) {
		return withParsed[FileAction](p, list_files, func(fa FileAction) (string, error) {
			return listFiles(fa.Directory)
		})
	},
	copy_file: func(p any) (string, error) {
		return withParsed[CopyAction](p, copy_file, func(ca CopyAction) (string, error) {
			return copyFileDirectoryInternal(ca.Source, ca.Destination)
		})
	},
	move_file: func(p any) (string, error) {
		return withParsed[MoveAction](p, move_file, func(ma MoveAction) (string, error) {
			return moveFile("", ma.Source, ma.Destination)
		})
	},
	append_file: func(p any) (string, error) {
		return withParsed[AppendAction](p, append_file, func(aa AppendAction) (string, error) {
			return appendToFile("", aa.FilePath, aa.Content)
		})
	},
	search_file: func(p any) (string, error) {
		return withParsed[SearchAction](p, search_file, func(sa SearchAction) (string, error) {
			return searchInFileOrDir("", sa.FilePath, sa.Pattern, sa.Recursive)
		})
	},
	create_directory: func(p any) (string, error) {
		return withParsed[CreateDirectoryAction](p, create_directory, func(cda CreateDirectoryAction) (string, error) {
			return createDirectory("", cda.DirectoryPath)
		})
	},
}

var (
	rootOnce   sync.Once
	cachedRoot string
	rootErr    error
)

func getRoot() (string, error) {
	rootOnce.Do(func() {
		wf := strings.TrimSpace(workerFolder)
		if wf == "" {
			rootErr = errors.New("WORKER_FOLDER not set")
			return
		}
		abs, err := filepath.Abs(wf)
		if err != nil {
			rootErr = fmt.Errorf("cannot get absolute path of WORKER_FOLDER: %w", err)
			return
		}
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				if mkErr := os.MkdirAll(abs, 0o755); mkErr != nil {
					rootErr = fmt.Errorf("cannot create WORKER_FOLDER: %w", mkErr)
					return
				}
			} else {
				rootErr = fmt.Errorf("cannot stat WORKER_FOLDER: %w", err)
				return
			}
		} else if !info.IsDir() {
			rootErr = fmt.Errorf("WORKER_FOLDER is not a directory: %s", abs)
			return
		}
		cachedRoot = filepath.Clean(abs)
	})
	return cachedRoot, rootErr
}

func ensureInsideRoot(p string) error {
	root, err := getRoot()
	if err != nil {
		return err
	}
	if root == "" {
		return errors.New("sandbox root not configured")
	}
	ok := withinRoot(root, p)
	if !ok {
		return fmt.Errorf("path escapes sandbox: %s", p)
	}
	return nil
}

func safeJoin(path string) (string, error) {
	root, err := getRoot()
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", errors.New("sandbox root not configured")
	}

	if path == "" || path == "." {
		return root, nil
	}

	p := filepath.Clean(path)

	if filepath.IsAbs(p) {
		if !withinRoot(root, p) {
			return "", fmt.Errorf("absolute path outside sandbox: %s", p)
		}
		return p, nil
	}

	base := filepath.Base(root)
	sep := string(os.PathSeparator)
	if p == base {
		return root, nil
	}
	if strings.HasPrefix(p, base+sep) {
		p = strings.TrimPrefix(p, base+sep)
	}

	candidate := filepath.Clean(filepath.Join(root, p))
	if err = ensureInsideRoot(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func withinRoot(root, p string) bool {
	r := filepath.Clean(root)
	q := filepath.Clean(p)
	rel, err := filepath.Rel(r, q)
	if err != nil {
		return false
	}
	dotdots := ".." + string(os.PathSeparator)
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, dotdots) || rel == ".." {
		return false
	}
	return true
}

func denyIfNoRoot() error {
	_, err := getRoot()
	return err
}

func writeToFile(_ string, filename, content string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(filename)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		return "", err
	}
	log.Printf("✅ File %s written successfully.\n", path)
	return "Successfully wrote file " + path, nil
}

func readFile(_ string, filename string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(filename)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Printf("⚠️ File %s does not exist.\n", path)
		return "[ File " + filename + " was not found in path " + path + " ]", nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	log.Printf("✅ File %s read successfully.\n", path)
	return string(content), nil
}

func deleteFile(_ string, filename string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(filename)
	if err != nil {
		return "", err
	}
	if err = os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠️ File %s does not exist, nothing to delete.\n", path)
			return "File " + path + " does not exist, nothing to delete", nil
		}
		return "", err
	}
	log.Printf("✅ File %s deleted successfully.\n", path)
	return "Successfully deleted file " + path, nil
}

func listFiles(dir string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(dir)
	if err != nil {
		return "", err
	}
	tree, err := utils.BuildTree(path, nil, nil)
	if err != nil {
		return "", err
	}
	log.Printf("✅ Directory listing generated for %s.\n", path)
	return tree, nil
}

func copyFileDirectoryInternal(source, destination string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	if source == "" || destination == "" {
		return "", errors.New("both source and destination parameters are required")
	}
	src, err := safeJoin(source)
	if err != nil {
		return "", fmt.Errorf("invalid source: %w", err)
	}
	dst, err := safeJoin(destination)
	if err != nil {
		return "", fmt.Errorf("invalid destination: %w", err)
	}
	info, err := os.Lstat(src)
	if err != nil {
		return "", fmt.Errorf("source does not exist: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		var target string
		target, err = filepath.EvalSymlinks(src)
		if err != nil {
			return "", fmt.Errorf("cannot resolve symlink source: %w", err)
		}
		if err = ensureInsideRoot(target); err != nil {
			return "", err
		}
		info, err = os.Stat(target)
		if err != nil {
			return "", err
		}
		src = target
	}
	if info.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return "", fmt.Errorf("copy operation failed: %w", err)
		}
	} else {
		if err := copyFile(src, dst); err != nil {
			return "", fmt.Errorf("copy operation failed: %w", err)
		}
	}
	log.Printf("✅ File %s copied to %s.\n", src, dst)
	return "Successfully copied " + src + " to " + dst, nil
}

func copyFile(source, dest string) error {
	if err := ensureInsideRoot(source); err != nil {
		return err
	}
	if err := ensureInsideRoot(dest); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if err = os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if st, err := os.Stat(source); err == nil {
		_ = os.Chmod(dest, st.Mode()&0o777)
	}
	return nil
}

func copyDir(source, dest string) error {
	if err := ensureInsideRoot(source); err != nil {
		return err
	}
	if err := ensureInsideRoot(dest); err != nil {
		return err
	}
	srcInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func moveFile(_ string, source, destination string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	src, err := safeJoin(source)
	if err != nil {
		return "", err
	}
	dst, err := safeJoin(destination)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err = os.Rename(src, dst); err != nil {
		return "", err
	}
	log.Printf("✅ Moved %s to %s.\n", src, dst)
	return "Successfully moved " + src + " to " + dst, nil
}

func appendToFile(_ string, filename, content string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(filename)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		return "", err
	}
	log.Printf("✅ Content appended to file %s.\n", path)
	return "Successfully appended content to " + path, nil
}

func searchInFileOrDir(_ string, pathParam, pattern string, recursive bool) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	absPath, err := safeJoin(pathParam)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("could not access path: %w", err)
	}
	if stat.IsDir() {
		var results []string
		if recursive {
			results, err = searchDirectoryRecursive(absPath, pattern)
		} else {
			results, err = searchDirectory(absPath, pattern)
		}
		if err != nil {
			return "", err
		}
		return strings.Join(results, "\n"), nil
	}
	found, err := searchFile(absPath, pattern)
	if err != nil {
		return "", err
	}
	return strings.Join(found, "\n"), nil
}

func searchDirectoryRecursive(baseDir, pattern string) ([]string, error) {
	if err := ensureInsideRoot(baseDir); err != nil {
		return nil, err
	}
	var matches []string
	err := filepath.WalkDir(baseDir, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() {
			found, fe := searchFile(p, pattern)
			if fe != nil {
				return nil
			}
			matches = append(matches, found...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func searchDirectory(baseDir, pattern string) ([]string, error) {
	if err := ensureInsideRoot(baseDir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			continue
		}
		p := filepath.Join(baseDir, e.Name())
		found, fe := searchFile(p, pattern)
		if fe != nil {
			continue
		}
		matches = append(matches, found...)
	}
	return matches, nil
}

func searchFile(path, pattern string) ([]string, error) {
	if err := ensureInsideRoot(path); err != nil {
		return nil, err
	}
	var matches []string
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	re, err := regexp.Compile(pattern)
	if err != nil {
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
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

func createDirectory(_ string, directoryPath string) (string, error) {
	if err := denyIfNoRoot(); err != nil {
		return "", err
	}
	path, err := safeJoin(directoryPath)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	log.Printf("✅ Directory %s created successfully.\n", path)
	return "Successfully created directory " + path, nil
}
