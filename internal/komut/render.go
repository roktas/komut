package komut

import (
	"fmt"
	"strings"
)

func RenderTemplate(template string, args []string) (string, error) {
	var b strings.Builder
	b.Grow(len(template))

	for i := 0; i < len(template); i++ {
		if template[i] != '$' || i+1 >= len(template) {
			b.WriteByte(template[i])
			continue
		}

		next := template[i+1]
		switch {
		case next == '$':
			b.WriteByte('$')
			i++
		case next == '*':
			b.WriteString(strings.Join(args, " "))
			i++
		case next >= '1' && next <= '9':
			index := int(next - '1')
			if index >= len(args) {
				return "", fail(ErrMissingArgument, "", fmt.Sprintf("template requires argument %d", index+1))
			}
			b.WriteString(args[index])
			i++
		default:
			b.WriteByte('$')
		}
	}

	return b.String(), nil
}
