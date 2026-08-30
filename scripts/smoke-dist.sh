#!/bin/sh
set -eu

dist=${1:?usage: smoke-dist.sh DIST_DIR}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' 0 HUP INT TERM

home="$tmp/home"
project="$home/project"
commands="$project/.agents/commands"
mkdir -p "$commands"
cat > "$commands/hello.md" <<'EOF'
---
description: Friendly hello
---
Hello $1
EOF

for plugin in codex claude opencode; do
        output=$(
                cd "$project"
                HOME="$home" "$dist/plugins/$plugin/bin/x" '$x hello world'
        )
        if [ "$output" != 'Hello world' ]; then
                echo "smoke: $plugin launcher returned: $output" >&2
                exit 1
        fi

        help=$(
                cd "$project"
                HOME="$home" "$dist/plugins/$plugin/bin/x" '$x'
        )
        printf '%s' "$help" | grep -F ':new' >/dev/null
        printf '%s' "$help" | grep -F ':version' >/dev/null
        printf '%s' "$help" | grep -F 'hello' >/dev/null
        printf '%s' "$help" | grep -F 'Friendly hello' >/dev/null

        version=$(
                cd "$project"
                HOME="$home" "$dist/plugins/$plugin/bin/x" '$x :version'
        )
        if [ "$version" != 'Komut 0.3.0' ]; then
                echo "smoke: $plugin version returned: $version" >&2
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

claude_native_payload=$(printf '{"hook_event_name":"UserPromptExpansion","expansion_type":"slash_command","command_name":"komut:x","command_args":"hello world","command_source":"plugin","cwd":"%s"}' "$project")
claude_native=$(printf '%s' "$claude_native_payload" | HOME="$home" CLAUDE_PLUGIN_ROOT="$dist/plugins/claude" "$dist/plugins/claude/hooks/run.cmd")
printf '%s' "$claude_native" | grep -F '"hookEventName":"UserPromptExpansion"' >/dev/null
printf '%s' "$claude_native" | grep -F 'Hello world' >/dev/null

noop=$(printf '{"prompt":"hello","cwd":"%s"}' "$project" | HOME="$home" "$dist/plugins/codex/bin/x" --hook)
if [ -n "$noop" ]; then
        echo "smoke: non-Komut hook input produced output" >&2
        exit 1
fi
