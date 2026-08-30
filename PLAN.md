# Plan

## Constraints

- Keep `SPEC.md` as the source of truth for dispatcher behavior.
- Preserve the current directory layout. Add host files only under the existing
  `src/adapters/`, `tests/`, and distribution surfaces.
- Keep adapters thin. Parsing, resolution, rendering, and composition remain in
  the Go dispatcher.
- Prefer deletion and standard-library code over new abstractions or
  dependencies.

## Review baseline

The review covered the dispatcher, launchers, all host adapters, repository and
package tests, distribution scripts, manifests, CI, `README.md`, and `SPEC.md`.
The initial test, vet, format, syntax, build, and distribution smoke checks
passed.

The review found:

1. The shared hook transport silently ignored malformed JSON and a missing
   working directory. The host could then process the original `$x` text instead
   of failing the invocation.
2. Scope detection compared home paths lexically. A symlink alias for the user
   home could therefore classify the user command tree as project scope and apply
   the wrong symlink policy.
3. The OpenCode native alias trimmed its argument string before forwarding it.
   This changed trailing whitespace in opaque lead text instead of passing the
   text after `/x` unchanged.
4. Tests retained a redundant parser boundary case and a small amount of local
   cleanup residue. Two Go expressions could also use the direct Go 1.26 standard
   library forms identified by the Modern Go Guidelines.

## Phase 1: remediation

- [x] Make recognized hook invocations fail on malformed transport input or a
  missing working directory; keep unrelated prompts and events as no-ops.
- [x] Compare existing home paths by filesystem identity, with the current
  lexical comparison as the fast path and fallback for prospective paths.
- [x] Preserve non-empty OpenCode native command arguments exactly.
- [x] Remove the confirmed test residue and apply only the relevant modern Go
  simplifications.
- [x] Run the full test, vet, format, syntax, distribution build, and smoke
  matrix.
- [x] Commit and push the remediation as one reviewable change (`e50e769`).

## Phase 2: Google Antigravity

Antigravity 2.0 and Antigravity CLI (`agy`) discover the same plugin shape:
`plugin.json` plus optional skills. Installed skills are available to the agent
by description and as native slash commands. The current hook contract does not
provide the active user prompt; reading the persistent transcript would add an
undocumented transport dependency and still leave multi-workspace scope
ambiguous.

Plan:

- [x] Add one self-contained `src/adapters/antigravity` plugin with an `x` skill
  for canonical `$x` invocations and the native `/x` alias.
- [x] Have the skill pass the raw invocation to the existing packaged launcher
  from the current session working directory, then treat dispatcher stdout as
  the operative prompt.
- [x] Include the adapter in the existing distribution build and smoke loop; do
  not add a prompt hook, transcript parser, MCP server, dependency, or new core
  abstraction.
- [x] Add focused manifest, skill-contract, and package tests.
- [x] Update `SPEC.md` and `README.md` with the supported alias, installation,
  and the skill-mediated transport boundary.
- [x] Run the full verification matrix and review the result for leftovers.
