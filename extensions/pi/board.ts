// El tablero que se le enseña al agente. Se deriva de la salida de `list` y se
// recorta: el contexto es el recurso escaso, así que aquí no viajan descripciones
// ni comentarios salvo cuando se pide el detalle a propósito (docs/contracts.md §7).

/** Lo que la extensión necesita de una tarea; el resto de la tarea se queda en el CLI. */
export interface Task {
	id: number;
	ref: string;
	title: string;
	status: string;
	blocked: boolean;
	priority: number;
	parent_id: number | null;
	tags: string[];
}

/** Una tarea con su descripción y lo que la rodea, tal como lo devuelve `show`. */
export interface Detail extends Task {
	description: string | null;
	children: Task[];
	blockers: Task[];
	dependents: Task[];
	comments: { body: string }[];
}

export interface TreeNode {
	ref: string;
	title: string;
	status: string;
	blocked: boolean;
	children: TreeNode[];
}

export interface Board {
	counts: { open: number; in_progress: number; done: number; total: number };
	inProgress: Task[];
	ready: Task[];
	tree: TreeNode[];
}

const MAX_TREE_NODES = 40;
const MAX_LISTED = 8;
const MAX_TITLE = 60;
const MAX_COMMENT = 500;

/** Deriva el tablero: contadores por estado, lo que está en curso, lo que se puede empezar y el árbol. */
export function deriveBoard(tasks: Task[]): Board {
	const counts = { open: 0, in_progress: 0, done: 0, total: tasks.length };
	for (const task of tasks) {
		if (task.status === "open") counts.open++;
		else if (task.status === "in_progress") counts.in_progress++;
		else if (task.status === "done") counts.done++;
	}

	const byId = new Map(tasks.map((task) => [task.id, task]));
	const childrenOf = new Map<number, Task[]>();
	const roots: Task[] = [];

	for (const task of tasks) {
		const parent = task.parent_id === null ? undefined : byId.get(task.parent_id);
		if (!parent) {
			roots.push(task);
			continue;
		}
		const siblings = childrenOf.get(parent.id) ?? [];
		siblings.push(task);
		childrenOf.set(parent.id, siblings);
	}

	const node = (task: Task): TreeNode => ({
		ref: task.ref,
		title: task.title,
		status: task.status,
		blocked: task.blocked,
		children: (childrenOf.get(task.id) ?? []).map(node),
	});

	return {
		counts,
		inProgress: tasks.filter((task) => task.status === "in_progress"),
		ready: tasks.filter((task) => task.status === "open" && !task.blocked),
		tree: roots.map(node),
	};
}

/** Compone el texto que lee el modelo: contadores, dos listas cortas y el árbol recortado. */
export function renderBoard(board: Board): string {
	const lines = [
		`tareas: ${board.counts.total} (abiertas ${board.counts.open}, en curso ${board.counts.in_progress}, cerradas ${board.counts.done})`,
		`en curso: ${listOf(board.inProgress)}`,
		`listas para empezar: ${listOf(board.ready)}`,
	];

	const flat = flatten(board.tree);
	if (flat.length > 0) {
		lines.push("árbol:");
		for (const [depth, node] of flat.slice(0, MAX_TREE_NODES)) {
			lines.push(`${"  ".repeat(depth)}- ${nodeLine(node)}`);
		}
		const hidden = flat.length - Math.min(flat.length, MAX_TREE_NODES);
		if (hidden > 0) lines.push(`… ${hidden} tareas no mostradas: pídelas con tasks_show`);
	}

	return lines.join("\n");
}

/** Contexto breve de cada turno, sin el árbol ni campos privados. */
export function renderSummary(board: Board): string {
	const listed = (tasks: Task[], limit: number) => {
		const shown = tasks.slice(0, limit).map((task) => `${task.ref} · ${clip(task.title, MAX_TITLE)}`);
		const hidden = tasks.length - shown.length;
		return `${shown.join(", ") || "ninguna"}${hidden > 0 ? ` (+${hidden} más; usa tasks_board)` : ""}`;
	};
	return [
		`tareas: ${board.counts.total} (abiertas ${board.counts.open}, en curso ${board.counts.in_progress}, cerradas ${board.counts.done})`,
		`en curso: ${listed(board.inProgress, MAX_LISTED)}`,
		`listas para empezar: ${listed(board.ready, 5)}`,
		"Usa el tablero como único sistema de tareas: crear o seleccionar una tarea antes de tocar archivos y marcarla en curso.",
		"Anota decisiones y bloqueos como comentarios; cerrar solo tras verificar.",
	].join("\n");
}

/** Compone el texto del detalle de una tarea, con sus comentarios más recientes. */
export function renderDetail(detail: Detail): string {
	const lines = [
		`${detail.ref} · ${detail.title}`,
		`estado: ${detail.status}${detail.blocked ? " (bloqueada)" : ""} | prioridad: ${detail.priority} | etiquetas: ${detail.tags.join(", ") || "ninguna"}`,
	];

	if (detail.description) lines.push("", detail.description);
	if (detail.children.length > 0) lines.push("", `hijas (${detail.children.length}): ${listOf(detail.children)}`);
	if (detail.blockers.length > 0) lines.push(`bloqueada por: ${listOf(detail.blockers)}`);
	if (detail.dependents.length > 0) lines.push(`bloquea a: ${listOf(detail.dependents)}`);

	if (detail.comments.length > 0) {
		const recent = detail.comments.slice(-10);
		lines.push("", `comentarios (${detail.comments.length}, los ${recent.length} últimos):`);
		for (const comment of recent) lines.push(`- ${clip(comment.body, MAX_COMMENT)}`);
	}

	return lines.join("\n");
}

function listOf(tasks: Task[]): string {
	if (tasks.length === 0) return "ninguna";
	const shown = tasks.slice(0, MAX_LISTED).map((task) => `${task.ref} · ${task.title}`);
	const hidden = tasks.length - shown.length;
	return hidden > 0 ? `${shown.join(", ")} (+${hidden} más)` : shown.join(", ");
}

function nodeLine(node: TreeNode): string {
	const marks = [node.status === "done" ? "hecha" : node.status.replace("_", " ")];
	if (node.blocked) marks.push("bloqueada");
	return `${node.ref} · ${node.title} [${marks.join(", ")}]`;
}

function flatten(nodes: TreeNode[], depth = 0, out: [number, TreeNode][] = []): [number, TreeNode][] {
	for (const node of nodes) {
		out.push([depth, node]);
		flatten(node.children, depth + 1, out);
	}
	return out;
}

function clip(text: string, limit: number): string {
	const oneLine = text.replace(/\s+/g, " ").trim();
	return oneLine.length > limit ? `${oneLine.slice(0, limit - 1)}…` : oneLine;
}
