package komut

import "testing"

func TestRenderTemplate(t *testing.T) {
	got, err := RenderTemplate("one=$1 two=$2 all=$* dollar=$$ unknown=$x", []string{"alpha", "beta gamma"})
	if err != nil {
		t.Fatal(err)
	}
	want := "one=alpha two=beta gamma all=alpha beta gamma dollar=$ unknown=$x"
	if got != want {
		t.Fatalf("RenderTemplate() = %q, want %q", got, want)
	}
}

func TestRenderTemplateMissingArgument(t *testing.T) {
	_, err := RenderTemplate("Compare $1 with $2.", []string{"foo"})
	assertErrorCode(t, err, ErrMissingArgument)
}

func TestRenderTemplateIsSinglePass(t *testing.T) {
	got, err := RenderTemplate("value=$1", []string{"$2"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "value=$2" {
		t.Fatalf("RenderTemplate() = %q, want %q", got, "value=$2")
	}
}

func TestRenderTemplateDollarDoesNotRecurse(t *testing.T) {
	got, err := RenderTemplate("$$1", []string{"value"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "$1" {
		t.Fatalf("RenderTemplate() = %q, want %q", got, "$1")
	}
}
