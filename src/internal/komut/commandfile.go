package komut

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

type commandFile struct {
	Body        string
	Description string
}

func parseCommandFile(content string) (commandFile, error) {
	metadata, body, hasFrontmatter, err := splitFrontmatter(content)
	if err != nil {
		return commandFile{}, err
	}
	if !hasFrontmatter {
		return commandFile{Body: content, Description: headingDescription(content)}, nil
	}

	var fields map[string]any
	if err := yaml.Unmarshal([]byte(metadata), &fields); err != nil {
		return commandFile{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	description := ""
	if value, ok := fields["description"]; ok && value != nil {
		text, ok := value.(string)
		if !ok {
			return commandFile{}, fmt.Errorf("frontmatter description must be a string")
		}
		description = normalizeDescription(text)
	}
	if description == "" {
		description = headingDescription(body)
	}
	return commandFile{Body: body, Description: description}, nil
}

func splitFrontmatter(content string) (metadata, body string, found bool, err error) {
	line, next := commandFileLine(content, 0)
	if line != "---" {
		return "", content, false, nil
	}
	if next >= len(content) {
		return "", "", true, fmt.Errorf("unterminated YAML frontmatter")
	}

	metadataStart := next
	for next <= len(content) {
		lineStart := next
		line, after := commandFileLine(content, lineStart)
		if line == "---" {
			return content[metadataStart:lineStart], content[after:], true, nil
		}
		if after >= len(content) {
			break
		}
		next = after
	}
	return "", "", true, fmt.Errorf("unterminated YAML frontmatter")
}

func commandFileLine(content string, start int) (string, int) {
	if start >= len(content) {
		return "", len(content)
	}
	if offset := strings.IndexByte(content[start:], '\n'); offset >= 0 {
		end := start + offset
		line := strings.TrimSuffix(content[start:end], "\r")
		return line, end + 1
	}
	return strings.TrimSuffix(content[start:], "\r"), len(content)
}

func headingDescription(body string) string {
	for raw := range strings.SplitSeq(body, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" {
			continue
		}

		hashes := 0
		for hashes < len(line) && hashes < 6 && line[hashes] == '#' {
			hashes++
		}
		if hashes == 0 || hashes == len(line) {
			return ""
		}
		r, _ := utf8.DecodeRuneInString(line[hashes:])
		if !unicode.IsSpace(r) {
			return ""
		}
		return trimClosingHeadingHashes(strings.TrimSpace(line[hashes:]))
	}
	return ""
}

func trimClosingHeadingHashes(text string) string {
	end := len(text)
	for end > 0 && text[end-1] == '#' {
		end--
	}
	if end == len(text) {
		return text
	}
	if end == 0 {
		return ""
	}
	r, _ := utf8.DecodeLastRuneInString(text[:end])
	if !unicode.IsSpace(r) {
		return text
	}
	return strings.TrimSpace(text[:end])
}

func normalizeDescription(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
