package komut

import (
	"errors"
	"strings"
)

func Dispatch(input, cwd, home string) (string, error) {
	invocation, err := Parse(input)
	if err != nil {
		return "", err
	}

	resolver, err := NewResolver(cwd, home)
	if err != nil {
		return "", err
	}

	if hasHelpCommand(invocation) {
		if len(invocation.Commands) != 1 || len(invocation.Commands[0].Args) != 0 || invocation.HasLead {
			return "", fail(ErrInvalidInvocation, "help", "builtin help must be used as $x help")
		}
		return resolver.Help()
	}

	components := make([]string, 0, len(invocation.Commands)+1)
	if invocation.Lead != "" {
		components = append(components, invocation.Lead)
	}

	for _, command := range invocation.Commands {
		content, err := resolver.Read(command.Name)
		if err != nil {
			return "", err
		}
		file, err := parseCommandFile(content)
		if err != nil {
			return "", fail(ErrInvalidCommandFile, command.Name, err.Error())
		}
		rendered, err := RenderTemplate(file.Body, command.Args)
		if err != nil {
			if e, ok := errors.AsType[*Error](err); ok && e.Command == "" {
				e.Command = command.Name
			}
			return "", err
		}
		components = append(components, rendered)
	}

	return strings.Join(components, "\n\n"), nil
}

func hasHelpCommand(invocation Invocation) bool {
	for _, command := range invocation.Commands {
		if command.Name == "help" {
			return true
		}
	}
	return false
}
