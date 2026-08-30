package komut

import "strings"

func Dispatch(input, cwd, home string) (string, error) {
	invocation, err := Parse(input)
	if err != nil {
		return "", err
	}

	resolver, err := NewResolver(cwd, home)
	if err != nil {
		return "", err
	}

	components := make([]string, 0, len(invocation.Commands)+1)
	if invocation.Lead != "" {
		components = append(components, invocation.Lead)
	}

	for _, command := range invocation.Commands {
		template, err := resolver.Read(command.Name)
		if err != nil {
			return "", err
		}
		rendered, err := RenderTemplate(template, command.Args)
		if err != nil {
			if e, ok := err.(*Error); ok && e.Command == "" {
				e.Command = command.Name
			}
			return "", err
		}
		components = append(components, rendered)
	}

	return strings.Join(components, "\n\n"), nil
}
