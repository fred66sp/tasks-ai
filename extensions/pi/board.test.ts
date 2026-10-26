import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { deriveBoard, renderBoard, renderComments, renderDetail, type Task } from "./board.ts";

function task(over: Partial<Task> & { id: number; ref: string }): Task {
	return {
		title: `tarea ${over.ref}`,
		status: "open",
		blocked: false,
		priority: 2,
		parent_id: null,
		tags: [],
		updated_at: "2026-01-01T00:00:00Z",
		...over,
	};
}

describe("deriveBoard", () => {
	it("cuenta por estado y separa lo que está en curso de lo que se puede empezar", () => {
		const tasks = [
			task({ id: 1, ref: "tk-1" }),
			task({ id: 2, ref: "tk-2", parent_id: 1, blocked: true }),
			task({ id: 3, ref: "tk-3", status: "in_progress" }),
			task({ id: 4, ref: "tk-4", status: "done" }),
		];

		const board = deriveBoard(tasks);

		assert.deepEqual(board.counts, { open: 2, in_progress: 1, done: 1, total: 4 });
		assert.deepEqual(board.inProgress.map((t) => t.ref), ["tk-3"]);
		// la bloqueada no se puede empezar, y la que está en curso ya se empezó
		assert.deepEqual(board.ready.map((t) => t.ref), ["tk-1"]);
	});

	it("elige como última cerrada la del momento de cierre más reciente", () => {
		const tasks = [
			task({ id: 1, ref: "tk-1", status: "done", updated_at: "2026-10-20T10:00:00Z" }),
			task({ id: 2, ref: "tk-2", status: "done", updated_at: "2026-10-21T09:00:00Z" }),
			task({ id: 3, ref: "tk-3", status: "in_progress", updated_at: "2026-10-22T09:00:00Z" }),
		];

		// la que está en curso tiene el momento más reciente y aun así no es una tarea cerrada
		assert.equal(deriveBoard(tasks).lastClosed?.ref, "tk-2");
	});

	it("a igualdad de momento deja la de número interno mayor", () => {
		const tasks = [
			task({ id: 7, ref: "tk-7", status: "done", updated_at: "2026-10-22T09:00:00Z" }),
			task({ id: 9, ref: "tk-9", status: "done", updated_at: "2026-10-22T09:00:00Z" }),
		];

		assert.equal(deriveBoard(tasks).lastClosed?.ref, "tk-9");
	});

	it("anida por padre y deja como raíz la hija cuyo padre no está en la lista", () => {
		const tasks = [
			task({ id: 1, ref: "tk-1" }),
			task({ id: 2, ref: "tk-2", parent_id: 1 }),
			task({ id: 9, ref: "tk-9", parent_id: 404 }),
		];

		const board = deriveBoard(tasks);

		assert.equal(board.tree.length, 2);
		assert.equal(board.tree[0].ref, "tk-1");
		assert.deepEqual(board.tree[0].children.map((c) => c.ref), ["tk-2"]);
		// una tarea nunca se pierde por tener un padre que no está
		assert.equal(board.tree[1].ref, "tk-9");
	});

	it("con el tablero vacío devuelve contadores a cero y listas vacías", () => {
		assert.deepEqual(deriveBoard([]), {
			counts: { open: 0, in_progress: 0, done: 0, total: 0 },
			inProgress: [],
			ready: [],
			lastClosed: undefined,
			tree: [],
		});
	});
});

describe("renderBoard", () => {
	it("recorta el árbol y dice cuánto ha dejado fuera", () => {
		const tasks = Array.from({ length: 45 }, (_, index) =>
			task({ id: index + 1, ref: `tk-${index + 1}` }),
		);

		const text = renderBoard(deriveBoard(tasks));
		const lines = text.split("\n").filter((line) => line.startsWith("- ") || line.startsWith("  - "));

		assert.equal(lines.length, 40);
		assert.match(text, /5 tareas no mostradas/);
	});

	it("no arrastra descripciones ni comentarios", () => {
		const tasks = [
			task({ id: 1, ref: "tk-1", title: "una tarea" }),
			task({ id: 2, ref: "tk-2", status: "in_progress" }),
		];

		const text = renderBoard(deriveBoard(tasks));

		assert.match(text, /tk-1 · una tarea/);
		assert.match(text, /en curso: tk-2/);
	});
});

describe("renderDetail", () => {
	it("trae la descripción entera y solo los 10 comentarios más recientes, con identidad y fecha", () => {
		const long = `${"a".repeat(700)}\n\n## Sección\n${"b".repeat(600)}`;
		const comments = Array.from({ length: 26 }, (_, index) => ({
			id: index + 1,
			body: index === 16 ? long : `comentario ${index + 1}`,
			created_at: `2026-10-18T12:0${index % 10}:00Z`,
		}));

		const text = renderDetail({
			...task({ id: 1, ref: "tk-1", priority: 1, tags: ["docs"] }),
			description: "una descripción larga",
			children: [task({ id: 2, ref: "tk-2", title: "hija", priority: 2 })],
			blockers: [],
			dependents: [],
			comments,
		});

		assert.match(text, /una descripción larga/);
		assert.match(text, /comentario 18/);
		assert.doesNotMatch(text, /comentario 16\n/);
		assert.match(text, /comentarios \(26, los 10 últimos\)/);
		assert.match(text, /tk-2 · hija/);
		assert.match(text, /#17 · 2026-10-18T12:06:00Z\n/);
		assert.ok(text.includes(long), "el detalle tampoco recorta ni recompone el cuerpo");
		assert.doesNotMatch(text, /#16 ·/);
	});

	it("renderComments lista los comentarios con su identidad y devuelve el cuerpo entero", () => {
		assert.equal(renderComments([]), "Sin comentarios.");

		const long = `${"a".repeat(700)}\n\n## Sección\n${"b".repeat(600)}`;
		const text = renderComments([
			{ id: 1, body: "primero", created_at: "2026-10-18T12:00:00Z" },
			{ id: 2, body: long, created_at: "2026-10-18T12:01:00Z" },
		]);

		assert.match(text, /#1 · 2026-10-18T12:00:00Z\nprimero/);
		assert.ok(text.includes(long), "el cuerpo largo se devuelve sin recortar ni recomponer");
	});
});
