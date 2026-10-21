import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, it } from "node:test";
import type { Detail } from "./board.ts";
import { runCli } from "./cli.ts";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

import tasksExtension from "./index.ts";
import { binary, ensureBinary } from "./test-binary.ts";

interface Registered {
	name: string;
	parameters: { properties: Record<string, Record<string, unknown>>; required?: string[] };
	execute: (
		id: string,
		params: Record<string, unknown>,
		signal: undefined,
		onUpdate: undefined,
		ctx: { cwd: string },
	) => Promise<{ content: { type: string; text: string }[]; details: unknown }>;
}

function harness(executeCli = runCli, cwd?: string) {
	const commands: Record<string, { handler: (args: string, ctx: any) => Promise<void> }> = {};
	const notifications: string[] = [];
	const statuses = new Map<string, string>();
	const widgets = new Map<string, string[]>();
	const ui = {
		notify(text: string) { notifications.push(text); },
		setStatus(key: string, text: string | undefined) {
			if (text === undefined) statuses.delete(key);
			else statuses.set(key, text);
		},
		setWidget(key: string, lines: string[] | undefined) {
			if (lines === undefined) widgets.delete(key);
			else widgets.set(key, lines);
		},
	};
	const context = (cwd: string, mode = "tui") => ({ cwd, mode, hasUI: mode === "tui", ui });
	const registered: Record<string, Registered> = {};
	const handlers: Record<string, (event: any, ctx: any) => any> = {};
	let active = ["read", "todo", "bash"];
	const changes: string[][] = [];
	const pi = {
		registerCommand(name: string, command: typeof commands[string]) {
			commands[name] = command;
		},
		registerTool(tool: Registered) {
			registered[tool.name] = tool;
			active.push(tool.name);
		},
		on(name: string, handler: (event: any, ctx: any) => any) {
			handlers[name] = handler;
			return () => {};
		},
		getActiveTools: () => [...active],
		setActiveTools(names: string[]) {
			active = [...names];
			changes.push([...names]);
		},
	} as unknown as ExtensionAPI;

	tasksExtension(pi, executeCli);
	const previous = [...active];
	const start = (ctx: any = cwd === undefined ? {} : { cwd }) => handlers.session_start?.({ type: "session_start", reason: "startup" }, ctx);
	start();
	return { registered, commands, notifications, statuses, widgets, context, handlers, previous, changes, start, active: () => [...active] };
}

function load(cwd?: string): Record<string, Registered> {
	return harness(undefined, cwd).registered;
}

