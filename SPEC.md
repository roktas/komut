# Komut Specification

Version: 0.1.0

## 1. Purpose

Komut provides a small, namespaced prompt-command protocol for AI agents.

A user writes:

```text
$x COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]
```

Komut resolves each command to a Markdown prompt file, substitutes command
arguments into that file, composes all rendered commands, and returns one final
prompt for the current agent turn.

Komut owns the command grammar, resolution rules, rendering rules, and the
`x` dispatcher. It does not ship application commands. Users and projects
provide their own command files.

Host integrations for Codex, Claude Code, OpenCode, and other agents are thin
adapters around the same dispatcher semantics.

## 2. Design principles

Komut follows these principles:

- `$x` is the stable public namespace across hosts.
- One invocation produces one final prompt and one agent turn.
- Command files are plain Markdown data, not programs.
- Parsing, resolution, and rendering belong to the central dispatcher, not host
  adapters.
- Project commands may override user commands without changing user files.
- Host-specific packaging must not change command semantics.
- The normal installed path must not require Go, Python, Ruby, Node.js, or a
  network download at runtime.

## 3. Invocation grammar

The logical grammar is:

```ebnf
invocation = "$x", ws, command, { ws, "+", ws, command },
             [ ws, "--", [ ws, lead ] ] ;

command    = name, { ws, argument } ;
name       = segment, { "/", segment } ;
argument   = bare | single_quoted | double_quoted ;
lead       = <remaining input> ;
```

Leading whitespace before `$x` is allowed. `$x` must be followed by whitespace.
`$xfoo` is not an invocation.

### 3.1 Command composition

An unquoted `+` token separates commands:

```text
$x code/review src/foo.rb + concise + lang/turkish
```

The commands are rendered independently and then composed in the declared
order. They are not a pipeline. A later command does not receive the output of
an earlier command as an argument.

`+` is syntax only when it is an unquoted token separated by whitespace. It is
ordinary text inside an argument such as `C++`, `a+b`, or `"a + b"`.

If any command fails to parse, resolve, read, or render, the whole invocation
fails. Komut must not return a partial composed prompt.

### 3.2 Lead text

The first unquoted `--` token ends Komut parsing for the whole invocation. The
remaining text is lead text:

```text
$x code/review src/foo.rb + concise -- This is a public API.
```

After `--`, `+`, `--`, quotes, and `$` sequences have no Komut grammar meaning.
The lead is not a command argument and is not used for template substitution.

A non-empty lead is placed before the rendered command content in the final
prompt.

### 3.3 Arguments and quoting

Whitespace separates arguments unless the text is quoted.

Single quotes preserve their contents literally until the closing single quote.
Double quotes preserve whitespace and allow `\"` and `\\` escapes. Komut does
not implement shell expansion, environment expansion, globbing, command
substitution, or any other shell feature.

An unquoted `+` or `--` token is syntax, not an argument. Quoted `"+"` and
`"--"` are ordinary arguments.

## 4. Command names and paths

A command name is one or more slash-separated segments:

```text
review
code/review
git/commit
foo/bar-baz
```

The complete command name must be at most 64 ASCII characters, including
slashes.

Each segment must:

- contain only lowercase ASCII letters, digits, and `-`;
- start and end with an ASCII letter or digit;
- contain at least one character;
- not contain `--`.

A command name must not start or end with `/` and must not contain `//`.
Characters such as `.`, `\\`, `_`, and whitespace are not allowed in command
names. These rules make path traversal impossible by construction.

A command name maps directly to a Markdown path. For example:

```text
foo/bar/baz
```

maps to:

```text
commands/foo/bar/baz.md
```

Komut does not enumerate command directories, try alternative names, or perform
fuzzy matching.

## 5. Command locations and scope

User commands live under:

```text
~/.agents/commands/
```

Project commands live under:

```text
<project>/.agents/commands/
```

Project scope is found by walking upward from the current working directory and
selecting the nearest `.agents/commands` path.

The user's home directory is not treated as project scope. The user command tree
at `~/.agents/commands` is resolved separately.

For each command, resolution order is:

1. the command path in the nearest project command tree, if such a tree exists;
2. the command path in the user command tree;
3. not found.

The nearest project `.agents/commands` tree is a scope boundary. If it exists but
does not contain the requested command, Komut falls back directly to user scope.
It must not continue searching more distant project command trees.

A project command shadows a user command with the same command name.

All commands in one invocation use the same project-scope discovery result.
Komut should perform the ancestor walk once per invocation.

## 6. Command file contract

A command file must be a readable, non-empty, UTF-8 regular Markdown file.
Komut treats the complete file contents as a prompt template.

Komut does not parse YAML frontmatter or any other command metadata layer.
Command files are data and are never executed.

### 6.1 Template substitution

The dispatcher recognizes these substitutions in command file content:

```text
$1 ... $9   positional command arguments
$*          all command arguments joined by one space
$$          a literal $
```

Each command has its own positional arguments. For example:

```text
$x foo a b + bar c
```

renders `foo` with `$1 = a`, `$2 = b`, and renders `bar` with `$1 = c`.

A referenced positional argument that was not supplied is an error. Komut must
not silently replace a missing argument with an empty string.

Substitution is single-pass. Text introduced by an argument is not scanned again
for substitutions.

A `$` sequence that does not match one of the forms above remains literal.

## 7. Prompt composition

Each command file is resolved and rendered independently. The rendered command
contents are then joined in invocation order with two LF characters (`\n\n`)
between components.

If a non-empty lead is present, it is the first component and is separated from
the first rendered command by the same two-LF separator.

Komut otherwise preserves command file content. It does not interpret or merge
instructions semantically.

For example:

