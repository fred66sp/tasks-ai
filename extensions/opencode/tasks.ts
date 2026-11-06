// El tablero como herramientas del agente OpenCode, sobre el CLI del producto
// (ADR-015). Las reglas, los mensajes y los rechazos son del producto: aquí no se
// reescriben. El tablero que se enseña al modelo y su recorte viven en la
// extensión de Pi, que no sabe de ningún agente; esto solo la conecta a OpenCode.

import {
	deriveBoard,
	renderBoard,
	renderComments,
	renderDetail,
	renderSummary,
	type Comment,
	type Detail,
	type Task,
} from "../pi/board.ts";
import { findProjectRoot, runCli, type Outcome } from "../pi/cli.ts";

/** El contexto de un plugin de OpenCode: solo lo que este adaptador usa. */
export interface PluginContext {
	location: { directory: string };
	tool: { transform(edit: (editor: ToolEditor) => void): Promise<unknown> };
	session: {
		hook(
			kind: "context",
			handler: (event: { system: TextBlock[] }) => void | Promise<void>,
		): Promise<unknown>;
	};
}

interface TextBlock {
	type: "text";
	text: string;
}

interface ToolEditor {
	add(tool: ToolDefinition): void;
}

export interface ToolDefinition {
	name: string;
	description: string;
	input: Schema;
	execute(input: Record<string, unknown>, context?: { signal?: AbortSignal }): Promise<{ content: string }>;
}

interface Schema {
	type: "object";
	properties: Record<string, unknown>;
	required?: string[];
	additionalProperties: false;
}

const REFERENCE = { type: "string", description: "la referencia visible de la tarea, por ejemplo tk-18" };
const PRIORITY = { type: "integer", minimum: 0, maximum: 3 };
const REASON = { type: "string", enum: ["done", "duplicate", "absorbed", "stale", "superseded"] };
const STATUS = { type: "string", enum: ["open", "in_progress"] };
const BODY = { type: "string", minLength: 1, pattern: "\\S" };

function schema(properties: Record<string, unknown>, required: string[] = []): Schema {
	return { type: "object", properties, required, additionalProperties: false };
}

/** Añade a command las banderas de los campos que la llamada nombra. */
function withFlags(command: string[], input: Record<string, unknown>, fields: readonly string[]): string[] {
	for (const field of fields) {
		const value = input[field];
		if (value !== undefined) command.push(`--${field}`, Array.isArray(value) ? value.join(",") : String(value));
	}
	return command;
}

/** Una línea para el modelo: la respuesta completa del producto no viaja. */
function line(task: Task, action: string): string {
	const title = task.title.replace(/\s+/g, " ").trim();
	return `${task.ref} · ${action} [${task.status}] ${title.length > 60 ? `${title.slice(0, 59)}…` : title}`;
}

/**
 * Registra las herramientas del tablero y el resumen que acompaña a cada petición.
 * Devuelve cuando las dos cosas están registradas.
 */