describe("la extensión", () => {
	let project: string;
	let tools: Record<string, Registered>;

	beforeEach(() => {
		ensureBinary();
		process.env.TASKS_AI_BIN = binary;
		project = mkdtempSync(path.join(tmpdir(), "tasks-ai-extension-"));
		const init = spawnSync(binary, ["init"], { cwd: project, encoding: "utf8" });
		assert.equal(init.status, 0, init.stderr);
		tools = load(project);
	});

	afterEach(() => {
		rmSync(project, { recursive: true, force: true });
	});

	function cli(...args: string[]) {
		const result = spawnSync(binary, args, { cwd: project, encoding: "utf8" });
		assert.equal(result.status, 0, result.stderr);
		return JSON.parse(result.stdout);
	}

	async function execute(name: string, params: Record<string, unknown>) {
		assert.ok(tools[name], `falta la herramienta ${name}`);
		return tools[name].execute("id", params, undefined, undefined, { cwd: project });
	}

	async function write(name: string, params: Record<string, unknown>) {
		const result = await execute(name, params);
		const task = result.details as Detail;
		assert.equal(result.content.length, 1);
		assert.equal(result.content[0].type, "text");
		assert.ok(result.content[0].text.startsWith(`${task.ref} · `));
		assert.ok(result.content[0].text.includes(`[${task.status}]`));
		assert.ok(result.content[0].text.length < 160);
		assert.ok(!result.content[0].text.includes("\n"));
		assert.deepEqual(result.details, (cli("list") as Detail[]).find((entry) => entry.ref === task.ref));
		return task;
	}

	it("/board enseña la épica, las subtareas y los bloqueos sin mensajes al modelo", async () => {
		cli("create", "--title", "épica del TUI");
		cli("create", "--title", "comando /board", "--parent", "tk-1");
		cli("create", "--title", "verificar comando", "--parent", "tk-1");
		cli("block", "tk-3", "tk-2");
		const extension = harness();
		assert.ok(extension.commands.board, "falta /board");
		assert.equal(await extension.commands.board.handler("", extension.context(project)), undefined);
		const text = extension.notifications.join("\n");
		assert.match(text, /tk-1 · épica del TUI/);
		assert.match(text, /  - tk-2 · comando \/board/);
		assert.match(text, /  - tk-3 · verificar comando \[open, bloqueada\]/);
	});

	it("/board y el estado sin instalación informan sin ejecutar el CLI", async () => {
		let calls = 0;
		const extension = harness(async () => { calls++; throw new Error("no debe ejecutarse"); });
		const ctx = extension.context(path.parse(project).root);
		assert.ok(extension.commands.board, "falta /board");
		await extension.commands.board.handler("", ctx);
		assert.match(extension.notifications.join(""), /no.*instalado/i);
		assert.match([...extension.statuses.values()].join(""), /no.*instalado/i);
		assert.equal(calls, 0);
	});

	it("el estado muestra la tarea en curso al iniciar y se limpia al descargar", async () => {
		cli("create", "--title", "09 comando /board");
		cli("update", "tk-1", "--status", "in_progress");
		const extension = harness();
		await extension.handlers.session_start({}, extension.context(project));
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: tk-1 · 09 comando /board"]);
		await extension.handlers.session_shutdown({}, extension.context(project));
		assert.equal(extension.statuses.size, 0);
	});

	it("el estado nombra todas las tareas en curso", async () => {
		cli("create", "--title", "primera en curso");
		cli("create", "--title", "segunda en curso");
		cli("update", "tk-1", "--status", "in_progress");
		cli("update", "tk-2", "--status", "in_progress");
		const extension = harness();
		await extension.handlers.session_start({}, extension.context(project));
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: 2 en curso · tk-1 · primera en curso, tk-2 · segunda en curso"]);
	});

	it("el tablero se ve como bloque con una línea por tarea en curso", async () => {
		cli("create", "--title", "primera en curso");
		cli("create", "--title", "segunda en curso");
		cli("update", "tk-1", "--status", "in_progress");
		cli("update", "tk-2", "--status", "in_progress");
		const extension = harness(undefined, project);
		await extension.handlers.session_start({}, extension.context(project));
		assert.deepEqual(extension.widgets.get("tasks_board"), [
			"🦊 Tasks-ai tablero:",
			"  ○ tk-1 · primera en curso",
			"  ○ tk-2 · segunda en curso",
		]);
	});

	it("el bloque dice que no hay nada en curso y se retira sin tablero o al descargar", async () => {
		cli("create", "--title", "pendiente");
		const extension = harness(undefined, project);
		await extension.handlers.session_start({}, extension.context(project));
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ sin tarea en curso"]);
		await extension.handlers.session_start({}, extension.context(path.parse(project).root));
		assert.equal(extension.widgets.has("tasks_board"), false, "sin tablero el bloque se retira");
		await extension.handlers.session_start({}, extension.context(project));
		await extension.handlers.session_shutdown({}, {});
		assert.equal(extension.widgets.has("tasks_board"), false, "al descargar el bloque se retira");
	});

	it("/board comparte list con el resumen, la herramienta y llamadas consecutivas", async () => {
		let lists = 0;
		const extension = harness(async (...args) => {
			if (args[0][0] === "list") lists++;
			return runCli(...args);
		});
		const ctx = extension.context(project);
		assert.ok(extension.commands.board, "falta /board");
		await extension.handlers.before_agent_start({ systemPromptOptions: { sections: {} } }, ctx);
		await extension.commands.board.handler("", ctx);
		await extension.commands.board.handler("", ctx);
		await extension.registered.tasks_board.execute("id", {}, undefined, undefined, ctx);
		assert.equal(lists, 1);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: sin tarea en curso"]);
	});

	it("/board en print y json no usa UI ni lee el CLI", async () => {
		const extension = harness(async () => { throw new Error("no debe ejecutarse"); });
		assert.ok(extension.commands.board, "falta /board");
		for (const mode of ["print", "json"]) {
			await extension.commands.board.handler("", { ...extension.context(project, mode), ui: undefined });
		}
		assert.deepEqual(extension.notifications, []);
		assert.equal(extension.statuses.size, 0);
	});

	it("/board informa de un fallo de lectura y permite reintentar", async () => {
		let calls = 0;
		const extension = harness(async () => ++calls === 1
			? { ok: false, code: 1, message: "fallo de lectura" }
			: { ok: true, data: [] });
		const ctx = extension.context(project);
		await extension.commands.board.handler("", ctx);
		assert.match(extension.notifications[0], /Tablero no disponible: fallo de lectura/);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: no disponible"]);
		await extension.commands.board.handler("", ctx);
		assert.equal(calls, 2);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: sin tarea en curso"]);
	});

	it("el estado y el bloque se actualizan tras escribir y al comenzar un turno", async (t) => {
		let now = 0;
		t.mock.method(Date, "now", () => now);
		cli("create", "--title", "tarea\nactual");
		const extension = harness();
		const ctx = extension.context(project);
		await extension.handlers.session_start({}, ctx);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: sin tarea en curso"]);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ sin tarea en curso"]);
		await extension.registered.tasks_update.execute("id", { ref: "tk-1", status: "in_progress" }, undefined, undefined, ctx);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: tk-1 · tarea actual"]);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ tk-1 · tarea actual"]);
		cli("update", "tk-1", "--status", "open");
		cli("update", "tk-1", "--status", "in_progress");
		cli("create", "--title", "otra");
		cli("update", "tk-2", "--status", "in_progress");
		cli("update", "tk-1", "--status", "open");
		// La siguiente lectura debe ver también cambios externos una vez caducada la caché.
		now = 1000;
		await extension.handlers.before_agent_start({ systemPromptOptions: { sections: {} } }, ctx);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: tk-2 · otra"]);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ tk-2 · otra"]);
		cli("close", "tk-2", "--reason", "done");
		now = 2000;
		await extension.handlers.before_agent_start({ systemPromptOptions: { sections: {} } }, ctx);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: sin tarea en curso"]);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ sin tarea en curso"]);
	});

	it("tasks_board refresca el bloque además de la línea de estado", async (t) => {
		let now = 0;
		t.mock.method(Date, "now", () => now);
		cli("create", "--title", "primera");
		const extension = harness(undefined, project);
		const ctx = extension.context(project);
		await extension.handlers.session_start({}, ctx);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ sin tarea en curso"]);
		cli("update", "tk-1", "--status", "in_progress");
		now = 1000;
		await extension.registered.tasks_board.execute("id", {}, undefined, undefined, ctx);
		assert.deepEqual([...extension.statuses.values()], ["🦊 tablero: tk-1 · primera"]);
		assert.deepEqual(extension.widgets.get("tasks_board"), ["🦊 Tasks-ai tablero:", "  ○ tk-1 · primera"]);
	});

	it("añade al prompt un resumen corto, sin mensaje ni detalles privados", async () => {
		cli("create", "--title", "trabajo actual", "--description", "detalle privado");
		cli("update", "tk-1", "--status", "in_progress");
		for (let i = 0; i < 7; i++) cli("create", "--title", `lista ${i}\nsegunda línea`);
		cli("block", "tk-2", "tk-1");
		const { handlers } = harness();
		assert.ok(handlers.before_agent_start, "falta before_agent_start");
		const sections = { existing: "conservar" };
		const result = await handlers.before_agent_start({ systemPromptOptions: { sections } }, { cwd: project });
		assert.equal(result?.message, undefined);
		assert.equal(result?.systemPrompt, undefined);
		assert.equal(sections.existing, "conservar");
		const summary = (sections as Record<string, string>).tasks_board;
		assert.match(summary, /tareas: 8 \(abiertas 7, en curso 1, cerradas 0\)/);
		assert.match(summary, /en curso: .*tk-1 · trabajo actual/);
		assert.match(summary, /tk-3 · lista 1/);
		assert.doesNotMatch(summary, /tk-2 ·|tk-8 ·|detalle privado/);
		assert.ok(summary.split("\n").length <= 12);
		assert.match(summary, /crear o seleccionar.*antes de tocar archivos/i);
		assert.match(summary, /marcar.*en curso/i);
		assert.match(summary, /decisiones y bloqueos.*comentarios/i);
		assert.match(summary, /cerrar solo tras verificar/i);
	});

	it("sin instalación informa en una línea y no ejecuta el CLI", async () => {
		let calls = 0;
		const { handlers } = harness(async () => { calls++; throw new Error("no debe ejecutarse"); });
		assert.ok(handlers.before_agent_start, "falta before_agent_start");
		const sections: Record<string, string> = {};
		await handlers.before_agent_start({ systemPromptOptions: { sections } }, { cwd: path.parse(project).root });
		assert.match(sections.tasks_board, /no.*instalado/i);
		assert.equal(sections.tasks_board.split("\n").length, 1);
		assert.equal(calls, 0);
	});

	it("retira solo todo al iniciar y restaura exactamente una vez al descargar", async () => {
		const extension = harness(undefined, project);
		assert.deepEqual(extension.active(), extension.previous.filter((name) => name !== "todo"));
		extension.start();
		assert.ok(extension.handlers.session_shutdown, "falta session_shutdown");
		await extension.handlers.session_shutdown({}, {});
		assert.deepEqual(extension.active(), extension.previous);
		const changes = extension.changes.length;
		await extension.handlers.session_shutdown({}, {});
		assert.equal(extension.changes.length, changes);
	});

	it("sin tablero deja el todo del arnés y lo dice en el resumen", async () => {
		const root = path.parse(project).root;
		const extension = harness(async () => { throw new Error("no debe ejecutarse"); }, root);
		assert.deepEqual(extension.active(), extension.previous, "todo sigue disponible");
		const sections: Record<string, string> = {};
		await extension.handlers.before_agent_start({ systemPromptOptions: { sections } }, extension.context(root));
		assert.equal(sections.tasks_board.split("\n").length, 1);
		assert.match(sections.tasks_board, /todo/);
	});

	it("comparte list entre el resumen y tasks_board e invalida tras escribir", async () => {
		let lists = 0;
		const extension = harness(async (...args) => {
			if (args[0][0] === "list") lists++;
			return runCli(...args);
		});
		assert.ok(extension.handlers.before_agent_start, "falta before_agent_start");
		await extension.handlers.before_agent_start({ systemPromptOptions: { sections: {} } }, { cwd: project });
		const board = extension.registered.tasks_board;
		await board.execute("id", {}, undefined, undefined, { cwd: project });
		await board.execute("id", {}, undefined, undefined, { cwd: project });
		assert.equal(lists, 1);
		await extension.registered.tasks_create.execute("id", { title: "nueva" }, undefined, undefined, { cwd: project });
		const updated = await board.execute("id", {}, undefined, undefined, { cwd: project });
		assert.equal(lists, 2);
		assert.match(updated.content[0].text, /tareas: 1/);
	});

	it("la caché caduca al segundo y no se comparte entre directorios", async (t) => {
		let now = 0;
		t.mock.method(Date, "now", () => now);
		let calls = 0;
		const extension = harness(async () => { calls++; return { ok: true, data: [] }; });
		const board = extension.registered.tasks_board;
		const read = (cwd = project) => board.execute("id", {}, undefined, undefined, { cwd });
		await read();
		now = 999;
		await read();
		assert.equal(calls, 1);
		now = 1000;
		await read();
		assert.equal(calls, 2);
		await read(path.join(project, "subdirectorio"));
		assert.equal(calls, 3);
	});

	it("comparte lecturas simultáneas y no guarda rechazos de list", async () => {
		let calls = 0;
		const extension = harness(async () => {
			calls++;
			return calls === 1 ? { ok: false, code: 1, message: "rechazo exacto" } : { ok: true, data: [] };
		});
		const read = () => extension.registered.tasks_board.execute("id", {}, undefined, undefined, { cwd: project });
		await assert.rejects(read, /^Error: rechazo exacto$/);
		await Promise.all([read(), read()]);
		assert.equal(calls, 2);
	});

	it("todas las escrituras invalidan y un rechazo conserva la lectura", async () => {
		let lists = 0;
		let reject = false;
		const extension = harness(async (args) => {
			if (args[0] === "list") { lists++; return { ok: true, data: [] }; }
			return reject ? { ok: false, code: 2, message: "entrada inválida" }
				: { ok: true, data: { ref: "tk-1", title: "tarea", status: "open" } };
		});
		const invoke = (name: string) => extension.registered[name].execute("id", {
			ref: "tk-1", title: "tarea", reason: "done", blocker: "tk-2", body: "comentario",
		}, undefined, undefined, { cwd: project });
		await invoke("tasks_board");
		for (const name of ["tasks_create", "tasks_update", "tasks_close", "tasks_reopen", "tasks_block", "tasks_unblock", "tasks_comment"]) {
			await invoke(name);
			await invoke("tasks_board");
		}
		assert.equal(lists, 8);
		reject = true;
		await assert.rejects(() => invoke("tasks_update"), /entrada inválida/);
		await invoke("tasks_board");
		assert.equal(lists, 8);
	});

	it("registra las tres herramientas de lectura y las ocho de escritura", () => {
		assert.deepEqual(Object.keys(tools).sort(), [
			"tasks_block", "tasks_board", "tasks_close", "tasks_comment", "tasks_comment_edit",
			"tasks_comment_list", "tasks_create", "tasks_reopen", "tasks_show", "tasks_unblock", "tasks_update",
		]);
	});

	it("los esquemas declaran los límites y enumerados del CLI sin valores por omisión", () => {
		for (const name of ["tasks_create", "tasks_update"]) {
			const fields = tools[name].parameters.properties;
			assert.equal(fields.priority.type, "integer");
			assert.equal(fields.priority.minimum, 0);
			assert.equal(fields.priority.maximum, 3);
			for (const field of Object.values(fields)) assert.equal(field.default, undefined);
		}
		assert.deepEqual(tools.tasks_create.parameters.required, ["title"]);
		assert.deepEqual(tools.tasks_update.parameters.required, ["ref"]);
		assert.equal(tools.tasks_create.parameters.properties.tags, undefined);
		const reasons = tools.tasks_close.parameters.properties.reason.anyOf as { const: string }[];
		assert.deepEqual(reasons.map((value) => value.const), ["done", "duplicate", "absorbed", "stale", "superseded"]);
		const statuses = tools.tasks_update.parameters.properties.status.anyOf as { const: string }[];
		assert.deepEqual(statuses.map((value) => value.const), ["open", "in_progress"]);
		assert.equal(tools.tasks_update.parameters.properties.parent.type, "string");
		assert.equal(tools.tasks_update.parameters.properties.parent.default, undefined);
		const body = tools.tasks_comment.parameters.properties.body;
		assert.equal(body.minLength, 1);
		assert.ok(!new RegExp(body.pattern as string).test(" \n\t"));
		const edit = tools.tasks_comment_edit.parameters.properties;
		assert.equal(edit.id.type, "integer");
		assert.equal(edit.id.minimum, 1);
		assert.equal(edit.body.minLength, 1);
		assert.ok(!new RegExp(edit.body.pattern as string).test(" \n\t"));
		assert.deepEqual(tools.tasks_comment_edit.parameters.required, ["ref", "id", "body"]);
		assert.deepEqual(tools.tasks_comment_list.parameters.required, ["ref"]);
	});

	it("tasks_board devuelve el tablero del proyecto", async () => {
		spawnSync(binary, ["create", "--title", "una tarea"], { cwd: project, encoding: "utf8" });

		const result = await tools.tasks_board.execute("id", {}, undefined, undefined, { cwd: project });

		assert.match(result.content[0].text, /tareas: 1 \(abiertas 1/);
		assert.match(result.content[0].text, /tk-1 · una tarea/);
	});

	it("tasks_show devuelve la tarea y rechaza una referencia que no existe", async () => {
		cli("create", "--title", "una tarea");
		const shown = await tools.tasks_show.execute("id", { ref: "tk-1" }, undefined, undefined, { cwd: project });
		assert.match(shown.content[0].text, /^tk-1 · una tarea/);

		await assert.rejects(
			() => tools.tasks_show.execute("id", { ref: "tk-999" }, undefined, undefined, { cwd: project }),
			/there is no task with reference "tk-999"/,
		);
	});

	it("tasks_create crea una tarea y una subtarea visibles en el tablero", async () => {
		const parent = await write("tasks_create", {
			title: "padre\ncon detalle", description: "descripción privada", notes: "notas privadas",
			assignee: "equipo", priority: 0,
		});
		const stored = cli("show", parent.ref);
		assert.equal(stored.description, "descripción privada");
		assert.equal(stored.notes, "notas privadas");
		assert.equal(stored.assignee, "equipo");
		assert.equal(stored.priority, 0);
		const child = await write("tasks_create", { title: "hija", parent: " TK-1 " });
		assert.equal(child.parent_id, parent.id);
		const board = await execute("tasks_board", {});
		assert.match(board.content[0].text, /tareas: 2 \(abiertas 2/);
		assert.match(board.content[0].text, /  - tk-2 · hija/);
		assert.doesNotMatch(board.content[0].text, /descripción privada|notas privadas/);
	});

	it("una entrada inválida devuelve el error exacto del CLI y no escribe nada", async () => {
		cli("create", "--title", "existente");
		const before = cli("list");
		const rejected = spawnSync(binary, ["create", "--title", "inválida", "--priority", "4"], {
			cwd: project, encoding: "utf8",
		});
		assert.equal(rejected.status, 2);
		const message = JSON.parse(rejected.stderr || rejected.stdout).error;
		await assert.rejects(() => execute("tasks_create", { title: "inválida", priority: 4 }),
			(error: Error) => error.message === message);
		assert.deepEqual(cli("list"), before);
	});

	it("las herramientas devuelven el rechazo del texto tal cual y no escriben nada", async () => {
		cli("create", "--title", "Base", "--description", "Decidir la proteccion de rama del repositorio");
		await execute("tasks_comment", { ref: "tk-1", body: "cuerpo legitimo" });
		const before = cli("show", "tk-1");

		const scenarios: { name: string; args: string[]; tool: string; params: Record<string, unknown> }[] = [
			{
				name: "la descripción de una tarea nueva nombra un commit",
				args: ["create", "--title", "Otra", "--description", "el commit acd789f lo hizo"],
				tool: "tasks_create", params: { title: "Otra", description: "el commit acd789f lo hizo" },
			},
			{
				name: "las notas de una tarea nueva nombran una rama",
				args: ["create", "--title", "Otra", "--notes", "ver feat/rich-task-text"],
				tool: "tasks_create", params: { title: "Otra", notes: "ver feat/rich-task-text" },
			},
			{
				name: "la descripción de un cambio declara el estado de la tarea",
				args: ["update", "tk-1", "--description", "estado: pendiente"],
				tool: "tasks_update", params: { ref: "tk-1", description: "estado: pendiente" },
			},
			{
				name: "las notas de un cambio declaran el estado de la entrega",
				args: ["update", "tk-1", "--notes", "sin commitear"],
				tool: "tasks_update", params: { ref: "tk-1", notes: "sin commitear" },
			},
			{
				name: "el cuerpo de un comentario nombra un remoto",
				args: ["comment", "tk-1", "--body", "origin/HEAD es la referencia"],
				tool: "tasks_comment", params: { ref: "tk-1", body: "origin/HEAD es la referencia" },
			},
			{
				name: "el cuerpo editado de un comentario declara el estado",
				args: ["comment", "edit", "tk-1", "1", "--body", "Hecho: el criterio"],
				tool: "tasks_comment_edit", params: { ref: "tk-1", id: 1, body: "Hecho: el criterio" },
			},
		];

		for (const scenario of scenarios) {
			const rejected = spawnSync(binary, scenario.args, { cwd: project, encoding: "utf8" });
			assert.equal(rejected.status, 2, scenario.name);
			const message = JSON.parse(rejected.stderr || rejected.stdout).error;
			assert.match(message, /^(the (description|notes|comment) (names|name|declares|declare) )/, scenario.name);
			await assert.rejects(() => execute(scenario.tool, scenario.params),
				(error: Error) => error.message === message, scenario.name);
		}

		// Ningún rechazo dejó rastro: la misma tarea, con su descripción y su comentario.
		assert.deepEqual(cli("show", "tk-1"), before);
		assert.equal(cli("list").length, 1);
	});

	it("tasks_close conserva abierta una tarea con hijas abiertas y propaga el rechazo", async () => {
		cli("create", "--title", "padre");
		cli("create", "--title", "hija", "--parent", "tk-1");
		const before = cli("list");
		const rejected = spawnSync(binary, ["close", " TK-1 ", "--reason", "done"], {
			cwd: project, encoding: "utf8",
		});
		assert.equal(rejected.status, 1);
		const message = JSON.parse(rejected.stderr || rejected.stdout).error;
		await assert.rejects(() => execute("tasks_close", { ref: " TK-1 ", reason: "done" }),
			(error: Error) => error.message === message);
		assert.deepEqual(cli("list"), before);
		assert.equal(cli("show", "tk-1").status, "open");
	});

	it("tasks_close cierra con motivo y tasks_reopen lo elimina", async () => {
		cli("create", "--title", "una tarea");
		const closed = await write("tasks_close", { ref: " TK-1 ", reason: "superseded" });
		assert.equal(closed.status, "done");
		assert.equal(cli("show", "tk-1").close_reason, "superseded");
		const reopened = await write("tasks_reopen", { ref: " TK-1 " });
		assert.equal(reopened.status, "open");
		assert.equal(cli("show", "tk-1").close_reason, null);
	});

	it("tasks_update solo cambia los campos pedidos y pone o quita etiquetas", async () => {
		cli("create", "--title", "original", "--description", "descripción", "--notes", "notas",
			"--assignee", "equipo", "--priority", "0");
		cli("tag", "create", "--name", "docs");
		cli("update", "tk-1", "--status", "in_progress");
		const before = cli("list")[0];
		const renamed = await write("tasks_update", { ref: " TK-1 ", title: "renombrada" });
		assert.deepEqual({ ...renamed, title: before.title, updated_at: before.updated_at }, before);
		const tagged = await write("tasks_update", { ref: "tk-1", tags: ["docs"] });
		assert.deepEqual(tagged.tags, ["docs"]);
		const cleared = await write("tasks_update", {
			ref: "tk-1", tags: [], description: "", notes: "", assignee: "", priority: 3,
		});
		assert.deepEqual(cleared.tags, []);
		const stored = cli("show", "tk-1");
		assert.equal(stored.description, null);
		assert.equal(stored.notes, null);
		assert.equal(stored.assignee, null);
		assert.equal(stored.priority, 3);
		const opened = await write("tasks_update", { ref: "tk-1", status: "open" });
		assert.equal(opened.status, "open");
	});

	it("tasks_update asigna, mueve y retira el padre como el CLI", async () => {
		cli("create", "--title", "primer padre");
		cli("create", "--title", "segundo padre");
		cli("create", "--title", "hija");
		const assigned = await write("tasks_update", { ref: " tk-3 ", parent: " TK-1 " });
		assert.equal(assigned.parent_id, cli("show", "tk-1").id);
		assert.deepEqual(cli("show", "tk-1").children.map((child: { ref: string }) => child.ref), ["tk-3"]);
		const moved = await write("tasks_update", { ref: "tk-3", parent: "tk-2" });
		assert.equal(moved.parent_id, cli("show", "tk-2").id);
		assert.deepEqual(cli("show", "tk-1").children, []);
		assert.deepEqual(cli("show", "tk-2").children.map((child: { ref: string }) => child.ref), ["tk-3"]);
		const removed = await write("tasks_update", { ref: "tk-3", parent: "" });
		assert.equal(removed.parent_id, null);
		assert.deepEqual(cli("show", "tk-2").children, []);
	});

	it("tasks_update compone --parent solo cuando el parámetro se da", async () => {
		cli("create", "--title", "con padre");
		cli("create", "--title", "hija", "--parent", "tk-1");
		const calls: string[][] = [];
		const extension = harness(async (args: string[]) => {
			calls.push(args);
			return runCli(args, project);
		});
		const update = async (params: Record<string, unknown>) => {
			calls.length = 0;
			await extension.registered.tasks_update.execute("id", { ref: "tk-2", ...params },
				undefined, undefined, { cwd: project });
			assert.equal(calls.length, 1);
			return calls[0];
		};
		assert.deepEqual(await update({ title: "renombrada" }), ["update", "tk-2", "--title", "renombrada"]);
		assert.equal(cli("show", "tk-2").parent_id, cli("show", "tk-1").id);
		assert.deepEqual(await update({ parent: "" }), ["update", "tk-2", "--parent", ""]);
		assert.equal(cli("show", "tk-2").parent_id, null);
	});

	it("tasks_update propaga el rechazo de un padre que no existe", async () => {
		cli("create", "--title", "una tarea");
		const before = cli("list");
		const rejected = spawnSync(binary, ["update", "tk-1", "--parent", "tk-999"], { cwd: project, encoding: "utf8" });
		assert.equal(rejected.status, 1);
		const message = JSON.parse(rejected.stderr || rejected.stdout).error;
		await assert.rejects(() => execute("tasks_update", { ref: "tk-1", parent: "tk-999" }),
			(error: Error) => error.message === message);
		assert.deepEqual(cli("list"), before);
	});

	it("tasks_update propaga el rechazo de ciclo del producto", async () => {
		cli("create", "--title", "padre");
		cli("create", "--title", "hija", "--parent", "tk-1");
		const before = cli("list");
		const rejected = spawnSync(binary, ["update", "tk-1", "--parent", "tk-2"], { cwd: project, encoding: "utf8" });
		assert.equal(rejected.status, 2);
		const message = JSON.parse(rejected.stderr || rejected.stdout).error;
		await assert.rejects(() => execute("tasks_update", { ref: "tk-1", parent: "tk-2" }),
			(error: Error) => error.message === message);
		assert.equal(cli("show", "tk-1").parent_id, null);
		assert.deepEqual(cli("list"), before);
	});

	it("tasks_block bloquea y tasks_unblock retira el bloqueo", async () => {
		cli("create", "--title", "dependiente");
		cli("create", "--title", "bloqueante");
		const blocked = await write("tasks_block", { ref: " TK-1 ", blocker: " TK-2 " });
		assert.equal(blocked.blocked, true);
		assert.equal(cli("show", "tk-1").blockers[0].ref, "tk-2");
		const unblocked = await write("tasks_unblock", { ref: " TK-1 ", blocker: " TK-2 " });
		assert.equal(unblocked.blocked, false);
		assert.deepEqual(cli("show", "tk-1").blockers, []);
	});

	it("tasks_comment añade el comentario sin volcarlo en el texto", async () => {
		cli("create", "--title", "una tarea");
		const result = await execute("tasks_comment", { ref: " TK-1 ", body: "comentario privado\nsegunda línea" });
		assert.deepEqual((cli("show", "tk-1") as Detail).comments.map((comment) => comment.body),
			["comentario privado\nsegunda línea"]);
		assert.deepEqual(result.details, cli("list")[0]);
		assert.match(result.content[0].text, /^tk-1 · .*\[open\]/);
		assert.doesNotMatch(result.content[0].text, /comentario privado|\n/);
	});

	it("tasks_comment_list lee los comentarios con su identidad y su fecha", async () => {
		cli("create", "--title", "una tarea");
		await execute("tasks_comment", { ref: "tk-1", body: "primero" });
		await execute("tasks_comment", { ref: "tk-1", body: "segundo" });

		const result = await execute("tasks_comment_list", { ref: " TK-1 " });
		const comments = result.details as { id: number; body: string; created_at: string }[];

		assert.equal(comments.length, 2);
		assert.deepEqual(comments.map((comment) => comment.id), [1, 2]);
		assert.deepEqual(comments.map((comment) => comment.body), ["primero", "segundo"]);
		assert.ok(comments.every((comment) => /^\d{4}-\d{2}-\d{2}T/.test(comment.created_at)));
		assert.match(result.content[0].text, /#1 · 20\d\d-.*primero/s);
		assert.match(result.content[0].text, /#2 · 20\d\d-.*segundo/s);
	});

	it("tasks_comment_list responde vacía y propaga el rechazo de una tarea sin referencia", async () => {
		cli("create", "--title", "una tarea");
		const empty = await execute("tasks_comment_list", { ref: "tk-1" });
		assert.equal((empty.details as unknown[]).length, 0);
		assert.match(empty.content[0].text, /Sin comentarios/);

		await assert.rejects(
			() => execute("tasks_comment_list", { ref: "tk-999" }),
			/there is no task with reference "tk-999"/,
		);
	});

	it("tasks_comment_edit cambia el cuerpo y conserva identidad y fecha", async () => {
		cli("create", "--title", "una tarea");
		await execute("tasks_comment", { ref: "tk-1", body: "versión original" });
		const before = (cli("show", "tk-1") as Detail).comments[0];

		const result = await execute("tasks_comment_edit", { ref: " TK-1 ", id: 1, body: "versión corregida" });
		const after = (cli("show", "tk-1") as Detail).comments[0];

		assert.equal(after.id, before.id);
		assert.equal(after.created_at, before.created_at);
		assert.equal(after.body, "versión corregida");
		assert.match(result.content[0].text, /^tk-1 · .*\[open\]/);
	});

	it("tasks_comment_edit propaga los rechazos del producto y no escribe nada", async () => {
		cli("create", "--title", "una tarea");
		cli("create", "--title", "otra");
		await execute("tasks_comment", { ref: "tk-2", body: "de la otra" });
		await execute("tasks_comment", { ref: "tk-1", body: "el mío" });
		const before = cli("show", "tk-1").comments;

		const rejected = spawnSync(binary, ["comment", "edit", "tk-1", "1", "--body", "nada"], { cwd: project, encoding: "utf8" });
		assert.equal(rejected.status, 1);
		const message = JSON.parse(rejected.stderr || rejected.stdout).error;
		await assert.rejects(() => execute("tasks_comment_edit", { ref: "tk-1", id: 1, body: "nada" }),
			(error: Error) => error.message === message);

		await assert.rejects(() => execute("tasks_comment_edit", { ref: "tk-1", id: 99, body: "nada" }),
			/there is no comment 99 in task "tk-1"/);
		await assert.rejects(() => execute("tasks_comment_edit", { ref: "tk-1", id: 0, body: "nada" }));

		assert.deepEqual(cli("show", "tk-1").comments, before);
		assert.deepEqual(cli("show", "tk-2").comments.map((comment: { body: string }) => comment.body), ["de la otra"]);
	});

	it("el detalle conserva la identidad y la fecha de los comentarios", async () => {
		cli("create", "--title", "una tarea");
		await execute("tasks_comment", { ref: "tk-1", body: "con identidad" });
		const shown = await execute("tasks_show", { ref: "tk-1" });
		const comment = (cli("show", "tk-1") as Detail).comments[0];

		assert.match(shown.content[0].text, new RegExp(`#${comment.id} · ${comment.created_at.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")} · con identidad`));
	});

	it("sin tablero instalado la herramienta falla con el mensaje del producto", async () => {
		const empty = mkdtempSync(path.join(tmpdir(), "tasks-ai-sin-tablero-"));
		try {
			await assert.rejects(
				() => tools.tasks_board.execute("id", {}, undefined, undefined, { cwd: empty }),
				/there is no project with the product installed/,
			);
		} finally {
			rmSync(empty, { recursive: true, force: true });
		}
	});
});
