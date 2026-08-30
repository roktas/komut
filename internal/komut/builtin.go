package komut

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	builtinHelp   = ":help"
	builtinCreate = ":create"
)

func dispatchBuiltin(invocation Invocation, resolver *Resolver) (string, error) {
	if len(invocation.Commands) != 1 {
		return "", fail(ErrInvalidInvocation, "", "builtin commands cannot be composed")
	}

	command := invocation.Commands[0]
	switch command.Name {
	case builtinHelp:
		if len(command.Args) != 0 || invocation.HasLead {
			return "", fail(ErrInvalidInvocation, builtinHelp, "builtin help accepts no arguments or lead text")
		}
		return resolver.Help()
	case builtinCreate:
		return createPrompt(command.Args, invocation, resolver)
	default:
		return "", fail(ErrInvalidCommand, command.Name, "unknown builtin command")
	}
}

func createPrompt(args []string, invocation Invocation, resolver *Resolver) (string, error) {
	scope := "project"
	scopeSet := false
	name := ""

	for _, arg := range args {
		switch arg {
		case "--user", "--project":
			selected := strings.TrimPrefix(arg, "--")
			if scopeSet && selected != scope {
				return "", fail(ErrInvalidInvocation, builtinCreate, "choose only one of --user or --project")
			}
			scope = selected
			scopeSet = true
		default:
			if name != "" {
				return "", fail(ErrInvalidInvocation, builtinCreate, "expected at most one command name")
			}
			if !ValidCommandName(arg) || arg == "help" {
				return "", fail(ErrInvalidCommand, arg, "invalid or reserved application command name")
			}
			name = arg
		}
	}

	root := resolver.projectCommands
	if scope == "user" {
		root = resolver.userCommands
	} else if root == "" {
		root = filepath.Join(resolver.cwd, ".agents", "commands")
	}

	var b strings.Builder
	b.WriteString("Author a Komut command using the host agent's normal file-reading and file-editing tools.\n\n")
	fmt.Fprintf(&b, "Scope: %s\n", scope)
	if name == "" {
		fmt.Fprintf(&b, "Target directory: %s\n", root)
		b.WriteString("Command name: not specified; determine a valid name with the user before writing.\n")
	} else {
		fmt.Fprintf(&b, "Command name: %s\n", name)
		fmt.Fprintf(&b, "Target file: %s\n", filepath.Join(root, filepath.FromSlash(name)+".md"))
	}

	b.WriteString("\nCommand file rules:\n")
	b.WriteString("- Create parent directories when needed.\n")
	b.WriteString("- Optional YAML frontmatter may contain a one-line `description`.\n")
	b.WriteString("- The Markdown body is the prompt template.\n")
	b.WriteString("- Templates may use `$1` through `$9`, `$*`, and `$$`.\n")
	b.WriteString("- Inspect an existing target file before changing it.\n")
	b.WriteString("- Ask the user when important command behavior is missing; do not invent it.\n")
	b.WriteString("- Do not launch an external editor; use the host agent's ordinary editing tools.\n")

	if invocation.HasLead && invocation.Lead != "" {
		b.WriteString("\nUser authoring intent:\n")
		b.WriteString(invocation.Lead)
	}

	return b.String(), nil
}

func isBuiltin(name string) bool {
	return strings.HasPrefix(name, ":")
}
