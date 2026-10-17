// Extensión del agente Pi: el tablero del proyecto como herramientas, sobre el CLI
// del producto (ADR-012). Las reglas y los rechazos siguen siendo del producto.

import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { deriveBoard, renderBoard, renderDetail, renderSummary, type Board, type Detail, type Task } from "./board.ts";
import { findProjectRoot, runCli, type Outcome } from "./cli.ts";

const taskFields = {
	title: Type.Optional(Type.String({ pattern: "\\S", description: "el título de la tarea" })),
	description: Type.Optional(Type.String()),
	notes: Type.Optional(Type.String()),
	assignee: Type.Optional(Type.String()),
	priority: Type.Optional(Type.Integer({ minimum: 0, maximum: 3 })),
};

const reference = Type.String({ description: "la referencia visible de la tarea, por ejemplo tk-18" });

export default function tasksExtension(pi: ExtensionAPI, executeCli = runCli) {
	let previousTools: string[] | undefined;
	let cached: { cwd: string; at: number; pending: Promise<Outcome> } | undefined;

	let statusUI: ExtensionContext["ui"] | undefined;

	pi.on("session_start", async (_event, ctx) => {
		// Sin tablero, el todo del arnés es el único sitio donde anotar: no se retira.
		if (previousTools === undefined && ctx.cwd !== undefined && findProjectRoot(ctx.cwd) !== null) {
			previousTools = [...pi.getActiveTools()];
			pi.setActiveTools(previousTools.filter((name) => name !== "todo"));
		}
		if (ctx.mode === "tui" && ctx.hasUI) await refreshUI(ctx);
	});
	pi.on("session_shutdown", () => {
		statusUI?.setStatus("tasks_board", undefined);
		statusUI = undefined;
		if (previousTools !== undefined) pi.setActiveTools(previousTools);
		previousTools = undefined;
		cached = undefined;
	});

	async function readBoard(cwd: string, signal?: AbortSignal) {
		if (!cached || cached.cwd !== cwd || Date.now() - cached.at >= 1000) {
			cached = { cwd, at: Date.now(), pending: executeCli(["list"], cwd, signal) };
		}
		const entry = cached;
		try {
			const outcome = await entry.pending;
			if (!outcome.ok) throw new Error(outcome.message);
			return deriveBoard(outcome.data as Task[]);
		} catch (error) {
			if (cached === entry) cached = undefined;
			throw error;
		}
	}

	function setBoardStatus(ctx: ExtensionContext, board?: Board, message = "no hay tablero instalado en este proyecto") {
		if (ctx.mode !== "tui" || !ctx.hasUI) return;
		const current = board?.inProgress[0];
		const text = current ? `${current.ref} · ${current.title.replace(/\s+/g, " ").trim()}`
			: board ? "sin tarea en curso" : message;
		statusUI = ctx.ui;
		ctx.ui.setStatus("tasks_board", `tablero: ${text.length > 100 ? `${text.slice(0, 99)}…` : text}`);
	}

	async function refreshUI(ctx: ExtensionContext) {
		if (findProjectRoot(ctx.cwd) === null) {
			setBoardStatus(ctx);
			return "Tablero: no hay tablero instalado en este proyecto.";
		}
		try {
			const board = await readBoard(ctx.cwd, ctx.signal);
			setBoardStatus(ctx, board);
			return renderBoard(board);
		} catch (error) {
			setBoardStatus(ctx, undefined, "no disponible");
			return `Tablero no disponible: ${error instanceof Error ? error.message : String(error)}`;
		}
	}

	pi.registerCommand("board", {
		description: "Muestra el tablero del proyecto en el TUI",
		async handler(_args, ctx) {
			if (ctx.mode !== "tui" || !ctx.hasUI) return;
			ctx.ui.notify(await refreshUI(ctx), "info");
		},
	});

	pi.on("before_agent_start", async (event, ctx) => {
		let summary = "Tablero: no hay tablero instalado en este proyecto; la herramienta todo sigue disponible para anotar el trabajo.";
		if (findProjectRoot(ctx.cwd) !== null) {
			const board = await readBoard(ctx.cwd, ctx.signal);
			summary = renderSummary(board);
			setBoardStatus(ctx, board);
		} else {
			setBoardStatus(ctx);
		}
		event.systemPromptOptions.sections.tasks_board = summary;
	});

	/** Una línea para el modelo; la respuesta completa del producto queda en details. */
	async function writeTask(args: string[], action: string, ctx: ExtensionContext, signal?: AbortSignal) {
		const outcome = await executeCli(args, ctx.cwd, signal);
		if (!outcome.ok) throw new Error(outcome.message);
		cached = undefined;
		if (ctx.mode === "tui" && ctx.hasUI) await refreshUI(ctx);

		const task = outcome.data as Task;
		const title = task.title.replace(/\s+/g, " ").trim();
		return {
			content: [{ type: "text" as const, text: `${task.ref} · ${action} [${task.status}] ${title.length > 60 ? `${title.slice(0, 59)}…` : title}` }],
			details: outcome.data,
		};
	}

	pi.registerTool({
		name: "tasks_board",
		label: "Board",
		description:
			"El tablero del proyecto: cuánto hay abierto, qué está en curso, qué se puede empezar y el árbol de tareas. No trae descripciones ni comentarios; para eso está tasks_show.",
		promptSnippet: "Read the project task board",
		parameters: Type.Object({}),
		async execute(_toolCallId, _params, signal, _onUpdate, ctx) {
			const board = await readBoard(ctx.cwd, signal);
			setBoardStatus(ctx, board);

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
			const outcome = await executeCli(["show", params.ref], ctx.cwd, signal);
			if (!outcome.ok) throw new Error(outcome.message);

			return {
				content: [{ type: "text", text: renderDetail(outcome.data as Detail) }],
				details: outcome.data,
			};
		},
	});

	pi.registerTool({
		name: "tasks_create",
		label: "Crear tarea",
		description: "Crea una tarea; usa parent para crear una subtarea de una tarea existente.",
		parameters: Type.Object({
			...taskFields,
			title: Type.String({ pattern: "\\S", description: "el título de la tarea" }),
			parent: Type.Optional(reference),
		}),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			const args = ["create"];
			for (const field of ["title", "description", "notes", "assignee", "priority", "parent"] as const) {
				if (params[field] !== undefined) args.push(`--${field}`, String(params[field]));
			}
			return writeTask(args, "creada", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_update",
		label: "Actualizar tarea",
		description: "Cambia solo los campos indicados de una tarea. Usa status solo; tags sustituye todas sus etiquetas.",
		parameters: Type.Object({
			ref: reference,
			...taskFields,
			status: Type.Optional(Type.Union([Type.Literal("open"), Type.Literal("in_progress")])),
			tags: Type.Optional(Type.Array(Type.String())),
		}),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			const args = ["update", params.ref];
			for (const field of ["title", "description", "notes", "assignee", "priority", "status", "tags"] as const) {
				const value = params[field];
				if (value !== undefined) args.push(`--${field}`, Array.isArray(value) ? value.join(",") : String(value));
			}
			return writeTask(args, "actualizada", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_close",
		label: "Cerrar tarea",
		description: "Cierra una tarea con un motivo cuando termina o deja de ser necesaria; sus subtareas deben estar cerradas.",
		parameters: Type.Object({
			ref: reference,
			reason: Type.Union([
				Type.Literal("done"), Type.Literal("duplicate"), Type.Literal("absorbed"),
				Type.Literal("stale"), Type.Literal("superseded"),
			]),
		}),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			return writeTask(["close", params.ref, "--reason", params.reason], "cerrada", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_reopen",
		label: "Reabrir tarea",
		description: "Reabre una tarea cerrada cuando vuelve a necesitar trabajo y retira su motivo de cierre.",
		parameters: Type.Object({ ref: reference }),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			return writeTask(["reopen", params.ref], "reabierta", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_block",
		label: "Bloquear tarea",
		description: "Añade una dependencia cuando una tarea debe esperar a otra; blocker es la tarea que la bloquea.",
		parameters: Type.Object({ ref: reference, blocker: reference }),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			return writeTask(["block", params.ref, params.blocker], "bloqueo añadido", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_unblock",
		label: "Retirar bloqueo",
		description: "Retira la dependencia de blocker cuando una tarea ya no debe esperar a esa otra tarea.",
		parameters: Type.Object({ ref: reference, blocker: reference }),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			return writeTask(["unblock", params.ref, params.blocker], "bloqueo retirado", ctx, signal);
		},
	});

	pi.registerTool({
		name: "tasks_comment",
		label: "Comentar tarea",
		description: "Añade un comentario para dejar avances o contexto en una tarea; no edita comentarios anteriores.",
		parameters: Type.Object({
			ref: reference,
			body: Type.String({ minLength: 1, pattern: "\\S", description: "el texto no vacío del comentario" }),
		}),
		async execute(_toolCallId, params, signal, _onUpdate, ctx) {
			return writeTask(["comment", params.ref, "--body", params.body], "comentario añadido", ctx, signal);
		},
	});
}
