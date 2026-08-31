# Komut Specification

Version: 0.3.1

## 1. Purpose

Komut provides a small, namespaced prompt-command protocol for AI agents.

The canonical cross-host syntax is:

```text
$x [COMMAND [ARGS...] [ + COMMAND [ARGS...] ... ] [ -- LEAD ]]
```

Application commands resolve to Markdown prompt files. Builtin control commands
live under the reserved `:` namespace. Komut owns parsing, scope resolution,
metadata handling, template rendering, composition, builtins, and final prompt
generation. Host plugins are thin adapters around the same dispatcher.

## 2. Design principles

- `$x` is the stable public namespace across hosts.
- `$x` with no command is useful and opens help.
- Application commands and builtin control commands use separate namespaces.
- Command files are Markdown prompt data with optional metadata, not programs.
- Project commands may override user commands without changing user files.
- Builtins may generate prompts but should not perform agent work themselves.
- Host-native commands are aliases only; they must use the central dispatcher.
- Normal installation must not require a language runtime or network download.

## 3. Invocation grammar

The logical grammar is:

```ebnf
invocation   = "$x", [ ws, command, { ws, "+", ws, command },
               [ ws, "--", [ ws, lead ] ] ] ;
command      = builtin | application ;
builtin      = builtin_name, { ws, argument } ;
application  = name, { ws, argument } ;
builtin_name = ":", builtin_segment ;
name         = segment, { "/", segment } ;
argument     = bare | single_quoted | double_quoted ;
lead         = <remaining input> ;
```

Leading whitespace before `$x` is allowed. `$xfoo` is not an invocation. After
`$x`, either end-of-input or whitespace is required.

An empty invocation is an exact alias for help:

```text
$x  -> $x :help
```

Two more exact aliases are defined:

```text
$x help  -> $x :help
$x ?     -> $x :help
```

`:help` is the canonical name. Application command name `help` is therefore
reserved and `help.md` cannot override the builtin.

Builtin names begin with `:`. A builtin segment starts with a lowercase ASCII
letter and then contains only lowercase ASCII letters, digits, and `-`.
Application command names never begin with `:`.

### 3.1 Composition

An unquoted standalone `+` separates application commands:

```text
$x code/review src/foo.go + concise + lang/turkish
```

Commands render independently and are joined in declared order. They are not a
pipeline. A builtin must be the only command in an invocation and cannot be
composed with another command.

`+` is ordinary argument text in values such as `C++`, `a+b`, or `"a + b"`.
Failure of any composed command fails the entire invocation with no partial
prompt.

### 3.2 Lead text

The first unquoted standalone `--` ends Komut parsing globally. Everything after
it is opaque lead text.

```text
$x code/review src/foo.go + concise -- Keep the public API stable.
```

A non-empty lead is placed before rendered application command content. A builtin
may define its own lead behavior. `:new` uses lead text as the optional one-line
command description; `:help` and `:version` reject lead text.

### 3.3 Quoting

Whitespace separates arguments. Single quotes preserve their contents literally.
Double quotes preserve whitespace and unescape `\"` and `\\`. Komut does not
perform shell expansion, environment expansion, globbing, or command
substitution. A quoted argument must end at a token boundary.

## 4. Application command names

A command name is one or more slash-separated segments such as:

```text
review
code/review
git/commit
foo/bar-baz
```

The complete name is at most 64 ASCII characters including slashes. Each segment:

- contains only lowercase ASCII letters, digits, and `-`;
- starts and ends with a letter or digit;
- is non-empty;
- does not contain `--`.

Names must not start or end with `/`, contain `//`, or contain `.`, `\\`, `_`,
`:`, or whitespace. A name maps directly to a Markdown path; `foo/bar/baz` maps
to `commands/foo/bar/baz.md`.

Normal application dispatch does not enumerate command directories, try
alternative names, or perform fuzzy matching.

## 5. Scope and resolution

User commands live under:

```text
~/.agents/commands/
```

Project commands live under:

```text
<project>/.agents/commands/
```

Komut walks upward from the current working directory and selects the nearest
existing `.agents/commands` directory. The home command tree is resolved
separately and is never treated as project scope.

For each application command, resolution order is:

1. nearest project command tree;
2. user command tree;
3. not found.

The nearest project command tree is a scope boundary. A miss there falls directly
to user scope; Komut does not search more distant project trees. A project command
shadows a same-name user command. One invocation performs one project-scope walk.

