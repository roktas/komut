package komut

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLeadIsOpaqueAfterSeparator(t *testing.T) {
	got, err := Parse("$x foo --  a + b -- \"quoted\"\nnext")
	if err != nil {
		t.Fatal(err)
	}
	want := "a + b -- \"quoted\"\nnext"
	if got.Lead != want {
		t.Fatalf("Lead = %q, want %q", got.Lead, want)
	}
}

func TestParseRejectsQuoteConcatenation(t *testing.T) {
	for _, input := range []string{`$x foo "a"b`, `$x foo a"b"`, `$x foo 'a'b`} {
		if _, err := Parse(input); err == nil {
			t.Fatalf("Parse(%q) error = nil", input)
		}
	}
}

func TestRenderDollarEdgeCases(t *testing.T) {
	got, err := RenderTemplate(`$0 $10 $$1 $$$1 $x $`, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `$0 A0 $1 $A $x $`; got != want {
		t.Fatalf("RenderTemplate() = %q, want %q", got, want)
	}
}

func TestDispatchAcceptsCRLFCommandFileWithoutNormalization(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	cwd := filepath.Join(home, "work")
	if err := os.MkdirAll(filepath.Join(home, ".agents", "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".agents", "commands", "foo.md")
	if err := os.WriteFile(path, []byte("A $1\r\nB\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Dispatch("$x foo z", cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if want := "A z\r\nB\r\n"; got != want {
		t.Fatalf("Dispatch() = %q, want %q", got, want)
	}
}

func FuzzParseNeverPanics(f *testing.F) {
	for _, seed := range []string{"", "$x foo", "$x foo + bar -- lead", `$x foo "a + b"`, string([]byte{0xff})} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(input)
	})
}

func FuzzRenderNeverPanics(f *testing.F) {
	for _, seed := range []string{"", "$1", "$$", "$*", "💠$9"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, template string) {
		_, _ = RenderTemplate(template, []string{"a", "b"})
	})
}

func TestErrorTypeIsDiscoverable(t *testing.T) {
	_, err := Parse("$x ../bad")
	xerr, ok := errors.AsType[*Error](err)
	if !ok || xerr.Code != ErrInvalidCommand {
		t.Fatalf("error = %#v", err)
	}
}

func TestDollarStarPreservesParsedArgumentValues(t *testing.T) {
	inv, err := Parse(`$x foo "a b" '' c`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderTemplate(`<$*>`, inv.Commands[0].Args)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<a b  c>" {
		t.Fatalf("got %q", got)
	}
	if strings.Count(got, "  ") != 1 {
		t.Fatalf("empty argument was not preserved in join: %q", got)
	}
}

func TestReadValidFileRejectsIdentityChange(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.md")
	second := filepath.Join(dir, "second.md")
	if err := os.WriteFile(first, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	_, err = readValidFile(second, "foo", expected, ErrUnsafeProjectPath)
	xerr, ok := errors.AsType[*Error](err)
	if !ok || xerr.Code != ErrUnsafeProjectPath {
		t.Fatalf("error = %#v", err)
	}
}
