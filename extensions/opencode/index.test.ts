// Pruebas del adaptador de OpenCode: el registro de las herramientas, el mapeo al
// CLI, el resumen que entra en el contexto y los rechazos del producto. El CLI se
// sustituye por uno falso: lo que se comprueba es el adaptador, no el producto, que
// tiene sus propias pruebas.

import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test, { type TestContext } from "node:test";
import { register, type ToolDefinition } from "./tasks.ts";
import type { Outcome } from "../pi/cli.ts";
import type { Task } from "../pi/board.ts";

const names = [
	"tasks_board", "tasks_show", "tasks_create", "tasks_update", "tasks_close",
	"tasks_reopen", "tasks_block", "tasks_unblock", "tasks_comment",
	"tasks_comment_list", "tasks_comment_edit",
];

interface Harness {
	tools: Map<string, ToolDefinition>;
	hooks: ((event: { system: { type: "text"; text: string }[] }) => void | Promise<void>)[];
	calls: string[][];
}

function project(t: TestContext, withBoard = true): string {
	const dir = mkdtempSync(path.join(tmpdir(), "tasks-ai-opencode-"));
	if (withBoard) mkdirSync(path.join(dir, ".tasks-ai"));
	t.after(() => rmSync(dir, { recursive: true, force: true }));
	return dir;
}

function task(fields: Partial<Task> = {}): Task {
	return {
		id: 1, ref: "tk-1", title: "Preparar el análisis", status: "open", blocked: false,
		priority: 2, parent_id: null, tags: ["docs"], updated_at: "2026-01-01T00:00:00Z",
		...fields,
	};
}

async function harness(directory: string, reply: (args: string[]) => Outcome): Promise<Harness> {
	const tools = new Map<string, ToolDefinition>();
	const hooks: Harness["hooks"] = [];
	const calls: string[][] = [];

	const executeCli = async (args: string[]): Promise<Outcome> => {
		calls.push(args);
		return reply(args);
	};

	await register(
		{
			location: { directory },
			tool: {
				async transform(edit) {
					edit({ add: (tool) => tools.set(tool.name, tool) });
				},
			},
			session: {
				async hook(_kind, handler) {
					hooks.push(handler);
				},
			},
		},
		executeCli,
	);

	return { tools, hooks, calls };
}

test("registra las once herramientas del contrato", async (t) => {
	const { tools, hooks } = await harness(project(t), () => ({ ok: true, data: [] }));

	assert.deepEqual([...tools.keys()].sort(), [...names].sort());
	for (const tool of tools.values()) {
		assert.ok(tool.description.trim() !== "", `${tool.name} sin descripción`);
		assert.equal(tool.input.type, "object");
		assert.equal(tool.input.additionalProperties, false);
	}
	assert.equal(hooks.length, 1);
});

test("las descripciones dicen el momento de uso", async (t) => {
	const { tools } = await harness(project(t), () => ({ ok: true, data: [] }));

	assert.match(tools.get("tasks_update")!.description, /in_progress.*status open/s);
	assert.match(tools.get("tasks_comment")!.description, /en el momento en que surgen/);
	assert.match(tools.get("tasks_close")!.description, /revisión o su pedido expreso/);
	assert.match(tools.get("tasks_close")!.description, /una tarea pendiente no se cierra como done/);
});

test("tasks_board deriva el tablero de list", async (t) => {
	const tasks = [task(), task({ id: 2, ref: "tk-2", title: "Revisar", status: "done", updated_at: "2026-01-02T00:00:00Z" })];
	const { tools, calls } = await harness(project(t), () => ({ ok: true, data: tasks }));

	const result = await tools.get("tasks_board")!.execute({});

	assert.deepEqual(calls, [["list"]]);
	assert.match(result.content, /tareas: 2 \(abiertas 1, en curso 0, cerradas 1\)/);
});

test("una escritura manda solo los campos que nombra y descarta el tablero cacheado", async (t) => {
	const { tools, calls } = await harness(project(t), (args) => {
		if (args[0] === "list") return { ok: true, data: [] };
		return { ok: true, data: task({ ref: "tk-9", title: "Nueva", status: "open" }) };
	});

	await tools.get("tasks_board")!.execute({});
	const created = await tools.get("tasks_create")!.execute({ title: "Nueva", parent: "tk-1", tags: ["feat", "task"] });
	await tools.get("tasks_board")!.execute({});

	assert.deepEqual(calls[1], ["create", "--title", "Nueva", "--parent", "tk-1", "--tags", "feat,task"]);
	assert.match(created.content, /^tk-9 · creada \[open\] Nueva/);
	assert.equal(calls.filter((args) => args[0] === "list").length, 2);
});

test("tasks_update manda el padre vacío para quitarlo", async (t) => {
	const { tools, calls } = await harness(project(t), () => ({ ok: true, data: task() }));

	await tools.get("tasks_update")!.execute({ ref: "tk-1", status: "in_progress", parent: "" });

	assert.deepEqual(calls[0], ["update", "tk-1", "--status", "in_progress", "--parent", ""]);
});

test("tasks_comment_edit manda el número del comentario", async (t) => {
	const { tools, calls } = await harness(project(t), () => ({ ok: true, data: task() }));

	await tools.get("tasks_comment_edit")!.execute({ ref: "tk-1", id: 3, body: "hola" });

	assert.deepEqual(calls[0], ["comment", "edit", "tk-1", "3", "--body", "hola"]);
});

test("un rechazo del producto llega con su mensaje", async (t) => {
	const { tools } = await harness(project(t), () => ({ ok: false, code: 1, message: "task tk-9 not found" }));

	await assert.rejects(tools.get("tasks_show")!.execute({ ref: "tk-9" }), /task tk-9 not found/);
});

test("el resumen del tablero entra en el contexto de cada petición", async (t) => {
	const { hooks } = await harness(project(t), () => ({ ok: true, data: [task()] }));

	const event = { system: [] as { type: "text"; text: string }[] };
	await hooks[0](event);

	assert.equal(event.system.length, 1);
	assert.equal(event.system[0].type, "text");
	assert.match(event.system[0].text, /Tablero del proyecto:/);
	assert.match(event.system[0].text, /tareas: 1/);
	assert.match(event.system[0].text, /Empezar el trabajo: tasks_update con status in_progress/);
	assert.match(event.system[0].text, /revisión o su pedido expreso/);
});

test("sin tablero, el contexto lo dice en vez de fallar", async (t) => {
	const { hooks } = await harness(project(t, false), () => ({ ok: true, data: [] }));

	const event = { system: [] as { type: "text"; text: string }[] };
	await hooks[0](event);

	assert.match(event.system[0].text, /no hay tablero instalado/);
});

test("si el producto no responde, el contexto lo cuenta sin romper la petición", async (t) => {
	const { hooks } = await harness(project(t), () => ({ ok: false, code: -1, message: "could not run tasks-ai" }));

	const event = { system: [] as { type: "text"; text: string }[] };
	await hooks[0](event);

	assert.match(event.system[0].text, /Tablero no disponible: could not run tasks-ai/);
});
