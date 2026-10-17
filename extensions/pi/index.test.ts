import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { after, before, describe, it } from "node:test";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

import tasksExtension from "./index.ts";

const binary = path.resolve(import.meta.dirname, "../../bin/tasks-ai");

interface Registered {
	name: string;
	execute: (
		id: string,
		params: Record<string, unknown>,
		signal: undefined,
		onUpdate: undefined,
		ctx: { cwd: string },
	) => Promise<{ content: { type: string; text: string }[]; details: unknown }>;
}

function load(): Record<string, Registered> {
	const registered: Record<string, Registered> = {};
	const pi = {
		registerTool(tool: Registered) {
			registered[tool.name] = tool;
		},
	} as unknown as ExtensionAPI;

	tasksExtension(pi);

	return registered;
}

describe("la extensión", () => {
	let project: string;
	let tools: Record<string, Registered>;

	before(() => {
		process.env.TASKS_AI_BIN = binary;
		project = mkdtempSync(path.join(tmpdir(), "tasks-ai-extension-"));
		const init = spawnSync(binary, ["init"], { cwd: project, encoding: "utf8" });
		assert.equal(init.status, 0, init.stderr);
		tools = load();
	});

	after(() => {
		rmSync(project, { recursive: true, force: true });
	});

	it("registra las dos herramientas de lectura", () => {
		assert.deepEqual(Object.keys(tools).sort(), ["tasks_board", "tasks_show"]);
	});

	it("tasks_board devuelve el tablero del proyecto", async () => {
		spawnSync(binary, ["create", "--title", "una tarea"], { cwd: project, encoding: "utf8" });

		const result = await tools.tasks_board.execute("id", {}, undefined, undefined, { cwd: project });

		assert.match(result.content[0].text, /tareas: 1 \(abiertas 1/);
		assert.match(result.content[0].text, /tk-1 · una tarea/);
	});

	it("tasks_show devuelve la tarea y rechaza una referencia que no existe", async () => {
		const shown = await tools.tasks_show.execute("id", { ref: "tk-1" }, undefined, undefined, { cwd: project });
		assert.match(shown.content[0].text, /^tk-1 · una tarea/);

		await assert.rejects(
			() => tools.tasks_show.execute("id", { ref: "tk-999" }, undefined, undefined, { cwd: project }),
			/there is no task with reference "tk-999"/,
		);
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
