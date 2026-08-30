# Komut

Komut turns small Markdown prompt files into explicit, composable commands for
AI coding agents.

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

The same `$x` syntax works across supported hosts. Komut resolves the command
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

Invoke it in the agent prompt with:

```text
$x review src/foo.go
```

YAML frontmatter is optional. When present, it is metadata and is not sent to
the agent. `description` is used by the builtin help command. Without a
frontmatter description, help uses the first non-empty ATX Markdown heading in
the command body, if present.

Project commands override user commands with the same name. Komut finds the
nearest project `.agents/commands` directory while walking upward from the
current working directory.

Command names may contain `/`. A command such as:

```text
$x git/review HEAD~3..HEAD
```

resolves `git/review` as:

```text
commands/git/review.md
```

## Help

In the agent prompt, list commands available from the current user and project
scopes with:

```text
$x help
```

Project commands win duplicate names. Help sorts commands by name and shows the
frontmatter description, first ATX heading, or no description in that order.
If no commands exist, it shows the absolute user-wide and project command
directories where commands can be created.

`help` is reserved; a `help.md` file cannot override the builtin.

## Syntax

The general form is:

```text
$x COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]
```

Use `+` to compose commands into one prompt:

```text
$x code/review src/foo.go + concise + lang/turkish
```

Use `--` to add free-form lead text. The lead is placed before the rendered
commands and is not parsed as Komut syntax:

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

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

See [SPEC.md](SPEC.md) for the complete grammar, resolution rules, metadata
contract, builtin behavior, and security rules.

## Install

Released plugin packages are generated on the `dist` branch. They contain the
platform launchers and prebuilt Go dispatcher, so installation does not require
a Go toolchain.

If this repository is private, make sure the host can clone `roktas/komut` with
your existing Git credentials.

### Codex

Add the Komut marketplace:

```sh
codex plugin marketplace add roktas/komut
```

Start Codex and open the plugin browser:

```text
/plugins
```

Select the **Komut** marketplace, install `komut`, then start a new Codex
session.

In the Codex prompt, invoke a command with:

```text
$x review src/foo.go
```

Codex marketplace documentation:
<https://developers.openai.com/plugins/build/plugins>

### Claude Code

Inside Claude Code, add the marketplace and install Komut:

```text
/plugin marketplace add roktas/komut
/plugin install komut@komut
```

If Claude Code asks you to reload plugins, run:

```text
/reload-plugins
```

In the Claude Code prompt, invoke a command with:

```text
$x review src/foo.go
```

Claude Code marketplace documentation:
<https://code.claude.com/docs/en/discover-plugins>

### OpenCode V2

Komut currently targets the OpenCode V2 plugin API. Install the generated
OpenCode package directly from the `dist` branch:

```sh
opencode2 plugin add "github:roktas/komut#dist::path:plugins/opencode"
```

Restart the OpenCode service or start a new OpenCode session after installation.

In the OpenCode prompt, invoke a command with:

```text
$x review src/foo.go
```

OpenCode V2 plugin documentation:
<https://opencode.ai/v2/docs/plugins>

## Supported platforms

The generated packages currently include dispatcher binaries for:

- macOS arm64
- macOS amd64
- Linux arm64
- Linux amd64
- Windows amd64

Each host package contains the same dispatcher. Host adapters only connect the
host prompt lifecycle to that dispatcher.

## Development

Komut uses Go 1.26 as its minimum development version. The installed plugin does
not require Go.

Run the core checks with:

```sh
go test -race ./...
go vet ./...
```

Build all self-contained host packages with:

```sh
sh scripts/build-dist.sh ./dist
```

On Linux, smoke-test the generated packages with:

```sh
sh scripts/smoke-dist.sh ./dist
```

CI also tests the Windows launcher and Claude hook wrapper on a native Windows
runner.

The `dist` branch is generated. Do not edit it by hand.

## License

GPL-3.0. See [LICENSE](LICENSE).
