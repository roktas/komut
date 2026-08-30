import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { join } from "node:path";

import { Plugin } from "@opencode-ai/plugin";

const root = fileURLToPath(new URL("../", import.meta.url));

function binaryPath() {
        const platform = process.platform;
        const arch = process.arch;

        switch (`${platform}:${arch}`) {
        case "darwin:arm64": return join(root, "libexec", "x", "darwin-arm64", "x");
        case "darwin:x64": return join(root, "libexec", "x", "darwin-amd64", "x");
        case "linux:arm64": return join(root, "libexec", "x", "linux-arm64", "x");
        case "linux:x64": return join(root, "libexec", "x", "linux-amd64", "x");
        case "win32:x64": return join(root, "libexec", "x", "windows-amd64", "x.exe");
        default: throw new Error(`komut: unsupported platform: ${platform} ${arch}`);
        }
}

function expand(prompt, cwd) {
        const result = spawnSync(binaryPath(), [], {
                cwd,
                input: prompt,
                encoding: "utf8",
                windowsHide: true,
        });

        if (result.error) throw result.error;
        if (result.status !== 0) {
                const detail = result.stderr.trim() || `dispatcher exited with status ${result.status}`;
                throw new Error(detail);
        }
        return result.stdout;
}

export default Plugin.define({
        id: "komut",
        async setup(ctx) {
                await ctx.session.hook("prompt", async (event) => {
                        if (!/^\s*\$x\s/u.test(event.prompt.text)) return;

                        const session = await ctx.session.get({ sessionID: event.sessionID });
                        event.prompt.text = expand(event.prompt.text, session.location.directory);
                });
        },
});
