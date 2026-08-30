package main

import (
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
