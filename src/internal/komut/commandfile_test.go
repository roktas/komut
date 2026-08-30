package komut

import "testing"

func TestParseCommandFileFrontmatter(t *testing.T) {
	content := "---\ndescription: Review code carefully.\nignored: true\n---\n\nReview $1.\n"
	got, err := parseCommandFile(content)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "Review code carefully." {
		t.Fatalf("description = %q", got.Description)
	}
	if got.Body != "\nReview $1.\n" {
		t.Fatalf("body = %q", got.Body)
	}
}

func TestParseCommandFileHeadingFallback(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "plain", content: "\n\n## Review code\nBody", want: "Review code"},
		{name: "frontmatter", content: "---\nother: value\n---\n\n# Commit changes\nBody", want: "Commit changes"},
		{name: "closing hashes", content: "### Concise output ###\nBody", want: "Concise output"},
		{name: "not heading", content: "Review code\n# Later heading", want: ""},
		{name: "too many hashes", content: "####### Not a heading", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommandFile(tt.content)
			if err != nil {
				t.Fatal(err)
			}
			if got.Description != tt.want {
				t.Fatalf("description = %q, want %q", got.Description, tt.want)
			}
		})
	}
}

func TestParseCommandFileDescriptionWinsAndNormalizesWhitespace(t *testing.T) {
	content := "---\ndescription: >\n  Review code for\n  correctness.\n---\n# Ignored heading\n"
	got, err := parseCommandFile(content)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "Review code for correctness." {
		t.Fatalf("description = %q", got.Description)
	}
}

func TestParseCommandFileRejectsMalformedFrontmatter(t *testing.T) {
	for _, content := range []string{
		"---\ndescription: broken",
		"---\ndescription: [\n---\nBody",
		"---\ndescription: 123\n---\nBody",
	} {
		if _, err := parseCommandFile(content); err == nil {
			t.Fatalf("parseCommandFile(%q) error = nil", content)
		}
	}
}

func TestParseCommandFileSupportsCRLFFrontmatter(t *testing.T) {
	content := "---\r\ndescription: CRLF command\r\n---\r\nBody\r\n"
	got, err := parseCommandFile(content)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "CRLF command" || got.Body != "Body\r\n" {
		t.Fatalf("file = %#v", got)
	}
}
