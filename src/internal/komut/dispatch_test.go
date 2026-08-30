package komut

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDispatchProjectOverridesUserAndComposes(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "src", "project")
	cwd := filepath.Join(project, "lib")
	mustMkdirAll(t, cwd)

	writeCommand(t, home, "code/review", "user review $1")
	writeCommand(t, project, "code/review", "project review $1")
	writeCommand(t, home, "concise", "Be concise.")

	got, err := Dispatch(`$x code/review src/foo.rb + concise -- Public API.`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	want := "Public API.\n\nproject review src/foo.rb\n\nBe concise."
	if got != want {
		t.Fatalf("Dispatch() = %q, want %q", got, want)
	}
}

func TestDispatchMissingProjectCommandFallsBackToUser(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "src", "project")
	cwd := filepath.Join(project, "sub")
	mustMkdirAll(t, filepath.Join(project, ".agents", "commands"))
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "review", "user review")

	got, err := Dispatch(`$x review`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user review" {
		t.Fatalf("Dispatch() = %q", got)
	}
}

func TestDispatchNearestProjectTreeIsBoundary(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	outer := filepath.Join(home, "src", "outer")
	inner := filepath.Join(outer, "inner")
	cwd := filepath.Join(inner, "sub")
	mustMkdirAll(t, cwd)
	writeCommand(t, outer, "review", "outer review")
	mustMkdirAll(t, filepath.Join(inner, ".agents", "commands"))

	_, err := Dispatch(`$x review`, cwd, home)
	assertErrorCode(t, err, ErrCommandNotFound)
}

func TestDispatchHomeCommandsAreUserScope(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "src", "project")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "review", "user review")

	got, err := Dispatch(`$x review`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user review" {
		t.Fatalf("Dispatch() = %q", got)
	}
}

func TestDispatchTreatsSymlinkAliasAsUserHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}

	base := t.TempDir()
	home := filepath.Join(base, "real-home")
	homeAlias := filepath.Join(base, "home")
	commands := filepath.Join(home, ".agents", "commands")
	target := filepath.Join(base, "review.md")
	mustMkdirAll(t, commands)
	mustWriteFile(t, target, []byte("user review"))
	if err := os.Symlink(target, filepath.Join(commands, "review.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(home, homeAlias); err != nil {
		t.Fatal(err)
	}

	got, err := Dispatch(`$x review`, home, homeAlias)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user review" {
		t.Fatalf("Dispatch() = %q", got)
	}

	_, err = Dispatch(`$x :new review`, home, homeAlias)
	assertErrorCode(t, err, ErrInvalidInvocation)
}

func TestDispatchSlashCommand(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "foo/bar/baz", "arg=$1")

	got, err := Dispatch(`$x foo/bar/baz value`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "arg=value" {
		t.Fatalf("Dispatch() = %q", got)
	}
}

func TestDispatchFailureProducesNoPartialPrompt(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "first", "first")

	got, err := Dispatch(`$x first + missing`, cwd, home)
	if err == nil {
		t.Fatal("Dispatch() error = nil")
	}
	if got != "" {
		t.Fatalf("Dispatch() partial output = %q", got)
	}
}

func TestDispatchMissingPositionalArgumentNamesCommand(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "compare", "Compare $1 with $2.")

	_, err := Dispatch(`$x compare foo`, cwd, home)
	assertErrorCode(t, err, ErrMissingArgument)
	if !strings.Contains(err.Error(), "compare") {
		t.Fatalf("error = %q, want command name", err)
	}
}

func TestProjectCommandSymlinkIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "src", "project")
	cwd := filepath.Join(project, "sub")
	commands := filepath.Join(project, ".agents", "commands")
	mustMkdirAll(t, commands)
	mustMkdirAll(t, cwd)
	secret := filepath.Join(base, "secret.txt")
	mustWriteFile(t, secret, []byte("do not read"))
	if err := os.Symlink(secret, filepath.Join(commands, "review.md")); err != nil {
		t.Fatal(err)
	}

	_, err := Dispatch(`$x review`, cwd, home)
	assertErrorCode(t, err, ErrUnsafeProjectPath)
}

func TestProjectNestedDirectorySymlinkIsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	base := t.TempDir()
	home := filepath.Join(base, "home")
	project := filepath.Join(home, "src", "project")
	cwd := filepath.Join(project, "sub")
	commands := filepath.Join(project, ".agents", "commands")
	outside := filepath.Join(base, "outside")
	mustMkdirAll(t, commands)
	mustMkdirAll(t, cwd)
	mustMkdirAll(t, outside)
	mustWriteFile(t, filepath.Join(outside, "review.md"), []byte("do not read"))
	if err := os.Symlink(outside, filepath.Join(commands, "code")); err != nil {
		t.Fatal(err)
	}

	_, err := Dispatch(`$x code/review`, cwd, home)
	assertErrorCode(t, err, ErrUnsafeProjectPath)
}

func TestProjectAgentsAndCommandsSymlinksAreRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}

	t.Run("agents", func(t *testing.T) {
		base := t.TempDir()
		home := filepath.Join(base, "home")
		project := filepath.Join(home, "project")
		cwd := filepath.Join(project, "sub")
		outsideAgents := filepath.Join(base, "outside-agents")
		mustMkdirAll(t, filepath.Join(outsideAgents, "commands"))
		mustMkdirAll(t, cwd)
		mustWriteFile(t, filepath.Join(outsideAgents, "commands", "review.md"), []byte("do not read"))
		if err := os.Symlink(outsideAgents, filepath.Join(project, ".agents")); err != nil {
			t.Fatal(err)
		}
		_, err := Dispatch(`$x review`, cwd, home)
		assertErrorCode(t, err, ErrUnsafeProjectPath)
	})

	t.Run("commands", func(t *testing.T) {
		base := t.TempDir()
		home := filepath.Join(base, "home")
		project := filepath.Join(home, "project")
		cwd := filepath.Join(project, "sub")
		outside := filepath.Join(base, "outside")
		mustMkdirAll(t, filepath.Join(project, ".agents"))
		mustMkdirAll(t, cwd)
		mustMkdirAll(t, outside)
		mustWriteFile(t, filepath.Join(outside, "review.md"), []byte("do not read"))
		if err := os.Symlink(outside, filepath.Join(project, ".agents", "commands")); err != nil {
			t.Fatal(err)
		}
		_, err := Dispatch(`$x review`, cwd, home)
		assertErrorCode(t, err, ErrUnsafeProjectPath)
	})
}

func TestUserScopeSymlinkIsAllowed(t *testing.T) {
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
	mustWriteFile(t, filepath.Join(outside, "review.md"), []byte("user symlink"))
	if err := os.Symlink(outside, filepath.Join(commands, "code")); err != nil {
		t.Fatal(err)
	}

	got, err := Dispatch(`$x code/review`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "user symlink" {
		t.Fatalf("Dispatch() = %q", got)
	}
}

func TestInvalidCommandFiles(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	commands := filepath.Join(home, ".agents", "commands")
	mustMkdirAll(t, commands)
	mustMkdirAll(t, cwd)
	mustWriteFile(t, filepath.Join(commands, "empty.md"), nil)
	mustWriteFile(t, filepath.Join(commands, "bad.md"), []byte{0xff, 0xfe})
	mustMkdirAll(t, filepath.Join(commands, "dir.md"))

	for _, command := range []string{"empty", "bad", "dir"} {
		t.Run(command, func(t *testing.T) {
			_, err := Dispatch(`$x `+command, cwd, home)
			assertErrorCode(t, err, ErrInvalidCommandFile)
		})
	}
}

func writeCommand(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, ".agents", "commands", filepath.FromSlash(name)+".md")
	mustMkdirAll(t, filepath.Dir(path))
	mustWriteFile(t, path, []byte(content))
	return path
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
