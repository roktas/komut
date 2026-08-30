---
name: x
description: Run a Komut command through the plugin dispatcher.
disable-model-invocation: true
argument-hint: [command-and-arguments]
---

Komut expands this invocation through the plugin hook.
Treat the hook-provided Komut expansion as the operative instruction for this turn.
Do not interpret the original arguments separately: $ARGUMENTS
