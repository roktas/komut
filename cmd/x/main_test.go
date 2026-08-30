package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadInvocation(t *testing.T) {
	got, err := readInvocation(nil, strings.NewReader("$x foo"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "$x foo" {
		t.Fatalf("readInvocation(stdin) = %q", got)
	}

	got, err = readInvocation([]string{"$x bar"}, strings.NewReader("ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "$x bar" {
		t.Fatalf("readInvocation(arg) = %q", got)
	}

	if _, err := readInvocation([]string{"$x", "foo"}, strings.NewReader("")); err == nil {
		t.Fatal("readInvocation(multiple args) error = nil")
	}
}

func TestHookNonInvocationIsNoop(t *testing.T) {
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(`{"prompt":"hello","cwd":"/tmp"}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q", out.String())
	}
}

func TestHookMalformedInputIsNoop(t *testing.T) {
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(`{`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q", out.String())
	}
}

func TestHookDispatchesUsingPayloadCWD(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(base, "project")
	commands := filepath.Join(project, ".agents", "commands")
	if err := os.MkdirAll(commands, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commands, "foo.md"), []byte("Hello $1"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"prompt":"$x foo world","cwd":` + quoteJSON(project) + `}`
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello world" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestHookInvocationErrorPropagates(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"prompt":"$x missing","cwd":` + quoteJSON(base) + `}`
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(input), &out); err == nil {
		t.Fatal("error = nil")
	}
	if out.Len() != 0 {
		t.Fatalf("partial output = %q", out.String())
	}
}

func quoteJSON(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
