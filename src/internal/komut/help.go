package komut

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type HelpEntry struct {
	Name        string
	Description string
}

func (r *Resolver) Help() (string, error) {
	entries, err := r.ListCommands()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("Builtins:\n\n")
	writeBuiltinHelp(&b)

	if len(entries) == 0 {
		projectCommands := r.projectCommands
		if projectCommands == "" {
			projectCommands = filepath.Join(r.cwd, ".agents", "commands")
		}
		fmt.Fprintf(
			&b,
			"\nNo Komut commands found.\n\nCreate user-wide commands in:\n  %s",
			r.userCommands,
		)
		if samePath(projectCommands, r.userCommands) {
			b.WriteString("\n\nProject commands are unavailable from the user home.\nRun Komut from a project directory; project commands live in:\n  <project>/.agents/commands")
		} else {
			fmt.Fprintf(&b, "\n\nCreate project commands in:\n  %s", projectCommands)
		}
		return b.String(), nil
	}

	width := 0
	for _, entry := range entries {
		width = max(width, len(entry.Name))
	}

	b.WriteString("\nCommands:\n\n")
	for i, entry := range entries {
		b.WriteString(entry.Name)
		if entry.Description != "" {
			b.WriteString(strings.Repeat(" ", width-len(entry.Name)+2))
			b.WriteString(entry.Description)
		}
		if i+1 < len(entries) {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

func writeBuiltinHelp(b *strings.Builder) {
	registry := builtinRegistry()
	width := 0
	for _, builtin := range registry {
		width = max(width, len(builtin.Name))
	}
	for _, builtin := range registry {
		b.WriteString(builtin.Name)
		b.WriteString(strings.Repeat(" ", width-len(builtin.Name)+2))
		b.WriteString(builtin.Description)
		if len(builtin.Aliases) != 0 {
			b.WriteString(" Aliases: ")
			b.WriteString(strings.Join(builtin.Aliases, ", "))
		}
		b.WriteByte('\n')
	}
}

func (r *Resolver) ListCommands() ([]HelpEntry, error) {
	commands := make(map[string]HelpEntry)
	if err := r.listUserCommands(commands); err != nil {
		return nil, err
	}
	if err := r.listProjectCommands(commands); err != nil {
		return nil, err
	}

	entries := slices.SortedFunc(maps.Values(commands), func(a, b HelpEntry) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return entries, nil
}

func (r *Resolver) listProjectCommands(commands map[string]HelpEntry) error {
	if r.projectCommands == "" {
		return nil
	}

	return filepath.WalkDir(r.projectCommands, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == r.projectCommands {
				return fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot enumerate project commands: %v", walkErr))
			}
			return nil
		}
		if path == r.projectCommands {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(r.projectCommands, path)
		if err != nil {
			return nil
		}
		logical := filepath.ToSlash(rel)
		if entry.IsDir() {
			if !validCommandPrefix(logical) {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		name := strings.TrimSuffix(logical, ".md")
		if !ValidCommandName(name) {
			return nil
		}

		description := ""
		content, found, err := readProjectCommand(r.projectCommands, name)
		if err == nil && found {
			description = bestEffortDescription(content)
		}
		commands[name] = HelpEntry{Name: name, Description: description}
		return nil
	})
}

func (r *Resolver) listUserCommands(commands map[string]HelpEntry) error {
	active := make(map[string]bool)
	return r.walkUserCommandDir(r.userCommands, "", active, commands)
}

func (r *Resolver) walkUserCommandDir(dir, prefix string, active map[string]bool, commands map[string]HelpEntry) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot inspect user command directory: %v", err))
	}
	if !info.IsDir() {
		return nil
	}

	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil
	}
	real, err = filepath.Abs(real)
	if err != nil {
		return nil
	}
	key := pathKey(real)
	if active[key] {
		return nil
	}
	active[key] = true
	defer delete(active, key)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fail(ErrInvalidCommandFile, "", fmt.Sprintf("cannot enumerate user commands: %v", err))
	}
	for _, entry := range entries {
		logical := entry.Name()
		if prefix != "" {
			logical = prefix + "/" + logical
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if !validCommandPrefix(logical) {
				continue
			}
			if err := r.walkUserCommandDir(path, logical, active, commands); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(logical, ".md")
		if !ValidCommandName(name) {
			continue
		}

		description := ""
		content, found, err := readUserCommand(r.userCommands, name)
		if err == nil && found {
			description = bestEffortDescription(content)
		}
		commands[name] = HelpEntry{Name: name, Description: description}
	}
	return nil
}

func validCommandPrefix(name string) bool {
	if name == "" || len(name) >= 64 {
		return false
	}
	for segment := range strings.SplitSeq(name, "/") {
		if !validSegment(segment) {
			return false
		}
	}
	return true
}

func bestEffortDescription(content string) string {
	file, err := parseCommandFile(content)
	if err != nil {
		return ""
	}
	return file.Description
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
