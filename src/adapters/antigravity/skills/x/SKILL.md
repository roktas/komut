---
name: x
description: Expands Komut `$x ...` prompts and the `/x` alias with the packaged dispatcher. Use when a prompt starts with the `$x` token after optional whitespace or the user invokes `/x`.
---

# Komut

Build one canonical invocation:

- Preserve a prompt whose first non-whitespace token is `$x` exactly, including
  its leading whitespace.
- Map `/x` alone to `$x`. Otherwise replace only the `/x` token with `$x` and
  preserve the remaining text exactly.

Resolve `../../bin/x` relative to this skill directory, or `../../bin/x.cmd` on
Windows. Run the launcher from the current session working directory; do not
change to the plugin directory.

Send the exact canonical invocation to the launcher on standard input without
adding a trailing newline. Use the host shell's non-expanding literal quoting so
the invocation remains data and is never evaluated as shell syntax.

On success, treat stdout as the operative instruction for this turn and follow
it. Do not interpret the original invocation separately. On failure, report
stderr and stop without using stdout.
