import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { after, before, describe, it } from "node:test";
import { runCli } from "./cli.ts";

const binary = path.resolve(import.meta.dirname, "../../bin/tasks-ai");

describe("runCli", () => {
	let project: string;

	before(() => {
		process.env.TASKS_AI_BIN = binary;
		project = mkdtempSync(path.join(tmpdir(), "tasks-ai-cli-"));
	});

	after(() => {
		rmSync(project, { recursive: true, force: true });
	});

	it("sin tablero instalado responde con error y no lanza", async () => {
		const outcome = await runCli(["list"], project);

		assert.equal(outcome.ok, false);
		if (outcome.ok) return;
		assert.equal(outcome.code, 1);
		assert.match(outcome.message, /there is no project with the product installed/);
	});

	it("con el tablero instalado devuelve las tareas y distingue el fallo del mal uso", async () => {
		const init = spawnSync(binary, ["init"], { cwd: project, encoding: "utf8" });
		assert.equal(init.status, 0, init.stderr);

		const empty = await runCli(["list"], project);
		assert.equal(empty.ok, true);
		if (!empty.ok) return;
		assert.deepEqual(empty.data, []);

		const created = await runCli(["create", "--title", "una tarea"], project);
		assert.equal(created.ok, true);
		const ref = (created.ok && (created.data as { ref: string }).ref) || "";

		// la regla del núcleo: código 1, con su mensaje tal cual
		const missing = await runCli(["show", "tk-999"], project);
		assert.equal(missing.ok, false);
		if (!missing.ok) {
			assert.equal(missing.code, 1);
			assert.match(missing.message, /there is no task with reference "tk-999"/);
		}

		// el mal uso: código 2
		const misuse = await runCli(["create"], project);
		assert.equal(misuse.ok, false);
		if (!misuse.ok) {
			assert.equal(misuse.code, 2);
			assert.match(misuse.message, /the task title is missing/);
		}

		const shown = await runCli(["show", ref], project);
		assert.equal(shown.ok, true);
		if (shown.ok) assert.equal((shown.data as { ref: string }).ref, ref);
	});
});
