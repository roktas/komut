package komut

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const helpBuiltins = "Builtins:\n\n" +
	":help     List available commands. Aliases: help, ?\n" +
	":new      Create a command with the agent.\n" +
	":version  Show the installed Komut version.\n"

func TestHelpListsCommandsWithDescriptionsAndProjectPrecedence(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "src", "project")
	cwd := filepath.Join(project, "sub")
	mustMkdirAll(t, cwd)

	writeCommand(t, home, "alpha", "# User alpha\nBody")
	writeCommand(t, home, "code/review", "---\ndescription: User review\n---\nUser body")
	writeCommand(t, project, "code/review", "---\ndescription: Project review\n---\nProject body")
	writeCommand(t, project, "zeta", "No heading here")
	writeCommand(t, home, "help", "This must never override builtin help")

	got, err := Dispatch(`$x :help`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	want := helpBuiltins + "\nCommands:\n\n" +
		"alpha        User alpha\n" +
		"code/review  Project review\n" +
		"zeta"
	if got != want {
		t.Fatalf("Dispatch($x :help) = %q, want %q", got, want)
	}
	if strings.Contains(got, "User review") || strings.Contains(got, "\nhelp") {
		t.Fatalf("unexpected shadowed or reserved command in help: %q", got)
	}
}

func TestHelpRecursivelyFindsUserSymlinkCommandsWithoutLooping(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	commands := filepath.Join(home, ".agents", "commands")
	outside := filepath.Join(base, "outside")
	mustMkdirAll(t, commands)
	mustMkdirAll(t, cwd)
	mustMkdirAll(t, outside)
	mustWriteFile(t, filepath.Join(outside, "review.md"), []byte("# Linked review\nBody"))
	if err := os.Symlink(outside, filepath.Join(commands, "code")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(commands, filepath.Join(outside, "loop")); err != nil {
		t.Fatal(err)
	}

	got, err := Dispatch(`$x`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "code/review  Linked review") {
		t.Fatalf("help did not discover symlink command: %q", got)
	}
	if strings.Contains(got, "loop/") {
		t.Fatalf("help followed a symlink cycle: %q", got)
	}
}

func TestHelpDoesNotFollowProjectSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "project")
	cwd := filepath.Join(project, "sub")
	commands := filepath.Join(project, ".agents", "commands")
	outside := filepath.Join(base, "outside")
	mustMkdirAll(t, commands)
	mustMkdirAll(t, cwd)
	mustMkdirAll(t, outside)
	mustWriteFile(t, filepath.Join(outside, "secret.md"), []byte("# Secret"))
	if err := os.Symlink(outside, filepath.Join(commands, "unsafe")); err != nil {
		t.Fatal(err)
	}
	writeCommand(t, home, "safe", "# Safe command")

	got, err := Dispatch(`$x :help`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "unsafe/secret") {
		t.Fatalf("help followed project symlink: %q", got)
	}
	if !strings.Contains(got, "safe  Safe command") {
		t.Fatalf("help lost user command: %q", got)
	}
}

func TestHelpWithNoCommandsShowsAbsoluteCreationPaths(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work", "project")
	mustMkdirAll(t, cwd)

	got, err := Dispatch(`$x`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	userCommands := filepath.Join(home, ".agents", "commands")
	projectCommands := filepath.Join(cwd, ".agents", "commands")
	for _, text := range []string{"Builtins:", "No Komut commands found.", userCommands, projectCommands} {
		if !strings.Contains(got, text) {
			t.Fatalf("help = %q, missing %q", got, text)
		}
	}
	if !filepath.IsAbs(userCommands) || !filepath.IsAbs(projectCommands) {
		t.Fatal("test paths must be absolute")
	}
}

func TestHelpNoCommandsUsesExistingNearestProjectPath(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "project")
	cwd := filepath.Join(project, "deep", "sub")
	projectCommands := filepath.Join(project, ".agents", "commands")
	mustMkdirAll(t, projectCommands)
	mustMkdirAll(t, cwd)

	got, err := Dispatch(`$x :help`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, projectCommands) {
		t.Fatalf("help = %q, missing project scope %q", got, projectCommands)
	}
	if strings.Contains(got, filepath.Join(cwd, ".agents", "commands")) {
		t.Fatalf("help suggested cwd instead of existing project scope: %q", got)
	}
}

func TestHelpRejectsArgumentsCompositionAndLead(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "foo", "Foo")

	for _, input := range []string{
		`$x :help extra`,
		`$x help + foo`,
		`$x foo + ?`,
		`$x :help -- lead`,
		`$x help --`,
	} {
		t.Run(input, func(t *testing.T) {
			_, err := Dispatch(input, cwd, home)
			assertErrorCode(t, err, ErrInvalidInvocation)
		})
	}
}

func TestHelpListsMalformedMetadataWithEmptyDescription(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "broken", "---\ndescription: [\n---\nBody")

	got, err := Dispatch(`$x :help`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if want := helpBuiltins + "\nCommands:\n\nbroken"; got != want {
		t.Fatalf("help = %q, want %q", got, want)
	}

	_, err = Dispatch(`$x broken`, cwd, home)
	assertErrorCode(t, err, ErrInvalidCommandFile)
}