When a builtin needs a project target and no project command tree exists, it may
use `<cwd>/.agents/commands` as the target that would establish project scope
there. This target is unavailable when `cwd` is the user home because it would be
the user command tree. Existing project path components must also pass the
project-scope safety rules before a builtin may present that target.

## 6. Command files

A command file must be a readable, non-empty, UTF-8 regular Markdown file. It may
start with YAML frontmatter delimited by `---` lines:

```md
---
description: Review code for correctness and maintainability.
---

Review $1.
```

Frontmatter is metadata and is never sent to the agent. Komut currently defines
one optional metadata field:

```yaml
description: <string>
```

Unknown fields are allowed and ignored. Malformed frontmatter makes the file
invalid for ordinary command execution. A file without frontmatter remains valid
and its complete contents are the prompt template.

### 6.1 Description

`:help` derives a description in this order:

1. non-empty string `description` from frontmatter;
2. otherwise, the first non-empty body line when it is an ATX Markdown heading
   (`#` through `######` followed by whitespace);
3. otherwise, empty.

Metadata descriptions may have whitespace collapsed for one-line display.
Description extraction is informational: help may still list a command whose
metadata cannot be parsed, with an empty description.

### 6.2 Template substitution

The body recognizes:

```text
$1 ... $9   positional command arguments
$*          all command arguments joined by one space
$$          a literal $
```

A referenced missing positional argument is an error. Substitution is single-pass;
argument text is not rescanned. Unknown `$` sequences remain literal.

## 7. Builtin commands

Builtin commands are central dispatcher operations. Unknown builtin names are
errors. Builtins do not resolve Markdown files and cannot be overridden.

The initial registry is:

```text
:help     List builtins and available application commands.
:new      Generate an agent prompt for authoring a command.
:version  Show the installed Komut version.
```

### 7.1 `:help`

Canonical form:

```text
$x :help
```

Aliases:

```text
$x
$x help
$x ?
```

`:help` accepts no arguments and no lead.

Help recursively enumerates only the user command tree and the selected nearest
project command tree. This is the explicit exception to the normal no-enumeration
rule. Valid `.md` paths become command names. Project commands win duplicate
names. Application commands are sorted lexicographically.

Project symlink files and symlink subtrees are not followed. User-scope symlinks
remain allowed and help must avoid traversal loops.

Help always lists builtins separately. A suitable form is:

```text
Builtins:

:help     List available commands. Aliases: help, ?
:new      Create a command with the agent.
:version  Show the installed Komut version.

Commands:

code/review   Review code for correctness and maintainability.
git/commit    Create a conventional commit.
```

If there are no application commands, help still lists builtins and explains
where to create commands. It shows the absolute user command directory. For
project scope, it shows the selected project directory when one exists, otherwise
`<cwd>/.agents/commands`. When `cwd` is the user home, help must not present the
user command tree as a project directory; it instead explains that project
commands require a project working directory.

### 7.2 `:new`

`:new` is a prompt generator. It never creates directories, writes files, or
launches an editor.

```text
$x :new COMMAND [--project] [ -- DESCRIPTION ]
```

`COMMAND` is required, must be the first argument, and must be a valid
non-reserved application command name. User scope is the default everywhere.
`--project` may follow the command name to select project scope explicitly.
`--user` is not supported. Other positional arguments are invalid; free-form
description text belongs after the global `--` marker.

Project scope is unavailable when `cwd` is the user home because its prospective
project directory would be the user command tree.

Examples:

```text
$x :new code/review
$x :new code/review --project
$x :new git/commit -- Create a Conventional Commits helper.
$x :new review --project -- Review code for correctness and compatibility.
```

The generated prompt always includes the absolute target Markdown file path. The
path is formed from the command name and always ends in `.md`; an authoring agent
must not create an extensionless command file.

Description text after `--`, when present and non-empty, is the command's one-line
description. When description is omitted, the generated prompt instructs the host
agent to ask the user for it before writing. The generated prompt always instructs
the agent to ask the user for the Markdown command body before writing, regardless
of whether a description was supplied.

New commands authored through `:new` use YAML frontmatter containing the
`description`, followed by the Markdown body as the prompt template. The prompt
also communicates `$1` through `$9`, `$*`, and `$$` substitutions, requires parent
directories to be created when needed, and requires an existing target to be
inspected before changing it. The host agent must not invent a missing
description or body.

