import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { join } from "node:path";

import { Plugin } from "@opencode-ai/plugin";

const root = fileURLToPath(new URL("../", import.meta.url));

function launcherPath() {
        return join(root, "bin", process.platform === "win32" ? "x.cmd" : "x");
}

function expand(prompt, cwd) {
        const result = spawnSync(launcherPath(), [], {
                cwd,
                input: prompt,
                encoding: "utf8",
                windowsHide: true,
                shell: process.platform === "win32",
        });

        if (result.error) throw result.error;
        if (result.status !== 0) {
                const detail = result.stderr.trim() || `dispatcher exited with status ${result.status}`;
                throw new Error(detail);
        }
        return result.stdout;
}

function invocation(args) {
        const text = args.trim();
        return text === "" ? "$x" : `$x ${text}`;
}

export default Plugin.define({
        id: "komut",
        async setup(ctx) {
                await ctx.command.transform((draft) => {
                        draft.add({
                                name: "x",
                                description: "Run a Komut command",
                                execute: async ({ sessionID, prompt, delivery }) => {
                                        await ctx.session.prompt({
                                                ...prompt,
                                                sessionID,
                                                text: invocation(prompt.text),
                                                delivery,
                                        });
                                },
                        });
                });

                await ctx.session.hook("prompt", async (event) => {
                        if (!/^\s*\$x(?:\s|$)/u.test(event.prompt.text)) return;

                        const session = await ctx.session.get({ sessionID: event.sessionID });
                        event.prompt.text = expand(event.prompt.text, session.directory);
                });
        },
});
