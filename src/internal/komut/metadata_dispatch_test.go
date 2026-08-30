package komut

import (
	"path/filepath"
	"testing"
)

func TestDispatchStripsFrontmatterBeforeRendering(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "review", "---\ndescription: Review a file\n---\nReview $1.")

	got, err := Dispatch(`$x review src/foo.go`, cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Review src/foo.go." {
		t.Fatalf("Dispatch() = %q", got)
	}
}

func TestDispatchMalformedFrontmatterFails(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	mustMkdirAll(t, cwd)
	writeCommand(t, home, "broken", "---\ndescription: [\n---\nBody")

	_, err := Dispatch(`$x broken`, cwd, home)
	assertErrorCode(t, err, ErrInvalidCommandFile)
}