If no project command tree exists yet, explicit project scope may target
`<cwd>/.agents/commands`. Before generating that project target, Komut checks any
existing `.agents` and `commands` path components. A symlink or unexpected path
type is unsafe and fails closed.

Filesystem mutation and approvals remain the host agent's responsibility.

### 7.3 `:version`

Canonical form:

```text
$x :version
```

`:version` accepts no arguments and no lead. It returns the installed product
version in this form:

```text
Komut 0.3.1
```

The value comes from the built dispatcher, not by reading repository files at
runtime.

## 8. Prompt composition

Each application command is resolved, metadata is removed, and its body is
rendered independently. Components are joined in invocation order with two LF
characters (`\n\n`). Non-empty lead is the first component.

Komut otherwise preserves prompt body content and does not merge instructions
semantically. Prompt-generating builtins such as `:new` return one prompt
directly and do not participate in application composition.

## 9. Filesystem safety

For project scope, ordinary resolution rejects:

- symlink `.agents`;
- symlink `commands`;
- symlink intermediate directories in a slash command path;
- symlink selected command files;
- unexpected selected path types.

The selected project command must remain in the selected project command tree.
User-scope symlinks are allowed, but the final selected target must be a readable
regular file. Unsafe or malformed paths fail closed.

`:new` computes target paths only and does not open or write them. When explicit
project scope would establish a new project command tree, it applies the project
path safety rules to existing path components before generating the authoring
prompt.

## 10. Host adapters

Host adapters must not reimplement Komut grammar or resolution. `$x` is canonical
across hosts. A host may expose a native alias when it can translate that alias to
the same raw `$x` invocation and call the same dispatcher.

An adapter must run the dispatcher with the working directory of the current
invocation or session. Plugin installation, load, or package directories must not
be used as command-resolution scope.

Current native aliases are:

```text
Claude Code          /komut:x [ARGS...]
Google Antigravity   /x [ARGS...]
OpenCode V2          /x [ARGS...]
```

For all native aliases, the text after the native command is passed as the text
after `$x`. Thus an argument-free native invocation maps to `$x` and therefore
to `:help`. Codex uses the canonical `$x` syntax.

Antigravity uses its plugin skill transport because its hook events do not expose
the active user prompt. The skill passes the invocation as data to the packaged
launcher from the current session working directory and treats successful
dispatcher output as the operative prompt. It must not infer the prompt from a
persistent transcript.

Each installed host package is self-contained and includes the same dispatcher
semantics.

## 11. Binary distribution

Installed users do not need Go. Host packages contain:

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

Launchers only select the matching binary. They do not parse Komut syntax.

## 12. Performance

Normal application dispatch must:

- avoid network I/O;
- avoid command-directory enumeration;
- perform at most one ancestor walk;
- read only selected command files;
- avoid subprocesses from the Go dispatcher;
- render all commands before returning one final prompt.

`:help` may enumerate the two selected command trees because listing is its
purpose. It must not search unrelated directories or more distant project scopes.

## 13. Errors

Komut must distinguish at least:

- invalid invocation syntax;
- invalid or unknown command name;
- command not found;
- unsafe project path;
- invalid command file or frontmatter;
- unterminated quote;
- missing positional argument;
- unsupported launcher platform.

Errors never produce a partial rendered prompt.

## 14. Acceptance criteria

Tests must verify at least:

- `$x`, `$x help`, `$x ?`, and `$x :help` resolve to the same help builtin;
- `$xfoo` remains invalid;
- builtin names use the `:` namespace and cannot be composed;
- `:new` requires the command name first, defaults to user scope, supports only
  explicit `--project` scope selection, always targets a `.md` file, accepts a
  description after `--`, and asks for missing description plus the command body;
- explicit project `:new` rejects unsafe prospective path components and rejects
  project scope at the user home;
- `:version` reports the built product version and rejects args/lead;
- quoting, composition, global `--`, substitutions, and missing arguments;
- project-over-user precedence and nearest-project scope boundaries;
- project symlink rejection and user-symlink support;
- YAML metadata stripping and description fallback behavior;
- help discovery, sorting, duplicate precedence, and no-command path guidance,
  including the user-home case;
- Claude `/komut:x`, Antigravity `/x`, and OpenCode `/x` native aliases use
  central dispatcher semantics, including argument-free help;
- host adapters pass the current invocation/session working directory to the
  dispatcher rather than a plugin installation or load directory;
- launchers and generated host packages preserve the same behavior.
