package komut_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCodexPluginManifest(t *testing.T) {
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	readJSON(t, "src/adapters/codex/.codex-plugin/plugin.json", &manifest)
	if manifest.Name != "komut" || manifest.Version != productVersion(t) {
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
	readJSON(t, "src/adapters/codex/hooks/hooks.json", &config)

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
	assertMarketplaceEntry(t, ".agents/plugins/marketplace.json", "plugins/codex")
}

func TestClaudePluginManifest(t *testing.T) {
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	readJSON(t, "src/adapters/claude/.claude-plugin/plugin.json", &manifest)
	if manifest.Name != "komut" || manifest.Version != productVersion(t) {
		t.Fatalf("unexpected Claude manifest: %#v", manifest)
	}
}

func TestClaudeHooksUseCrossPlatformWrapper(t *testing.T) {
	type hookGroup struct {
		Hooks []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
			Timeout int    `json:"timeout"`
		} `json:"hooks"`
	}
	var config struct {
		Hooks struct {
			UserPromptSubmit    []hookGroup `json:"UserPromptSubmit"`
			UserPromptExpansion []hookGroup `json:"UserPromptExpansion"`
		} `json:"hooks"`
	}
	readJSON(t, "src/adapters/claude/hooks/hooks.json", &config)

	for name, groups := range map[string][]hookGroup{
		"UserPromptSubmit":    config.Hooks.UserPromptSubmit,
		"UserPromptExpansion": config.Hooks.UserPromptExpansion,
	} {
		if len(groups) != 1 || len(groups[0].Hooks) != 1 {
			t.Fatalf("unexpected Claude %s hook layout", name)
		}
		hook := groups[0].Hooks[0]
		if hook.Type != "command" || hook.Command != `"${CLAUDE_PLUGIN_ROOT}/hooks/run.cmd"` || hook.Timeout != 5 {
			t.Fatalf("unexpected Claude %s hook: %#v", name, hook)
		}
	}

	source := string(readRepoFile(t, "src/adapters/claude/hooks/run.cmd"))
	for _, required := range []string{
		`exec "$CLAUDE_PLUGIN_ROOT/bin/x" --hook`,
		`call "%CLAUDE_PLUGIN_ROOT%\bin\x.cmd" --hook`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Claude wrapper is missing %q", required)
		}
	}
}

func TestClaudeNativeSkill(t *testing.T) {
	source := string(readRepoFile(t, "src/adapters/claude/skills/x/SKILL.md"))
	for _, required := range []string{
		"name: x",
		"disable-model-invocation: true",
		"argument-hint: [command-and-arguments]",
		"$ARGUMENTS",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Claude native skill is missing %q", required)
		}
	}
}

func TestClaudeMarketplaceTargetsDistBranch(t *testing.T) {
	assertMarketplaceEntry(t, ".claude-plugin/marketplace.json", "plugins/claude")
}

func TestAntigravityPluginManifest(t *testing.T) {
	var manifest struct {
		Schema      string `json:"$schema"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	readJSON(t, "src/adapters/antigravity/plugin.json", &manifest)
	if manifest.Schema != "https://antigravity.google/schemas/v1/plugin.json" || manifest.Name != "komut" || manifest.Description == "" {
		t.Fatalf("unexpected Antigravity manifest: %#v", manifest)
	}
}

func TestAntigravityNativeSkill(t *testing.T) {
	source := string(readRepoFile(t, "src/adapters/antigravity/skills/x/SKILL.md"))
	for _, required := range []string{
		"name: x",
		"`$x`",
		"`/x`",
		"`../../bin/x`",
		"`../../bin/x.cmd`",
		"current session working directory",
		"standard input",
		"operative instruction",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Antigravity skill is missing %q", required)
		}
	}
	for _, forbidden := range []string{"--hook", "transcript"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("Antigravity skill contains unsupported transport %q", forbidden)
		}
	}
}

func TestOpenCodePackage(t *testing.T) {
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
		Exports struct {
			Root string `json:"."`
		} `json:"exports"`
		Dependencies map[string]string `json:"dependencies"`
	}
	readJSON(t, "src/adapters/opencode/package.json", &manifest)
	if manifest.Name != "komut-opencode" || manifest.Version != productVersion(t) || manifest.Type != "module" || manifest.Exports.Root != "./src/index.js" {
		t.Fatalf("unexpected OpenCode manifest: %#v", manifest)
	}
	if manifest.Dependencies["@opencode-ai/plugin"] != "beta" {
		t.Fatalf("unexpected OpenCode plugin dependency: %#v", manifest.Dependencies)
	}

	source := string(readRepoFile(t, "src/adapters/opencode/src/index.js"))
	for _, required := range []string{
		`ctx.command.transform`,
		`draft.add({`,
		`name: "x"`,
		`text: invocation(prompt.text),`,
		`ctx.session.hook("prompt"`,
		`ctx.session.get({ sessionID: event.sessionID })`,
		`event.prompt.text = expand(event.prompt.text, session.directory);`,
		`join(root, "bin", process.platform === "win32" ? "x.cmd" : "x")`,
		`spawnSync(launcherPath()`,
		`/^\s*\$x(?:\s|$)/u`,
		"return args.trim() === \"\" ? \"$x\" : `$x ${args}`;",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("OpenCode adapter is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		`ctx.location.directory`,
		`session.location`,
		`text: expand(invocation(prompt.text)`,
		`libexec`,
		`process.arch`,
		`const text = args.trim();`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("OpenCode adapter contains stale or duplicated logic %q", forbidden)
		}
	}
}

func productVersion(t *testing.T) string {
	t.Helper()
	data := readRepoFile(t, "src/internal/komut/version.go")
	match := regexp.MustCompile(`const Version = "([^"]+)"`).FindSubmatch(data)
	if len(match) != 2 {
		t.Fatal("cannot read product version")
	}
	return string(match[1])
}

func assertMarketplaceEntry(t *testing.T, path, pluginPath string) {
	t.Helper()
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
	readJSON(t, path, &marketplace)
	if len(marketplace.Plugins) != 1 {
		t.Fatalf("%s plugins = %d, want 1", path, len(marketplace.Plugins))
	}
	plugin := marketplace.Plugins[0]
	if plugin.Name != "komut" || plugin.Source.Kind != "git-subdir" || plugin.Source.URL != "roktas/komut" || plugin.Source.Path != pluginPath || plugin.Source.Ref != "dist" {
		t.Fatalf("unexpected marketplace entry in %s: %#v", path, plugin)
	}
}

func readJSON(t *testing.T, path string, dst any) {
	t.Helper()
	data := readRepoFile(t, path)
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func readRepoFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(repoPath(path))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repoPath(path string) string {
	return filepath.Join("..", filepath.FromSlash(path))
}