export async function register(ctx: PluginContext, executeCli = runCli): Promise<void> {
	const directory = ctx.location.directory;
	let cached: { at: number; pending: Promise<Outcome> } | undefined;

	async function readBoard(signal?: AbortSignal) {
		if (!cached || Date.now() - cached.at >= 1000) {
			cached = { at: Date.now(), pending: executeCli(["list"], directory, signal) };
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

	async function write(args: string[], action: string, signal?: AbortSignal): Promise<string> {
		const outcome = await executeCli(args, directory, signal);
		if (!outcome.ok) throw new Error(outcome.message);
		cached = undefined;
		return line(outcome.data as Task, action);
	}

	async function summary(signal?: AbortSignal): Promise<string> {
		if (findProjectRoot(directory) === null) {
			return "Tablero: no hay tablero instalado en este proyecto; si lo hay, es el único registro de tareas.";
		}
		try {
			return `Tablero del proyecto:\n${renderSummary(await readBoard(signal))}`;
		} catch (error) {
			return `Tablero no disponible: ${error instanceof Error ? error.message : String(error)}`;
		}
	}

	const tools: ToolDefinition[] = [
		{
			name: "tasks_board",
			description:
				"El tablero del proyecto: cuánto hay abierto, qué está en curso, qué se puede empezar y el árbol de tareas. No trae descripciones ni comentarios; para eso está tasks_show.",
			input: schema({}),
			async execute(_input, context) {
				return { content: renderBoard(await readBoard(context?.signal)) };
			},
		},
		{
			name: "tasks_show",
			description:
				"Una tarea del proyecto con su descripción, sus hijas, sus bloqueos y sus últimos comentarios. Es la puerta al detalle: úsala cuando el tablero no baste.",
			input: schema({ ref: REFERENCE }, ["ref"]),
			async execute(input, context) {
				const outcome = await executeCli(["show", String(input.ref)], directory, context?.signal);
				if (!outcome.ok) throw new Error(outcome.message);
				return { content: renderDetail(outcome.data as Detail) };
			},
		},
		{
			name: "tasks_create",
			description:
				"Crea una tarea; usa parent para crear una subtarea de una tarea existente y tags para darle las etiquetas desde el primer momento.",
			input: schema(
				{
					title: { type: "string", pattern: "\\S", description: "el título de la tarea" },
					description: { type: "string" },
					notes: { type: "string" },
					assignee: { type: "string" },
					priority: PRIORITY,
					parent: REFERENCE,
					tags: { type: "array", items: { type: "string" } },
				},
				["title"],
			),
			async execute(input, context) {
				const args = withFlags(["create"], input, ["title", "description", "notes", "assignee", "priority", "parent", "tags"]);
				return { content: await write(args, "creada", context?.signal) };
			},
		},
		{
			name: "tasks_update",
			description:
				"Cambia solo los campos indicados de una tarea. Usa status solo; tags sustituye todas sus etiquetas; parent la asigna, la mueve o, con el texto vacío, la deja sin padre. Al empezar el trabajo en una tarea, ponla en curso con status in_progress; al dejarla, devuélvela con status open.",
			input: schema(
				{
					ref: REFERENCE,
					title: { type: "string", pattern: "\\S" },
					description: { type: "string" },
					notes: { type: "string" },
					assignee: { type: "string" },
					priority: PRIORITY,
					status: STATUS,
					tags: { type: "array", items: { type: "string" } },
					parent: { type: "string", description: "la referencia visible del padre: la asigna o la mueve, y el texto vacío deja la tarea sin padre" },
				},
				["ref"],
			),
			async execute(input, context) {
				const args = withFlags(["update", String(input.ref)], input, [
					"title", "description", "notes", "assignee", "priority", "status", "tags", "parent",
				]);
				return { content: await write(args, "actualizada", context?.signal) };
			},
		},
		{
			name: "tasks_close",
			description:
				"Cierra una tarea con un motivo cuando termina o deja de ser necesaria; sus subtareas deben estar cerradas. Con done, cierra solo después de presentar el resultado a la persona: con su revisión o su pedido expreso.",
			input: schema({ ref: REFERENCE, reason: REASON }, ["ref", "reason"]),
			async execute(input, context) {
				return { content: await write(["close", String(input.ref), "--reason", String(input.reason)], "cerrada", context?.signal) };
			},
		},
		{
			name: "tasks_reopen",
			description: "Reabre una tarea cerrada cuando vuelve a necesitar trabajo y retira su motivo de cierre.",
			input: schema({ ref: REFERENCE }, ["ref"]),
			async execute(input, context) {
				return { content: await write(["reopen", String(input.ref)], "reabierta", context?.signal) };
			},
		},
		{
			name: "tasks_block",
			description: "Añade una dependencia cuando una tarea debe esperar a otra; blocker es la tarea que la bloquea.",
			input: schema({ ref: REFERENCE, blocker: REFERENCE }, ["ref", "blocker"]),
			async execute(input, context) {
				return { content: await write(["block", String(input.ref), String(input.blocker)], "bloqueo añadido", context?.signal) };
			},
		},
		{
			name: "tasks_unblock",
			description: "Retira la dependencia de blocker cuando una tarea ya no debe esperar a esa otra tarea.",
			input: schema({ ref: REFERENCE, blocker: REFERENCE }, ["ref", "blocker"]),
			async execute(input, context) {
				return { content: await write(["unblock", String(input.ref), String(input.blocker)], "bloqueo retirado", context?.signal) };
			},
		},
		{
			name: "tasks_comment",
			description:
				"Añade un comentario para dejar avances o contexto en una tarea; para leerlos tasks_comment_list y para cambiar el cuerpo de uno tasks_comment_edit. Escribe el avance, la decisión o el bloqueo en el momento en que surgen, no al final.",
			input: schema({ ref: REFERENCE, body: BODY }, ["ref", "body"]),
			async execute(input, context) {
				return { content: await write(["comment", String(input.ref), "--body", String(input.body)], "comentario añadido", context?.signal) };
			},
		},
		{
			name: "tasks_comment_list",
			description: "Lee los comentarios de una tarea, del primero al último, cada uno con su identidad, su cuerpo y su fecha.",
			input: schema({ ref: REFERENCE }, ["ref"]),
			async execute(input, context) {
				const outcome = await executeCli(["comment", "list", String(input.ref)], directory, context?.signal);
				if (!outcome.ok) throw new Error(outcome.message);
				return { content: renderComments(outcome.data as Comment[]) };
			},
		},
		{
			name: "tasks_comment_edit",
			description:
				"Cambia el cuerpo del comentario de una tarea que lleva el número dado; la identidad y la fecha del comentario se conservan.",
			input: schema({ ref: REFERENCE, id: { type: "integer", minimum: 1 }, body: BODY }, ["ref", "id", "body"]),
			async execute(input, context) {
				return {
					content: await write(
						["comment", "edit", String(input.ref), String(input.id), "--body", String(input.body)],
						"comentario editado",
						context?.signal,
					),
				};
			},
		},
	];

	await ctx.tool.transform((editor) => {
		for (const tool of tools) editor.add(tool);
	});

	await ctx.session.hook("context", async (event) => {
		event.system.push({ type: "text", text: await summary() });
	});
}
