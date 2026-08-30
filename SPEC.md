# Komut Specification

Version: 0.3.0

## 1. Purpose

Komut provides a small, namespaced prompt-command protocol for AI agents.

A user writes:

```text
$x COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]
```

Komut resolves Markdown command files, substitutes arguments, composes rendered
commands, and returns one final prompt for the current agent turn. Komut also
provides builtin control commands under the reserved `:` namespace.

Komut owns the command grammar, resolution rules, rendering rules, builtin
commands, and the `x` dispatcher. It does not ship application commands. Users
and projects provide their own command files.

Host integrations for Codex, Claude Code, OpenCode, and other agents are thin
adapters around the same dispatcher semantics.

## 2. Design principles

- `$x` is the stable public namespace across hosts.
- One ordinary invocation produces one final prompt and one agent turn.
- Application commands and builtin control commands use separate namespaces.
- Command files are Markdown prompt data with optional metadata, not programs.
- Parsing, resolution, metadata handling, rendering, and builtins belong to the
  central dispatcher, not host adapters.
- Project commands may override user commands without changing user files.
- Builtins may generate prompts but should not perform agent work themselves.
- Host-specific packaging must not change command semantics.
- Normal installation must not require Go, Python, Ruby, Node.js, or a network
  download at runtime.

## 3. Invocation grammar

The logical grammar is:

```ebnf
invocation = "$x", ws, command, { ws, "+", ws, command },
             [ ws, "--", [ ws, lead ] ] ;

command     = builtin | application ;
builtin     = builtin_name, { ws, argument } ;
application = name, { ws, argument } ;
builtin_name = ":", builtin_segment ;
name        = segment, { "/", segment } ;
argument    = bare | single_quoted | double_quoted ;
lead        = <remaining input> ;
```

Leading whitespace before `$x` is allowed. `$x` must be followed by whitespace.
`$xfoo` is not an invocation.

Builtin names begin with `:`. A builtin segment starts with a lowercase ASCII
letter and then contains only lowercase ASCII letters, digits, and `-`.
Application command names never begin with `:`.

Two exact aliases are defined:

```text
$x help  -> $x :help
$x ?     -> $x :help
```

The aliases are parser-level conveniences. `:help` is the canonical builtin
name.

### 3.1 Command composition

An unquoted standalone `+` token separates application commands:

```text
$x code/review src/foo.go + concise + lang/turkish
```

Application commands are rendered independently and composed in declared order.
They are not a pipeline.

Builtin commands are control-plane operations and must be invoked alone. A
builtin cannot be composed with another builtin or an application command.

`+` is ordinary text inside values such as `C++`, `a+b`, or `"a + b"`.
If any composed application command fails, the whole invocation fails. Komut
must not return a partial prompt.

### 3.2 Lead text

The first unquoted standalone `--` token ends Komut parsing globally:

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

Everything after `--` is opaque lead text. It is not parsed for `+`, `--`,
quotes, or template substitutions. A non-empty lead is placed before rendered
application command content.

A builtin may define its own meaning for lead text. `:help` forbids lead text.
`:create` uses lead text as optional authoring intent for the command to create.

### 3.3 Arguments and quoting

Whitespace separates arguments unless text is quoted. Single quotes preserve
their contents literally. Double quotes preserve whitespace and allow `\"` and
`\\` escapes.

Komut does not implement shell expansion, environment expansion, globbing,
command substitution, or other shell features. Quoted `"+"` and `"--"` are
ordinary arguments.

## 4. Command names and paths

An application command name is one or more slash-separated segments:

```text
review
code/review
git/commit
foo/bar-baz
```

The complete name must be at most 64 ASCII characters, including slashes.
Each segment must:

- contain only lowercase ASCII letters, digits, and `-`;
- start and end with an ASCII letter or digit;
- contain at least one character;
- not contain `--`.

A name must not start or end with `/` or contain `//`. Characters such as `.`,
`\\`, `_`, `:`, and whitespace are not allowed.

A name maps directly to a Markdown path. `foo/bar/baz` maps to:

```text
commands/foo/bar/baz.md
```

Except for builtin operations that explicitly enumerate commands, Komut does not
enumerate command directories, try alternative names, or perform fuzzy matching
during normal dispatch.

The application name `help` is reserved because it is an alias for `:help`.
A `help.md` file cannot override the builtin and is not listed as an application
command.

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
selecting the nearest `.agents/commands` directory. The user's home directory is
not treated as project scope.

For each ordinary application command, resolution order is:

1. the command in the nearest project command tree, if present;
2. the command in the user command tree;
3. not found.

The nearest project command tree is a scope boundary. If it does not contain the
requested command, Komut falls back directly to user scope and does not search
more distant project command trees.

A project command shadows a user command with the same name. All commands in one
invocation use the same project-scope discovery result. The ancestor walk should
run once per invocation.

When a builtin needs a project command target and no project command tree exists,
it uses `<cwd>/.agents/commands` as the project target that would establish scope
at the current working directory.

## 6. Command file contract

