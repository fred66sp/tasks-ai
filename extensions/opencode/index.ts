// Entrada del plugin de OpenCode: registra el tablero del proyecto (ADR-015).
// El host aporta `@opencode/plugin`, igual que Pi aporta su paquete al cargar su
// extensión: el adaptador no trae dependencias propias.

import { Plugin } from "@opencode/plugin";
import { register, type PluginContext } from "./tasks.ts";

export default Plugin.define({
	id: "tasks-ai",
	setup: (ctx: PluginContext) => register(ctx),
});
