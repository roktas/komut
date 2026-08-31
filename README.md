# Komut

Komut turns small Markdown prompt files into explicit, composable commands for
AI coding agents.

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

`$x` is the canonical syntax across supported hosts. Komut resolves command
files, substitutes arguments, composes the result, and gives the host one final
prompt for the current turn.

Komut does not ship a command collection. You provide your own Markdown command
files.

## Command files

User-wide commands live in:

```text
~/.agents/commands/
```

Project commands live in:

```text
<project>/.agents/commands/
```

For example, create `~/.agents/commands/review.md`:

```md
---
description: Review code for correctness and compatibility.
---

Review $1.

Pay special attention to correctness and compatibility.
```

Invoke it with:

```text
$x review src/foo.go
```

YAML frontmatter is optional. When present, it is metadata and is not sent to
the agent. `description` is used by help. Without it, help uses the first
non-empty ATX Markdown heading in the command body, if present.

Project commands override user commands with the same name. Komut finds the
nearest project `.agents/commands` directory while walking upward from the
current working directory.

Command names may contain `/`. For example:

```text
$x git/review HEAD~3..HEAD
```

resolves `git/review` as `commands/git/review.md`.

## Builtins

Builtin control commands use the reserved `:` namespace:

```text
$x :help
$x :new code/review
$x :version
```

Help also has these conveniences:

```text
$x
$x help
$x ?
```

All four help forms are equivalent. Help lists builtins and the application
commands visible from the current user and project scopes. Project commands win
duplicate names. If no application commands exist, help shows the absolute
user-wide and project directories where they can be created.

`:new` does not write files or open an editor. It generates an instruction for
the current agent to create or edit a Markdown command file with the agent's
normal file tools:

```text
$x :new code/review
$x :new code/review --project
$x :new git/commit -- Create a Conventional Commits helper.
$x :new review --project -- Review code for correctness and compatibility.
```

The command name is required and is always the first argument. New commands are
user-wide by default and target `~/.agents/commands/`. Use `--project` after the
name only when the command should belong to the current project. `--user` is not
supported because user scope is already the default.

Text after `--` is the optional one-line command description. If it is omitted,
the generated prompt asks the user for a description. The agent always asks the
user for the command body before writing. New commands use YAML `description`
frontmatter followed by the Markdown prompt body, and the target filename always
ends in `.md`.

`:version` reports the installed dispatcher version:

```text
Komut 0.3.1
```

## Syntax

The general form is:

```text
$x [COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]]
```

Use `+` to compose application commands into one prompt:

```text
$x code/review src/foo.go + concise + lang/turkish
```

Use `--` to add free-form lead text. The lead is placed before rendered command
content and is not parsed as Komut syntax:

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

For `:new`, the same `--` marker has builtin-specific meaning: its text is the
command description.

Quote an argument when it contains spaces or when `+` or `--` must be literal:

```text
$x explain "a + b" "+" "--"
```

Command templates support:

```text
$1 ... $9   positional arguments
$*          all arguments joined by one space
$$          a literal $
```

A referenced positional argument that was not supplied is an error. Komut does
not perform shell expansion, globbing, environment expansion, or command
substitution.

See [SPEC.md](SPEC.md) for the complete protocol and security rules.

## Install

Released plugin packages are generated on the `dist` branch. They contain the
platform launchers and prebuilt Go dispatcher, so installation does not require
a Go toolchain.

If this repository is private, make sure the host can clone `roktas/komut` with
your existing Git credentials.

### Codex

Install Komut entirely from the shell:

```sh
codex plugin marketplace add roktas/komut && \
        codex plugin add komut@komut
```

Start a new Codex session after installation.

Update Komut with:

```sh
codex plugin marketplace upgrade komut && \
        codex plugin add komut@komut
```

Codex uses the canonical Komut syntax:

```text
$x review src/foo.go
```

Codex marketplace documentation:
<https://developers.openai.com/plugins/build/plugins>

### Google Antigravity

Install the generated Antigravity plugin from the shell:

```sh
git clone --branch dist --single-branch https://github.com/roktas/komut.git komut-dist && \
        agy plugin install ./komut-dist/plugins/antigravity
```

Start a new Antigravity session after installation.

Antigravity does not currently document a separate plugin-update command. Update
the distribution checkout and reinstall the staged plugin:

```sh
git -C komut-dist pull --ff-only origin dist && \
        agy plugin uninstall komut && \
        agy plugin install ./komut-dist/plugins/antigravity
```

For Antigravity 2.0 without the CLI, copy
`komut-dist/plugins/antigravity` to `~/.gemini/config/plugins/komut`.

The plugin exposes the native Antigravity skill command:

```text
/x review src/foo.go
```

With no arguments, `/x` opens Komut help. The canonical `$x ...` syntax also
works. Both forms use the installed `x` skill, which runs the packaged dispatcher
from the current session working directory and follows its output.

Antigravity plugin documentation:
<https://antigravity.google/docs/plugins>
<https://antigravity.google/docs/cli/plugins/>

### Claude Code

Install Komut entirely from the shell:

```sh
claude plugin marketplace add roktas/komut && \
        claude plugin install komut@komut
```

Start a new Claude Code session after installation.

Update Komut with:

```sh
claude plugin marketplace update komut && \
        claude plugin update komut@komut
```

The plugin exposes the native Claude Code skill command:

```text
/komut:x review src/foo.go
```

With no arguments, `/komut:x` opens Komut help. The canonical `$x ...` syntax
also works in Claude Code.

Claude Code plugin skills are namespaced by plugin and skill name, hence
`/komut:x` for the `komut` plugin's `x` skill.

Claude Code documentation:
<https://code.claude.com/docs/en/discover-plugins>
<https://code.claude.com/docs/en/plugins-reference>

### OpenCode V2

Install the generated OpenCode package from the shell:

```sh
opencode2 plugin add "github:roktas/komut#dist::path:plugins/opencode"
```

Restart the OpenCode service or start a new OpenCode session after installation.

Unpinned Git plugins are refreshed in the background when OpenCode starts. For a
deterministic immediate update, reinstall the package and restart the service:

```sh
opencode2 plugin remove "github:roktas/komut#dist::path:plugins/opencode" && \
        opencode2 plugin add "github:roktas/komut#dist::path:plugins/opencode" && \
        opencode2 service restart
```

The plugin registers a native OpenCode command:

```text
/x review src/foo.go
```

With no arguments, `/x` opens Komut help. The canonical `$x ...` syntax also
works in OpenCode.

OpenCode V2 plugin documentation:
<https://opencode.ai/v2/docs/plugins>
<https://opencode.ai/v2/docs/build/plugins/>

## Supported platforms

Generated packages currently include dispatcher binaries for:

- macOS arm64
- macOS amd64
- Linux arm64
- Linux amd64
- Windows amd64

Each host package contains the same dispatcher. Host adapters only connect host
invocation mechanisms to that dispatcher.

## Development

Komut uses Go 1.26 as its minimum development version. Installed plugins do not
require Go.

Run the core checks with:

```sh
go -C src test -race ./... && go -C src vet ./...
```

Run repository-level product and integration tests with:

```sh
go -C tests test ./...
```

Build all self-contained host packages with:

```sh
src/dist/build ./dist
```

On Linux, smoke-test generated packages with:

```sh
src/dist/smoke ./dist
```

CI also tests the Windows launcher and Claude hook wrapper on a native Windows
runner.

The `dist` branch is generated. Do not edit it by hand.

## License

GPL-3.0. See [LICENSE](LICENSE).
