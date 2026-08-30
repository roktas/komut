#!/bin/sh
set -eu

dist=${1:?usage: smoke-dist.sh DIST_DIR}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' 0 HUP INT TERM

home="$tmp/home"
project="$home/project"
commands="$project/.agents/commands"
mkdir -p "$commands"
printf '%s' 'Hello $1' > "$commands/hello.md"

for plugin in codex claude opencode; do
        output=$(
                cd "$project"
                HOME="$home" "$dist/plugins/$plugin/bin/x" '$x hello world'
        )
        if [ "$output" != 'Hello world' ]; then
                echo "smoke: $plugin launcher returned: $output" >&2
                exit 1
        fi
done

payload=$(printf '{"prompt":"$x hello world","cwd":"%s"}' "$project")
codex_hook=$(printf '%s' "$payload" | HOME="$home" "$dist/plugins/codex/bin/x" --hook)
claude_hook=$(printf '%s' "$payload" | HOME="$home" CLAUDE_PLUGIN_ROOT="$dist/plugins/claude" "$dist/plugins/claude/hooks/run.cmd")

if [ "$codex_hook" != "$claude_hook" ]; then
        echo "smoke: Codex and Claude hook transports differ" >&2
        exit 1
fi
printf '%s' "$codex_hook" | grep -F '"hookEventName":"UserPromptSubmit"' >/dev/null
printf '%s' "$codex_hook" | grep -F 'Hello world' >/dev/null

noop=$(printf '{"prompt":"hello","cwd":"%s"}' "$project" | HOME="$home" "$dist/plugins/codex/bin/x" --hook)
if [ -n "$noop" ]; then
        echo "smoke: non-Komut hook input produced output" >&2
        exit 1
fi
