package komut

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	builtinHelp    = ":help"
	builtinNew     = ":new"
	builtinVersion = ":version"
)

type builtinKind uint8

const (
	builtinKindHelp builtinKind = iota + 1
	builtinKindNew
	builtinKindVersion
)

type builtinSpec struct {
	Kind        builtinKind
	Name        string
	Description string
	Aliases     []string
}

var builtinRegistry = [...]builtinSpec{
	{
		Kind:        builtinKindHelp,
		Name:        builtinHelp,
		Description: "List available commands.",
		Aliases:     []string{"help", "?"},
	},
	{
		Kind:        builtinKindNew,
		Name:        builtinNew,
		Description: "Create a command with the agent.",
	},
	{
		Kind:        builtinKindVersion,
		Name:        builtinVersion,
		Description: "Show the installed Komut version.",
	},
}

func dispatchBuiltin(invocation Invocation, resolver *Resolver) (string, error) {
	if len(invocation.Commands) != 1 {
		return "", fail(ErrInvalidInvocation, "", "builtin commands cannot be composed")
	}

	command := invocation.Commands[0]
	builtin, ok := lookupBuiltin(command.Name)
	if !ok {
		return "", fail(ErrInvalidCommand, command.Name, "unknown builtin command")
	}

	switch builtin.Kind {
	case builtinKindHelp:
		return runHelpBuiltin(command, invocation, resolver)
	case builtinKindNew:
		return runNewBuiltin(command, invocation, resolver)
	case builtinKindVersion:
		return runVersionBuiltin(command, invocation)
	default:
		return "", fail(ErrInvalidCommand, command.Name, "unknown builtin command")
	}
}

func runHelpBuiltin(command Command, invocation Invocation, resolver *Resolver) (string, error) {
	if len(command.Args) != 0 || invocation.HasLead {
		return "", fail(ErrInvalidInvocation, command.Name, "builtin help accepts no arguments or lead text")
	}
	return resolver.Help()
}

func runNewBuiltin(command Command, invocation Invocation, resolver *Resolver) (string, error) {
	return newPrompt(command.Args, invocation, resolver)
}

func runVersionBuiltin(command Command, invocation Invocation) (string, error) {
	if len(command.Args) != 0 || invocation.HasLead {
		return "", fail(ErrInvalidInvocation, command.Name, "builtin version accepts no arguments or lead text")
	}
	return "Komut " + Version, nil
}

func lookupBuiltin(name string) (builtinSpec, bool) {
	for _, builtin := range builtinRegistry {
		if builtin.Name == name {
			return builtin, true
		}
	}
	return builtinSpec{}, false
}

func builtinAlias(name string) (string, bool) {
	for _, builtin := range builtinRegistry {
		for _, alias := range builtin.Aliases {
			if alias == name {
				return builtin.Name, true
			}
		}
	}
	return "", false
}

func reservedApplicationName(name string) bool {
	_, reserved := builtinAlias(name)
	return reserved
}

func newPrompt(args []string, invocation Invocation, resolver *Resolver) (string, error) {
	scope := "project"
	scopeSet := false
	name := ""

	for _, arg := range args {
		switch arg {
		case "--user", "--project":
			selected := strings.TrimPrefix(arg, "--")
			if scopeSet && selected != scope {
				return "", fail(ErrInvalidInvocation, builtinNew, "choose only one of --user or --project")
			}
			scope = selected
			scopeSet = true
		default:
			if name != "" {
				return "", fail(ErrInvalidInvocation, builtinNew, "expected at most one command name")
			}
			if !ValidCommandName(arg) {
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
