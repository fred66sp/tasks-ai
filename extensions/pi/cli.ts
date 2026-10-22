// Envuelve el CLI del producto: lo ejecuta, lee su JSON y traduce el código de
// salida. Nada más: las reglas y los mensajes son del producto, y aquí no se
// reescriben (ADR-012).

import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";
import { promisify } from "node:util";

const execute = promisify(execFile);

/** La respuesta del producto, o el motivo por el que no la hubo. */
export type Outcome = { ok: true; data: unknown } | { ok: false; code: number; message: string };

/** El proyecto es el primer directorio, subiendo, que tiene la carpeta del producto. */
export function findProjectRoot(from: string): string | null {
	let current = path.resolve(from);
	for (;;) {
		if (existsSync(path.join(current, ".tasks-ai"))) return current;
		const parent = path.dirname(current);
		if (parent === current) return null;
		current = parent;
	}
}

/**
 * Dónde está el binario: `TASKS_AI_BIN` si se ha dicho, si no el que el proyecto
 * tiene construido, y si no el que esté en el PATH.
 */
export function resolveBin(cwd: string): string {
	if (process.env.TASKS_AI_BIN) return process.env.TASKS_AI_BIN;

	const root = findProjectRoot(cwd);
	if (root !== null) {
		const installed = path.join(root, ".tasks-ai", "bin", "tasks-ai");
		if (existsSync(installed)) return installed;

		const local = path.join(root, "bin", "tasks-ai");
		if (existsSync(local)) return local;
	}

	return "tasks-ai";
}

/** Ejecuta el producto con los argumentos dados y devuelve su respuesta o su rechazo. */
export async function runCli(args: string[], cwd: string, signal?: AbortSignal): Promise<Outcome> {
	try {
		const { stdout } = await execute(resolveBin(cwd), args, {
			cwd,
			signal,
			maxBuffer: 16 * 1024 * 1024,
		});
		return { ok: true, data: parse(stdout) };
	} catch (failure) {
		const failed = failure as { code?: number | string; stdout?: string; stderr?: string; message?: string };

		// Sin código numérico el proceso no llegó a arrancar: no hay respuesta que leer.
		if (typeof failed.code !== "number") {
			return { ok: false, code: -1, message: failed.message ?? `could not run ${resolveBin(cwd)}` };
		}

		return { ok: false, code: failed.code, message: messageOf(failed) };
	}
}

function parse(raw: string): unknown {
	const text = raw.trim();
	return text === "" ? null : JSON.parse(text);
}

/** El mensaje del producto, tal cual: primero el cuerpo JSON de error, si no lo que haya. */
function messageOf(failed: { stdout?: string; stderr?: string; message?: string }): string {
	for (const stream of [failed.stderr, failed.stdout]) {
		const text = stream?.trim();
		if (!text) continue;
		try {
			const body = JSON.parse(text) as { error?: string };
			if (body.error) return body.error;
		} catch {
			// no es JSON: se usa el texto tal cual
		}
		return text;
	}

	return failed.message ?? "the product did not answer";
}
