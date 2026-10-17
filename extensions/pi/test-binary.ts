// Las suites necesitan el binario del producto, y en un clon limpio no está: `bin/`
// no se versiona. Lo construyen una vez y solo si falta, para que `npm test` funcione
// sin pasos previos.

import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";

/** La raíz del repositorio, que es la que tiene el `go.mod`. */
export const repoRoot = path.resolve(import.meta.dirname, "../..");

/** El binario que las suites ejecutan: el mismo que usa el producto desde el repositorio. */
export const binary = path.join(repoRoot, "bin", "tasks-ai");

/** Devuelve el binario, construyéndolo si no está. */
export function ensureBinary(): string {
	if (!existsSync(binary)) {
		execFileSync("go", ["build", "-o", "bin/tasks-ai", "./cmd/tasks-ai"], {
			cwd: repoRoot,
			stdio: "pipe",
		});
	}

	return binary;
}
