package komut_test

import "testing"

func TestClaudeExpansionHookIsScoped(t *testing.T) {
	var config struct {
		Hooks struct {
			UserPromptExpansion []struct {
				Matcher string `json:"matcher"`
			} `json:"UserPromptExpansion"`
		} `json:"hooks"`
	}
	readJSON(t, "plugins/claude/hooks/hooks.json", &config)

	groups := config.Hooks.UserPromptExpansion
	if len(groups) != 1 {
		t.Fatalf("UserPromptExpansion groups = %d, want 1", len(groups))
	}
	if groups[0].Matcher != `^(komut:x|x)$` {
		t.Fatalf("UserPromptExpansion matcher = %q", groups[0].Matcher)
	}
}
