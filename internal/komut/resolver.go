package komut

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

type Resolver struct {
	cwd             string
	projectCommands string
	userCommands    string
}

func NewResolver(cwd, home string) (*Resolver, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot resolve working directory: %v", err))
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return nil, fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot resolve home directory: %v", err))
	}
	projectCommands, err := findProjectCommands(cwd, home)
	if err != nil {
		return nil, err
	}
	return &Resolver{
		cwd:             cwd,
		projectCommands: projectCommands,
		userCommands:    filepath.Join(home, ".agents", "commands"),
	}, nil
}

func (r *Resolver) Read(name string) (string, error) {
	if !ValidCommandName(name) {
		return "", fail(ErrInvalidCommand, name, "invalid command name")
	}
	if r.projectCommands != "" {
		content, found, err := readProjectCommand(r.projectCommands, name)
		if err != nil {
			return "", err
		}
		if found {
			return content, nil
		}
	}
	content, found, err := readUserCommand(r.userCommands, name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fail(ErrCommandNotFound, name, "command not found")
	}
	return content, nil
}

func findProjectCommands(cwd, home string) (string, error) {
	current, err := filepath.Abs(cwd)
	if err != nil {
		return "", fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot resolve working directory: %v", err))
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return "", fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot resolve home directory: %v", err))
	}
	for !samePath(current, home) {
		agents := filepath.Join(current, ".agents")
		commands := filepath.Join(agents, "commands")
		commandsInfo, err := os.Lstat(commands)
		if err == nil {
			agentsInfo, agentsErr := os.Lstat(agents)
			if agentsErr != nil || agentsInfo.Mode()&os.ModeSymlink != 0 || !agentsInfo.IsDir() {
				return "", fail(ErrUnsafeProjectPath, "", fmt.Sprintf("unsafe project path: %s", agents))
			}
			if commandsInfo.Mode()&os.ModeSymlink != 0 || !commandsInfo.IsDir() {
				return "", fail(ErrUnsafeProjectPath, "", fmt.Sprintf("unsafe project path: %s", commands))
			}
			return commands, nil
		}
		if !os.IsNotExist(err) {
			if !pathHasNonDirectoryParent(commands) {
				return "", fail(ErrUnsafeProjectPath, "", fmt.Sprintf("cannot inspect project path %s: %v", commands, err))
			}
		}
		parent := filepath.Dir(current)
		if samePath(parent, current) {
			break
		}
		current = parent
	}
	return "", nil
}

func readProjectCommand(root, name string) (string, bool, error) {
	parts := strings.Split(name, "/")
	current := root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return "", false, nil
		}
		if err != nil {
			return "", false, fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot inspect command path: %v", err))
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", false, fail(ErrUnsafeProjectPath, name, "unsafe project command path")
		}
	}
	path := filepath.Join(current, parts[len(parts)-1]+".md")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot inspect command file: %v", err))
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", false, fail(ErrUnsafeProjectPath, name, "unsafe project command file")
	}
	content, err := readValidFile(path, name, info, ErrUnsafeProjectPath)
	if err != nil {
		return "", false, err
	}
	return content, true, nil
}

func readUserCommand(root, name string) (string, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(name)+".md")
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot inspect command file: %v", err))
	}
	if !info.Mode().IsRegular() {
		return "", false, fail(ErrInvalidCommandFile, name, "command file is not a regular file")
	}
	content, err := readValidFile(path, name, info, ErrInvalidCommandFile)
	if err != nil {
		return "", false, err
	}
	return content, true, nil
}

func readValidFile(path, name string, expected os.FileInfo, changedCode ErrorCode) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot open command file: %v", err))
	}
	defer file.Close()

	opened, err := file.Stat()
	if err != nil {
		return "", fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot inspect opened command file: %v", err))
	}
	if !opened.Mode().IsRegular() || !os.SameFile(expected, opened) {
		return "", fail(changedCode, name, "command file changed while opening")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fail(ErrInvalidCommandFile, name, fmt.Sprintf("cannot read command file: %v", err))
	}
	if len(data) == 0 {
		return "", fail(ErrInvalidCommandFile, name, "command file is empty")
	}
	if !utf8.Valid(data) {
		return "", fail(ErrInvalidCommandFile, name, "command file is not valid UTF-8")
	}
	return string(data), nil
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func pathHasNonDirectoryParent(path string) bool {
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		info, err := os.Lstat(parent)
		if err == nil {
			return !info.IsDir()
		}
		if !os.IsNotExist(err) {
			return false
		}
		path = parent
	}
}