A command file must be a readable, non-empty, UTF-8 regular Markdown file. It
may begin with optional YAML frontmatter. When frontmatter is present, the
frontmatter is metadata and the remaining Markdown body is the prompt template.
The metadata is never sent to the agent.

Command files are data and are never executed.

### 6.1 YAML frontmatter

Frontmatter is recognized only at the start of the file and is delimited by
lines containing `---`:

```md
---
description: Review code for correctness and maintainability.
---

Review $1.
```

The frontmatter must be valid YAML. Komut currently defines one metadata field:

```yaml
description: <string>
```

`description` is optional. Unknown frontmatter fields are allowed and ignored by
this specification. A malformed frontmatter block makes the command file
invalid for ordinary command execution.

A file without frontmatter remains valid and its complete contents are the
prompt template.

### 6.2 Description

The builtin `:help` derives an application command description in this order:

1. a non-empty string `description` in YAML frontmatter;
2. otherwise, the first non-empty body line if it is an ATX Markdown heading
   (`#` through `######` followed by whitespace);
3. otherwise, an empty description.

Only the heading text is used. The heading remains part of the prompt body.
Descriptions are displayed on one line; whitespace inside a metadata description
may be collapsed for help output.

Description extraction is informational: a file that cannot provide usable
metadata may still be listed by `:help` with an empty description.

### 6.3 Template substitution

The dispatcher recognizes these substitutions in the prompt body:

```text
$1 ... $9   positional command arguments
$*          all command arguments joined by one space
$$          a literal $
```

Each composed command has its own arguments. A referenced positional argument
that was not supplied is an error. Substitution is single-pass: text introduced
by an argument is not scanned again. Other `$` sequences remain literal.

## 7. Builtin commands

Builtin commands use the reserved `:` namespace and are implemented by the
central dispatcher. Unknown builtin names are errors.

A builtin invocation must contain exactly one command. Builtins do not resolve a
Markdown command file and cannot be overridden by user or project files.

The builtin registry initially contains:

```text
:help     List available builtins and application commands.
:create   Generate an agent prompt for authoring a command.
```

### 7.1 `:help`

Canonical form:

```text
$x :help
```

Aliases:

```text
$x help
$x ?
```

`:help` accepts no arguments and no lead text.

#### Discovery

`:help` is an explicit exception to the normal no-enumeration rule. It
recursively enumerates the selected nearest project command tree and the user
command tree.

A discovered `.md` path is listed only when its relative path maps to a valid
Komut application command name. `help.md` is omitted because `help` is reserved.

Project filesystem safety rules still apply. Project symlink files and symlink
subtrees are not commands and must not be followed by `:help`. User-scope
symlinks remain allowed; help should discover commands reachable through them
while avoiding traversal loops.

If the same command name exists in both scopes, the project command wins and is
listed once. Application commands are sorted lexicographically by name.

#### Output

Help lists builtins separately from application commands. A suitable text form
is:

```text
Builtins:

:help    List available commands. Aliases: help, ?
:create  Create a command with the agent.

Commands:

code/review   Review code for correctness and maintainability.
git/commit    Create a conventional commit.
text/concise
```

If no application commands are found, help still lists builtins and then
explains where to create application commands. It must show absolute paths for:

- the user command directory: `<home>/.agents/commands`;
- the selected or suggested project command directory.

If a nearest project command tree already exists, that tree is the project path.
If no project command tree exists, help uses `<cwd>/.agents/commands`.

### 7.2 `:create`

`:create` is a prompt generator. It does not create directories, write files,
launch an editor, or otherwise mutate the filesystem.

Syntax:

```text
$x :create [--user | --project] [COMMAND] [ -- LEAD ]
```

The default scope is project. `--user` selects the user command tree;
`--project` selects the project command tree explicitly. Supplying both scope
flags is invalid.

`COMMAND` is optional. When present, it must be a valid application command name
and must not be the reserved name `help`.

Examples:

```text
$x :create code/review
$x :create --user text/concise
$x :create git/commit -- Create a Conventional Commits helper.
$x :create -- Write a command that reviews API compatibility.
```

When a command name is present, `:create` computes its absolute target `.md`
path. For project scope it uses the nearest existing project command tree, or
`<cwd>/.agents/commands` when no project tree exists. For user scope it uses
`<home>/.agents/commands`.

The generated prompt instructs the host agent to use its normal file-reading and
file-editing tools to author the command. It should communicate at least:

- the selected scope and target directory or file;
- the Komut command-file format;
- that optional YAML frontmatter may contain `description`;
- that the Markdown body is the prompt template;
- the supported `$1` through `$9`, `$*`, and `$$` substitutions;
- that existing files should be inspected before modification;
- that the agent should ask the user for missing intent instead of inventing
  important command behavior.

If `COMMAND` is omitted, the prompt asks the agent to determine a valid command
name with the user before writing. If lead text is present, it is included as the
user's authoring intent.

The generated prompt is ordinary agent input. Filesystem mutation, approvals,
and interactive editing therefore remain the responsibility of the host agent,
not the Komut dispatcher.

