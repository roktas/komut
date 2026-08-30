# AGENTS.md

Read `SPEC.md` before changing behavior, command grammar, resolution, rendering,
security rules, packaging, or host adapters.

## Scope

Keep one source of truth for Komut semantics.

- The Go dispatcher owns parsing, resolution, rendering, and composition.
- Host adapters must stay thin and host-specific.
- Do not duplicate core command semantics in Codex, Claude Code, OpenCode, or
  other adapters.
- Do not add compatibility or migration code for the historical `roktas/.komut`
  prototype unless explicitly requested.
- Prefer deleting obsolete design remnants over preserving them as legacy notes.

## Layout

- `src/` contains implementation source; host adapters are in `src/adapters/`.
- `tests/` contains repository-level product and integration tests.
- `src/dist/` contains the tools that generate and smoke-test the `dist` branch.
- `bin/` contains the public platform launchers.

## Go

Follow the current guidance in
[JetBrains/go-modern-guidelines](https://github.com/JetBrains/go-modern-guidelines)
when writing or reviewing Go code.

- Detect the project's Go version from `src/go.mod`.
- Use language and standard-library features available in that version.
- Prefer modern Go idioms over legacy equivalents.
- Do not raise the minimum Go version only to use a newer idiom.
- Prefer the standard library unless an external dependency has a clear benefit.
- Keep packages small and aligned with actual responsibilities; do not create
  abstraction layers only to mirror the specification headings.

## Dispatcher

Keep the dispatcher deterministic and side-effect-light.

- No network I/O during dispatch.
- No shell evaluation of command text or arguments.
- No command-directory enumeration or fuzzy command lookup.
- Validate command names before constructing filesystem paths.
- Treat project command paths as untrusted repository-controlled input.
- Fail the complete invocation on parse, resolution, read, or render errors; do
  not emit partial prompt output.

## Launchers

`bin/x` and `bin/x.cmd` are platform launchers only.

- They select the packaged Go executable and transfer control to it.
- They must not parse `$x` syntax or implement command semantics.
- Keep them minimal and dependency-free for their target platform.
- Keep `bin/x` POSIX `sh`; do not introduce Bash-only syntax.
- For compact `case` arms, align each pattern with `case` and `esac`. Indent only
  the body of a multi-line arm.

Preferred form:

```sh
case "$os:$arch" in
Darwin:arm64)  target=darwin-arm64 ;;
Darwin:x86_64) target=darwin-amd64 ;;
*)
        ...
        ;;
esac
```

## Host adapters

Each host has its own source subtree under `src/adapters/`.

- Keep each installable host plugin self-contained.
- Use the host's native extension mechanism where possible.
- Do not require a separate user-level hook installer when native plugin
  packaging can carry the integration.
- Preserve `$x` as the canonical cross-host invocation syntax.
- Host-specific aliases are optional and must not change dispatcher semantics.

## Tests

Treat `SPEC.md` acceptance criteria as the minimum behavioral test matrix.

Add regression tests for every parser, filesystem-safety, resolution, rendering,
or composition bug before fixing it when practical.

Test core semantics independently from host adapters, then add focused adapter
integration tests for host-specific transport behavior.

## Generated and binary files

Do not hand-edit generated plugin-package copies or prebuilt dispatcher binaries.
Change canonical source or build inputs and regenerate them.
