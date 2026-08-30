package komut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpAliasesAreEquivalent(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "review", "# Review code\nBody")

	want, err := Dispatch(`$x :help`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{`$x`, `$x   `, `$x help`, `$x ?`} {
		got, err := Dispatch(input, cwd, home)
		if err != nil {
			t.Fatalf("Dispatch(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("Dispatch(%q) differs from :help", input)
		}
	}
}

func TestHelpListsBuiltinRegistry(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)

	got, err := Dispatch(`$x`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"Builtins:",
		":help     List available commands. Aliases: help, ?",
		":new      Create a command with the agent.",
		":version  Show the installed Komut version.",
		"No Komut commands found.",
	} {
		if !strings.Contains(got, text) {
			t.Fatalf("help = %q, missing %q", got, text)
		}
	}
}

func TestNewGeneratesProjectPromptWithoutMutation(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work", "project")
	mustMkdirAll(t, cwd)
	target := filepath.Join(cwd, ".agents", "commands", "code", "review.md")

	got, err := Dispatch(`$x :new code/review -- Review API compatibility.`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Scope: project", target, "Review API compatibility.", "`description`", "`$1` through `$9`"} {
		if !strings.Contains(got, text) {
			t.Fatalf("new prompt = %q, missing %q", got, text)
		}
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target was created or stat error = %v", err)
	}
}

func TestNewUserScopeAndUnnamedCommand(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)

	got, err := Dispatch(`$x :new --user`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Scope: user") || !strings.Contains(got, filepath.Join(home, ".agents", "commands")) || !strings.Contains(got, "determine a valid name with the user") {
		t.Fatalf("new prompt = %q", got)
	}
}

func TestNewRejectsInvalidOptionsAndNames(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)

	for _, input := range []string{
		`$x :new --user --project foo`,
		`$x :new foo bar`,
		`$x :new help`,
	} {
		if _, err := Dispatch(input, cwd, home); err == nil {
			t.Fatalf("Dispatch(%q) error = nil", input)
		}
	}
}

func TestVersionBuiltin(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)

	got, err := Dispatch(`$x :version`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Komut " + Version; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
	for _, input := range []string{`$x :version extra`, `$x :version -- lead`} {
		_, err := Dispatch(input, cwd, home)
		assertErrorCode(t, err, ErrInvalidInvocation)
	}
}

func TestBuiltinCannotComposeAndUnknownBuiltinFails(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "foo", "Foo")

	for _, input := range []string{`$x :help + foo`, `$x foo + :version`} {
		_, err := Dispatch(input, cwd, home)
		assertErrorCode(t, err, ErrInvalidInvocation)
	}
	_, err := Dispatch(`$x :missing`, cwd, home)
	assertErrorCode(t, err, ErrInvalidCommand)
}