## 8. Prompt composition

Each ordinary application command file is resolved, its metadata removed, and
its body rendered independently. Rendered command bodies are joined in invocation
order with two LF characters (`\n\n`) between components.

If a non-empty lead is present, it is the first component and is separated from
the first rendered body by the same two-LF separator.

Komut otherwise preserves prompt body content. It does not interpret or merge
instructions semantically.

Builtin output is defined by the builtin itself. A prompt-generating builtin such
as `:create` returns one agent prompt directly and does not participate in
application command composition.

## 9. Filesystem safety

Project command trees are repository-controlled input and receive stricter path
checks than user scope.

For project scope, Komut must reject during ordinary resolution:

- a symbolic-link `.agents` path;
- a symbolic-link `commands` path;
- any symbolic-link intermediate directory selected by a slash command name;
- a symbolic-link selected command file;
- a selected path component with an unexpected file type.

The selected project command must remain inside the selected project command
tree. User-scope symbolic links are allowed because they are user-controlled and
useful for dotfile management. The final selected user command must still resolve
to a readable regular file.

Malformed or unsafe ordinary command paths fail closed.

`:create` only computes target paths and returns prompt text. It does not open or
write the target file.

## 10. Dispatcher and host adapters

The central implementation is the `x` dispatcher written in Go. It owns:

- invocation parsing and command-name validation;
- builtin-name parsing and alias normalization;
- scope discovery and command resolution;
- command metadata parsing;
- template substitution and composition;
- builtin commands;
- final output generation.

Host adapters must not implement a second copy of these semantics. They should
only recognize explicit Komut input cheaply, invoke the packaged dispatcher with
the invocation and working directory, inject its result into the same turn, and
surface dispatcher errors.

`$x` remains the canonical cross-host syntax. A host adapter may expose a native
host command as an additional alias when that host has a suitable command
mechanism. The adapter must translate that native invocation to the same raw
Komut invocation and use the same central dispatcher.

Host-specific plugin packages live in separate repository subtrees:

```text
plugins/
├── codex/
├── claude/
└── opencode/
```

Each installed host package must be self-contained.

## 11. Binary distribution

Installed users must not need a Go toolchain. Supported host packages contain
prebuilt dispatcher binaries:

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

`bin/x` is a small POSIX shell launcher. `bin/x.cmd` is a small Windows launcher.
Launchers only select the matching binary. They do not parse Komut syntax or
implement command semantics.

## 12. Repository organization

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

The protocol and behavioral contracts in this specification are authoritative
over directory convenience. Generated distribution copies must not become
independent sources of command semantics.

## 13. Performance requirements

Normal application dispatch is on the interactive prompt path and must remain
small and predictable. It must:

- avoid network I/O;
- avoid command-directory enumeration;
- perform at most one ancestor walk per invocation;
- read only command files selected by the invocation;
- avoid starting subprocesses from the Go dispatcher;
- render all commands before returning one final prompt.

`:help` may enumerate the user and selected project command trees because listing
commands is its purpose. It must not search unrelated directories or more distant
project scopes.

`:create` must not enumerate command trees. It may perform normal project-scope
discovery to compute the target directory.

## 14. Errors

At minimum, Komut must distinguish these error classes internally:

- invalid invocation syntax;
- invalid application command name;
- invalid or unknown builtin name;
- command not found;
- unsafe project path;
- invalid command file or frontmatter;
- unterminated quote;
- missing positional argument;
- invalid builtin arguments;
- unsupported launcher platform.

Errors must not produce a partial rendered prompt.

## 15. Acceptance criteria

Tests must verify at least:

- `$x` boundary, quoting, `+`, and global `--` behavior;
- `$1` through `$9`, `$*`, `$$`, missing arguments, and single-pass rendering;
- slash command names and traversal rejection;
- project-over-user precedence and nearest project scope boundaries;
- project symlink rejection and user-scope symlink support;
- empty, unreadable, non-UTF-8, and non-regular command-file failures;
- valid frontmatter is removed from rendered prompt bodies;
- malformed frontmatter fails ordinary command execution;
- description prefers frontmatter, then the first non-empty ATX body heading,
  then empty text;
- `:help` is canonical and `help` plus `?` normalize to it;
- `help` remains reserved against `help.md`;
- builtin commands reject composition;
- unknown builtins fail cleanly;
- `:help` recursively lists valid command names from user and project scopes;
- project commands win duplicate names in help output;
- help output lists builtins and application commands deterministically;
- project symlink paths are not followed during help enumeration;
- user-scope symlink commands can be discovered without traversal loops;
- no-command help output shows absolute user and project command directories;
- `:create` defaults to project scope and supports `--user` and `--project`;
- `:create` validates an optional command name and rejects `help`;
- `:create` returns an authoring prompt without writing files;
- `:create` carries optional lead text into that prompt;
- ordinary dispatch still performs no command-directory enumeration;
- launchers and host adapters preserve dispatcher semantics across hosts;
- host-native aliases, when provided, produce the same dispatcher result as the
  equivalent `$x` invocation.
