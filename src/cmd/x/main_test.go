package main

import (
	"encoding/json"
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

func TestHookDispatchesUserPromptSubmitUsingPayloadCWD(t *testing.T) {
	base, home, project := hookTestProject(t)
	_ = base

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"prompt":"$x foo world","cwd":` + quoteJSON(project) + `}`
	got := runHookOutput(t, input)
	if got.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Fatalf("hook event = %q", got.HookSpecificOutput.HookEventName)
	}
	if want := hookPreamble + "Hello world"; got.HookSpecificOutput.AdditionalContext != want {
		t.Fatalf("additional context = %q, want %q", got.HookSpecificOutput.AdditionalContext, want)
	}
}

func TestHookUserPromptSubmitExactXShowsHelp(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(base, "project")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"prompt":"$x","cwd":` + quoteJSON(project) + `}`
	got := runHookOutput(t, input)
	if !strings.Contains(got.HookSpecificOutput.AdditionalContext, "Builtins:") {
		t.Fatalf("additional context = %q", got.HookSpecificOutput.AdditionalContext)
	}
}

func TestHookDispatchesClaudePromptExpansion(t *testing.T) {
	_, home, project := hookTestProject(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"hook_event_name":"UserPromptExpansion","expansion_type":"slash_command","command_name":"komut:x","command_args":"foo world","command_source":"plugin","cwd":` + quoteJSON(project) + `}`
	got := runHookOutput(t, input)
	if got.HookSpecificOutput.HookEventName != "UserPromptExpansion" {
		t.Fatalf("hook event = %q", got.HookSpecificOutput.HookEventName)
	}
	if want := hookPreamble + "Hello world"; got.HookSpecificOutput.AdditionalContext != want {
		t.Fatalf("additional context = %q, want %q", got.HookSpecificOutput.AdditionalContext, want)
	}
}

func TestHookClaudePromptExpansionWithoutArgsShowsHelp(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(base, "project")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	input := `{"hook_event_name":"UserPromptExpansion","expansion_type":"slash_command","command_name":"komut:x","command_args":"","command_source":"plugin","cwd":` + quoteJSON(project) + `}`
	got := runHookOutput(t, input)
	if !strings.Contains(got.HookSpecificOutput.AdditionalContext, "Builtins:") {
		t.Fatalf("additional context = %q", got.HookSpecificOutput.AdditionalContext)
	}
}

func TestHookUnrelatedPromptExpansionIsNoop(t *testing.T) {
	input := `{"hook_event_name":"UserPromptExpansion","expansion_type":"slash_command","command_name":"other:x","command_args":"foo","command_source":"plugin","cwd":"/tmp"}`
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
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

func hookTestProject(t *testing.T) (base, home, project string) {
	t.Helper()
	base = t.TempDir()
	home = filepath.Join(base, "home")
	project = filepath.Join(base, "project")
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
	return base, home, project
}

func runHookOutput(t *testing.T, input string) hookOutput {
	t.Helper()
	var out strings.Builder
	if err := run([]string{"--hook"}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var got hookOutput
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("parse hook output: %v", err)
	}
	return got
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
