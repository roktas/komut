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

func TestBuiltinRegistryIsSelfConsistent(t *testing.T) {
	seenNames := make(map[string]bool)
	seenAliases := make(map[string]bool)
	for _, builtin := range builtinRegistry() {
		if builtin.Handler == nil || builtin.Description == "" || !validBuiltinName(builtin.Name) {
			t.Fatalf("incomplete builtin registration: %#v", builtin)
		}
		if seenNames[builtin.Name] {
			t.Fatalf("duplicate builtin name %q", builtin.Name)
		}
		seenNames[builtin.Name] = true

		for _, alias := range builtin.Aliases {
			if seenAliases[alias] {
				t.Fatalf("duplicate builtin alias %q", alias)
			}
			seenAliases[alias] = true
			got, ok := builtinAlias(alias)
			if !ok || got != builtin.Name {
				t.Fatalf("builtinAlias(%q) = %q, %v", alias, got, ok)
			}
			if ValidCommandName(alias) {
				t.Fatalf("builtin alias %q is not reserved from application commands", alias)
			}
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
	if !strings.Contains(got, "Builtins:") || !strings.Contains(got, "No Komut commands found.") {
		t.Fatalf("help = %q", got)
	}
	for _, builtin := range builtinRegistry() {
		if !strings.Contains(got, builtin.Name) || !strings.Contains(got, builtin.Description) {
			t.Fatalf("help = %q, missing builtin %#v", got, builtin)
		}
		for _, alias := range builtin.Aliases {
			if !strings.Contains(got, alias) {
				t.Fatalf("help = %q, missing alias %q", got, alias)
			}
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
