# Komut Specification

Version: 0.2.0

## 1. Purpose

Komut provides a small, namespaced prompt-command protocol for AI agents.

A user writes:

```text
$x COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]
```

Komut resolves Markdown command files, substitutes arguments, composes the
rendered commands, and returns one final prompt for the current agent turn.
Komut also provides the reserved builtin command `$x help`.

Komut owns the command grammar, resolution rules, rendering rules, builtin
commands, and the `x` dispatcher. It does not ship application commands. Users
and projects provide their own command files.

Host integrations for Codex, Claude Code, OpenCode, and other agents are thin
adapters around the same dispatcher semantics.

## 2. Design principles

- `$x` is the stable public namespace across hosts.
- One ordinary invocation produces one final prompt and one agent turn.
- Command files are Markdown prompt data with optional metadata, not programs.
- Parsing, resolution, metadata handling, rendering, and builtins belong to the
  central dispatcher, not host adapters.
- Project commands may override user commands without changing user files.
- Host-specific packaging must not change command semantics.
- Normal installation must not require Go, Python, Ruby, Node.js, or a network
  download at runtime.

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

An unquoted standalone `+` token separates commands:

```text
$x code/review src/foo.go + concise + lang/turkish
```

Commands are rendered independently and composed in declared order. They are
not a pipeline.

`+` is ordinary text inside values such as `C++`, `a+b`, or `"a + b"`.
If any composed command fails, the whole invocation fails. Komut must not return
a partial prompt.

### 3.2 Lead text

The first unquoted standalone `--` token ends Komut parsing globally:

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

Everything after `--` is opaque lead text. It is not parsed for `+`, `--`,
quotes, or template substitutions. A non-empty lead is placed before rendered
command content.

### 3.3 Arguments and quoting

Whitespace separates arguments unless text is quoted. Single quotes preserve
their contents literally. Double quotes preserve whitespace and allow `\"` and
`\\` escapes.

Komut does not implement shell expansion, environment expansion, globbing,
command substitution, or other shell features. Quoted `"+"` and `"--"` are
ordinary arguments.

## 4. Command names and paths

A command name is one or more slash-separated segments:

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
`\\`, `_`, and whitespace are not allowed.

A name maps directly to a Markdown path. `foo/bar/baz` maps to:

```text
commands/foo/bar/baz.md
```

Except for the builtin `help`, Komut does not enumerate command directories,
try alternative names, or perform fuzzy matching during normal dispatch.

The name `help` is reserved. A `help.md` file cannot override the builtin and is
not listed as an application command.

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

For each ordinary command, resolution order is:

1. the command in the nearest project command tree, if present;
2. the command in the user command tree;
3. not found.

The nearest project command tree is a scope boundary. If it does not contain the
requested command, Komut falls back directly to user scope and does not search
more distant project command trees.

A project command shadows a user command with the same name. All commands in one
invocation use the same project-scope discovery result. The ancestor walk should
run once per invocation.

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

The builtin `help` derives a command description in this order:

1. a non-empty string `description` in YAML frontmatter;
2. otherwise, the first non-empty body line if it is an ATX Markdown heading
   (`#` through `######` followed by whitespace);
3. otherwise, an empty description.

Only the heading text is used. The heading remains part of the prompt body.
Descriptions are displayed on one line; whitespace inside a metadata description
may be collapsed for help output.

Description extraction is informational: a file that cannot provide usable
metadata may still be listed by `help` with an empty description.

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

## 7. Builtin help

`$x help` is a reserved builtin command. It lists application commands available
from the current user and project scopes.

The builtin form is exactly one command with no arguments, no composition, and
no lead text:

```text
$x help
```

Other uses of `help`, such as `$x help foo` or `$x help + concise`, are invalid.

### 7.1 Discovery

`help` is the explicit exception to the normal no-enumeration rule. It recursively
enumerates the selected nearest project command tree and the user command tree.

A discovered `.md` path is listed only when its relative path maps to a valid
Komut command name. `help.md` is omitted because `help` is reserved.

Project filesystem safety rules still apply. Project symlink files and symlink
subtrees are not commands and must not be followed by `help`. User-scope
symlinks remain allowed; help should discover commands reachable through them
while avoiding traversal loops.

If the same command name exists in both scopes, the project command wins and is
listed once. Help output is sorted lexicographically by command name.

### 7.2 Output

For each selected command, help prints the command name and, when available, its
description. The output is deterministic. A suitable text form is:

```text
code/review   Review code for correctness and maintainability.
git/commit    Create a conventional commit.
text/concise
```

If no application commands are found, help must explain where to create them and
show absolute paths for both scopes. It must show:

- the user command directory: `<home>/.agents/commands`;
- the project command directory.

If a nearest project command tree already exists, that tree is the project path.
If no project command tree exists, help uses `<cwd>/.agents/commands` as the
suggested path that would establish project scope at the current directory.

A no-command response should be equivalent to:

```text
No Komut commands found.

Create user-wide commands in:
  /absolute/home/.agents/commands

Create project commands in:
  /absolute/current/project/.agents/commands
```

## 8. Prompt composition

Each ordinary command file is resolved, its metadata removed, and its body
rendered independently. Rendered command bodies are joined in invocation order
with two LF characters (`\n\n`) between components.

If a non-empty lead is present, it is the first component and is separated from
the first rendered body by the same two-LF separator.

Komut otherwise preserves prompt body content. It does not interpret or merge
instructions semantically.

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

## 10. Dispatcher and host adapters

The central implementation is the `x` dispatcher written in Go. It owns:

- invocation parsing and command-name validation;
- scope discovery and command resolution;
- command metadata parsing;
- template substitution and composition;
- builtin commands, including `help`;
- final output generation.

Host adapters must not implement a second copy of these semantics. They should
only recognize explicit `$x` input cheaply, invoke the packaged dispatcher with
the prompt and working directory, inject its result into the same turn, and
surface dispatcher errors.

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

Normal dispatch is on the interactive prompt path and must remain small and
predictable. It must:

- avoid network I/O;
- avoid command-directory enumeration;
- perform at most one ancestor walk per invocation;
- read only command files selected by the invocation;
- avoid starting subprocesses from the Go dispatcher;
- render all commands before returning one final prompt.

`$x help` may enumerate the user and selected project command trees because
listing commands is its purpose. It must not search unrelated directories or
more distant project scopes.

## 14. Errors

At minimum, Komut must distinguish these error classes internally:

- invalid invocation syntax;
- invalid command name;
- command not found;
- unsafe project path;
- invalid command file or frontmatter;
- unterminated quote;
- missing positional argument;
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
- `help` is reserved and cannot be overridden by `help.md`;
- `$x help` recursively lists valid command names from user and project scopes;
- project commands win duplicate names in help output;
- help output is sorted and deterministic;
- project symlink paths are not followed during help enumeration;
- user-scope symlink commands can be discovered without traversal loops;
- no-command help output shows absolute user and project command directories;
- ordinary dispatch still performs no command-directory enumeration;
- launchers and host adapters preserve dispatcher semantics across hosts.
