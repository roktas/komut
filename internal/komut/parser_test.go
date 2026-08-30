package komut

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseCompositionAndLead(t *testing.T) {
	got, err := Parse(`  $x code/review "src/foo bar.rb" strict + concise + lang/turkish -- This is a public API.`)
	if err != nil {
		t.Fatal(err)
	}

	want := Invocation{
		Commands: []Command{
			{Name: "code/review", Args: []string{"src/foo bar.rb", "strict"}},
			{Name: "concise"},
			{Name: "lang/turkish"},
		},
		Lead: "This is a public API.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse() = %#v, want %#v", got, want)
	}
}

func TestParseQuotedSpecialTokensAreArguments(t *testing.T) {
	got, err := Parse(`$x explain "+" '--' "a + b"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"+", "--", "a + b"}
	if !reflect.DeepEqual(got.Commands[0].Args, want) {
		t.Fatalf("args = %#v, want %#v", got.Commands[0].Args, want)
	}
}

func TestParseSpecialSyntaxNeedsTokenBoundary(t *testing.T) {
	got, err := Parse(`$x explain C++ a+b --flag + concise`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"C++", "a+b", "--flag"}
	if !reflect.DeepEqual(got.Commands[0].Args, want) {
		t.Fatalf("args = %#v, want %#v", got.Commands[0].Args, want)
	}
}

func TestParseLeadStopsGrammar(t *testing.T) {
	got, err := Parse(`$x foo a + bar b -- Explain a + b -- literally "without parsing".`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Lead != `Explain a + b -- literally "without parsing".` {
		t.Fatalf("lead = %q", got.Lead)
	}
	if len(got.Commands) != 2 {
		t.Fatalf("commands = %d, want 2", len(got.Commands))
	}
}

func TestParseQuotesAndEscapes(t *testing.T) {
	got, err := Parse(`$x foo 'a b' "c d" "quote: \"x\"" "path\\name"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a b", "c d", `quote: "x"`, `path\name`}
	if !reflect.DeepEqual(got.Commands[0].Args, want) {
		t.Fatalf("args = %#v, want %#v", got.Commands[0].Args, want)
	}
}

func TestParseRejectsBadInvocations(t *testing.T) {
	tests := []struct {
		name string
		text string
		code ErrorCode
	}{
		{name: "prefix boundary", text: `$xfoo bar`, code: ErrInvalidInvocation},
		{name: "missing command", text: `$x `, code: ErrInvalidInvocation},
		{name: "leading plus", text: `$x + foo`, code: ErrInvalidInvocation},
		{name: "trailing plus", text: `$x foo +`, code: ErrInvalidInvocation},
		{name: "unterminated single quote", text: `$x foo 'bar`, code: ErrUnterminatedQuote},
		{name: "unterminated double quote", text: `$x foo "bar`, code: ErrUnterminatedQuote},
		{name: "quote concatenation", text: `$x foo "bar"baz`, code: ErrInvalidInvocation},
		{name: "invalid command", text: `$x ../secret`, code: ErrInvalidCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.text)
			assertErrorCode(t, err, tt.code)
		})
	}
}

func TestParseRejectsInvalidUTF8(t *testing.T) {
	_, err := Parse(string([]byte{'$', 'x', ' ', 'f', 'o', 'o', ' ', 0xff}))
	assertErrorCode(t, err, ErrInvalidInvocation)
}

func TestValidCommandName(t *testing.T) {
	valid := []string{"foo", "foo/bar", "foo/bar-baz", "foo2/bar3/baz"}
	invalid := []string{"", "/foo", "foo/", "foo//bar", "foo/../bar", "foo/./bar", "foo/_bar", "foo/bar_", "foo/ba--r", "Foo/bar"}

	for _, name := range valid {
		if !ValidCommandName(name) {
			t.Errorf("ValidCommandName(%q) = false", name)
		}
	}
	for _, name := range invalid {
		if ValidCommandName(name) {
			t.Errorf("ValidCommandName(%q) = true", name)
		}
	}
}

func assertErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s", want)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if e.Code != want {
		t.Fatalf("error code = %s, want %s (%v)", e.Code, want, err)
	}
}
