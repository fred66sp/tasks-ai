// Extensión del agente Pi: el tablero del proyecto como herramientas, sobre el CLI
// del producto (ADR-012). Aquí solo entran las de lectura; las de escritura llegan
// después y ejecutan los mismos comandos.

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { deriveBoard, renderBoard, renderDetail, type Detail, type Task } from "./board.ts";
import { runCli } from "./cli.ts";

export default function tasksExtension(pi: ExtensionAPI) {
	pi.registerTool({
		name: "tasks_board",
		label: "Board",
		description:
			"El tablero del proyecto: cuánto hay abierto, qué está en curso, qué se puede empezar y el árbol de tareas. No trae descripciones ni comentarios; para eso está tasks_show.",
		promptSnippet: "Read the project task board",
		parameters: Type.Object({}),
		async execute(_toolCallId, _params, signal, _onUpdate, ctx) {
			const outcome = await runCli(["list"], ctx.cwd, signal);
			if (!outcome.ok) throw new Error(outcome.message);

			const board = deriveBoard(outcome.data as Task[]);

			return {
				content: [{ type: "text", text: renderBoard(board) }],
				details: board,
			};
		},
	});

	pi.registerTool({
		name: "tasks_show",
		label: "Task",
		description:
			"Una tarea del proyecto con su descripción, sus hijas, sus bloqueos y sus últimos comentarios. Es la puerta al detalle: úsala cuando el tablero no baste.",
		promptSnippet: "Read one task from the project board",
		parameters: Type.Object({
			ref: Type.String({ description: "la referencia visible de la tarea, por ejemplo tk-18" }),
		}),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			const outcome = await runCli(["show", params.ref], ctx.cwd, signal);
			if (!outcome.ok) throw new Error(outcome.message);

			return {
				content: [{ type: "text", text: renderDetail(outcome.data as Detail) }],
				details: outcome.data,
			};
		},
	});
}