```text
$x code/review src/foo.rb + concise -- This is a public API.
```

conceptually produces:

```text
This is a public API.

<rendered code/review.md>

<rendered concise.md>
```

## 8. Filesystem safety

Project command trees are repository-controlled input and receive stricter path
checks than user scope.

For project scope, Komut must reject:

- a symbolic-link `.agents` path;
- a symbolic-link `commands` path;
- any symbolic-link intermediate directory selected by a slash command name;
- a symbolic-link selected command file;
- a selected path component with an unexpected file type.

The selected project command must remain inside the selected project
`.agents/commands` tree.

User-scope symbolic links are allowed because they are controlled by the user
and are useful for dotfile management. The final selected user command must
still resolve to a readable regular file.

A malformed or unsafe path must fail closed. Komut must never read an unrelated
file as a command.

## 9. Dispatcher

The central implementation is the `x` dispatcher written in Go.

The dispatcher owns:

- invocation parsing;
- command-name validation;
- project and user resolution;
- command-file validation and reading;
- template substitution;
- multi-command composition;
- final prompt generation.

Host adapters must not implement a second copy of these semantics.

The dispatcher operates on the full `$x ...` invocation text and emits the final
rendered prompt. The exact process transport used by host adapters is an
implementation detail, but the rendered result must be identical across hosts
for the same invocation, working directory, home directory, and command files.

## 10. Host adapters

Host adapters connect native host extension points to the central dispatcher.
They should do only the minimum host-specific work required to:

1. recognize an explicit `$x` invocation cheaply;
2. invoke the packaged dispatcher with the current prompt and working directory;
3. inject the rendered prompt into the same agent turn;
4. surface dispatcher errors without attempting a second resolution path.

Adapters may use hooks, prompt interception, native commands, or another host
mechanism as required by that host. These mechanisms are not part of the Komut
command protocol.

Native slash-command aliases may be added by a host adapter, but `$x` remains
the canonical cross-host syntax.

Host-specific plugin packages live in clearly separated repository subtrees,
for example:

```text
plugins/
├── codex/
├── claude/
└── opencode/
```

Each installed host package must be self-contained. It must not rely on files
outside its installed plugin subtree.

## 11. Binary distribution

Komut is implemented in Go but installed users must not need a Go toolchain.
Supported host packages contain prebuilt dispatcher binaries.

The package-level executable layout is:

```text
bin/
├── x
└── x.cmd

libexec/
└── x/
    ├── darwin-arm64/x
    ├── darwin-amd64/x
    ├── linux-arm64/x
    ├── linux-amd64/x
    └── windows-amd64/x.exe
```

`bin/x` is a small POSIX shell launcher. `bin/x.cmd` is a small Windows command
launcher. A launcher only detects the local OS/architecture and transfers
control to the matching prebuilt binary under `libexec/x/`.

Launchers do not parse Komut syntax and do not implement command semantics.

Normal plugin installation must not compile Go code, download an executable on
first use, or modify unrelated host configuration files.

## 12. Repository organization

The source repository separates the core implementation from host packaging:

```text
komut/
├── cmd/
│   └── x/
├── internal/
│   └── komut/
├── bin/
│   ├── x
│   └── x.cmd
├── plugins/
│   ├── codex/
│   ├── claude/
│   └── opencode/
├── scripts/
├── AGENTS.md
├── SPEC.md
└── README.md
```

The exact internal Go package split may evolve. The protocol and behavioral
contracts in this specification are authoritative over directory convenience.

Host package build/release steps may copy the launchers and prebuilt binaries
into each self-contained plugin subtree. Generated package copies must not become
independent sources of command semantics.

## 13. Performance requirements

Dispatch is on the interactive prompt path and should remain small and
predictable.

The implementation must:

- avoid network I/O during dispatch;
- avoid command-directory enumeration;
- perform at most one ancestor walk per invocation;
- read only the command files selected by the invocation;
- avoid starting subprocesses from the Go dispatcher;
- render all commands before returning one final prompt;
- keep non-`$x` host-adapter handling cheap.

No fixed wall-clock target is specified because process startup and filesystem
costs vary by platform and host.

## 14. Errors

At minimum, Komut must distinguish these error classes internally and present a
concise diagnostic to the host:

- invalid invocation syntax;
- invalid command name;
- command not found;
- unsafe project path;
- invalid command file;
- unterminated quote;
- missing positional argument;
- unsupported launcher platform.

Errors must not produce a partial rendered prompt.

## 15. Acceptance criteria

The core implementation is complete when tests verify at least:

- leading whitespace before `$x` is accepted;
- `$xfoo` is not accepted as `$x`;
- a single command renders correctly;
- multiple commands separated by `+` render in order into one prompt;
- quoted `+` and `--` remain arguments;
- `--` stops parsing globally and preserves the remaining lead text;
- lead text is placed before rendered commands;
- single- and double-quoted arguments work as specified;
- `$1` through `$9`, `$*`, and `$$` render correctly;
- a missing referenced positional argument fails the whole invocation;
- substitution is not recursive;
- slash command names resolve to nested Markdown paths;
- traversal-like and otherwise invalid command names are rejected;
- project commands override user commands;
- a missing nearest-project command falls back directly to user scope;
- the nearest project command tree is a scope boundary;
- the home command tree is not misclassified as project scope;
- project symlinks, including nested command-path symlinks, are rejected;
- user-scope symlinks remain usable when they resolve to regular files;
- empty, unreadable, non-UTF-8, and non-regular command files fail;
- one invocation performs one project ancestor walk;
- failure of any composed command produces no partial output;
- launchers select the expected packaged binary for each supported platform;
- host adapters produce the same rendered prompt semantics for equivalent input.
