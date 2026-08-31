package komut

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	builtinHelp    = ":help"
	builtinNew     = ":new"
	builtinVersion = ":version"
)

type builtinHandler func(Command, Invocation, *Resolver) (string, error)

type builtinSpec struct {
	Name        string
	Description string
	Aliases     []string
	Handler     builtinHandler
}

func builtinRegistry() []builtinSpec {
	return []builtinSpec{
		{
			Name:        builtinHelp,
			Description: "List available commands.",
			Aliases:     []string{"help", "?"},
			Handler:     runHelpBuiltin,
		},
		{
			Name:        builtinNew,
			Description: "Create a command with the agent.",
			Handler:     runNewBuiltin,
		},
		{
			Name:        builtinVersion,
			Description: "Show the installed Komut version.",
			Handler:     runVersionBuiltin,
		},
	}
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
	return builtin.Handler(command, invocation, resolver)
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

func runVersionBuiltin(command Command, invocation Invocation, _ *Resolver) (string, error) {
	if len(command.Args) != 0 || invocation.HasLead {
		return "", fail(ErrInvalidInvocation, command.Name, "builtin version accepts no arguments or lead text")
	}
	return "Komut " + Version, nil
}

func lookupBuiltin(name string) (builtinSpec, bool) {
	for _, builtin := range builtinRegistry() {
		if builtin.Name == name {
			return builtin, true
		}
	}
	return builtinSpec{}, false
}

func builtinAlias(name string) (string, bool) {
	for _, builtin := range builtinRegistry() {
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
	if len(args) == 0 {
		return "", fail(ErrInvalidInvocation, builtinNew, "command name is required as the first argument")
	}

	name := args[0]
	if !ValidCommandName(name) {
		return "", fail(ErrInvalidCommand, name, "invalid or reserved application command name")
	}

	scope := "project"
	scopeSet := false
	for _, arg := range args[1:] {
		switch arg {
		case "--user", "--project":
			selected := strings.TrimPrefix(arg, "--")
			if scopeSet && selected != scope {
				return "", fail(ErrInvalidInvocation, builtinNew, "choose only one of --user or --project")
			}
			scope = selected
			scopeSet = true
		default:
			return "", fail(ErrInvalidInvocation, builtinNew, "only --user or --project may follow the command name; put the description after --")
		}
	}

	home := filepath.Dir(filepath.Dir(resolver.userCommands))
	atHome := samePath(resolver.cwd, home)
	if !scopeSet && atHome {
		scope = "user"
	}

	var root string
	if scope == "user" {
		root = resolver.userCommands
	} else {
		var err error
		root, err = projectAuthoringRoot(resolver)
		if err != nil {
			return "", err
		}
	}

	target := filepath.Join(root, filepath.FromSlash(name)+".md")
	description := strings.TrimSpace(invocation.Lead)

	var b strings.Builder
	b.WriteString("Author a Komut command using the host agent's normal file-reading and file-editing tools.\n\n")
	fmt.Fprintf(&b, "Scope: %s\n", scope)
	fmt.Fprintf(&b, "Command name: %s\n", name)
	fmt.Fprintf(&b, "Target file: %s\n", target)
	if description == "" {
		b.WriteString("Description: not specified; ask the user for a one-line command description before writing.\n")
	} else {
		fmt.Fprintf(&b, "Description: %s\n", description)
	}
	b.WriteString("Command body: ask the user for the Markdown prompt body before writing.\n")

	b.WriteString("\nCommand file rules:\n")
	b.WriteString("- Write exactly the target file above. Its filename must end in `.md`; do not create an extensionless command file.\n")
	b.WriteString("- Create parent directories when needed.\n")
	b.WriteString("- Start the Markdown file with YAML frontmatter containing the one-line `description`.\n")
	b.WriteString("- Put the command prompt template in the Markdown body after the frontmatter.\n")
	b.WriteString("- Templates may use `$1` through `$9`, `$*`, and `$$`.\n")
	b.WriteString("- Inspect an existing target file before changing it.\n")
	b.WriteString("- Ask the user for the description when it was not supplied and always ask for the command body; do not invent either.\n")
	b.WriteString("- Do not launch an external editor; use the host agent's ordinary editing tools.\n")

	return b.String(), nil
}

func projectAuthoringRoot(resolver *Resolver) (string, error) {
	if resolver.projectCommands != "" {
		return resolver.projectCommands, nil
	}

	home := filepath.Dir(filepath.Dir(resolver.userCommands))
	if samePath(resolver.cwd, home) {
		return "", fail(
			ErrInvalidInvocation,
			builtinNew,
			"project scope is unavailable at the user home; omit --project for user scope or run from a project directory",
		)
	}

	agents := filepath.Join(resolver.cwd, ".agents")
	commands := filepath.Join(agents, "commands")
	for _, path := range []string{agents, commands} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fail(ErrUnsafeProjectPath, builtinNew, fmt.Sprintf("cannot inspect project authoring path %s: %v", path, err))
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fail(ErrUnsafeProjectPath, builtinNew, fmt.Sprintf("unsafe project authoring path: %s", path))
		}
	}
	return commands, nil
}

func isBuiltin(name string) bool {
	return strings.HasPrefix(name, ":")
}
