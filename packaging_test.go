package komut_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCodexPluginManifest(t *testing.T) {
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	readJSON(t, "plugins/codex/.codex-plugin/plugin.json", &manifest)
	if manifest.Name != "komut" || manifest.Version != "0.1.0" {
		t.Fatalf("unexpected Codex manifest: %#v", manifest)
	}
}

func TestCodexHookUsesPackagedDispatcher(t *testing.T) {
	var config struct {
		Hooks struct {
			UserPromptSubmit []struct {
				Hooks []struct {
					Type           string `json:"type"`
					Command        string `json:"command"`
					CommandWindows string `json:"commandWindows"`
				} `json:"hooks"`
			} `json:"UserPromptSubmit"`
		} `json:"hooks"`
	}
	readJSON(t, "plugins/codex/hooks/hooks.json", &config)

	groups := config.Hooks.UserPromptSubmit
	if len(groups) != 1 || len(groups[0].Hooks) != 1 {
		t.Fatalf("unexpected UserPromptSubmit hook layout")
	}
	hook := groups[0].Hooks[0]
	if hook.Type != "command" || hook.Command != `"$PLUGIN_ROOT/bin/x" --hook` || hook.CommandWindows != `"%PLUGIN_ROOT%\bin\x.cmd" --hook` {
		t.Fatalf("unexpected Codex hook: %#v", hook)
	}
}

func TestCodexMarketplaceTargetsDistBranch(t *testing.T) {
	var marketplace struct {
		Plugins []struct {
			Name   string `json:"name"`
			Source struct {
				Kind string `json:"source"`
				URL  string `json:"url"`
				Path string `json:"path"`
				Ref  string `json:"ref"`
			} `json:"source"`
		} `json:"plugins"`
	}
	readJSON(t, ".agents/plugins/marketplace.json", &marketplace)
	if len(marketplace.Plugins) != 1 {
		t.Fatalf("plugins = %d, want 1", len(marketplace.Plugins))
	}
	plugin := marketplace.Plugins[0]
	if plugin.Name != "komut" || plugin.Source.Kind != "git-subdir" || plugin.Source.URL != "roktas/komut" || plugin.Source.Path != "plugins/codex" || plugin.Source.Ref != "dist" {
		t.Fatalf("unexpected marketplace entry: %#v", plugin)
	}
}

func readJSON(t *testing.T, path string, dst any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}
