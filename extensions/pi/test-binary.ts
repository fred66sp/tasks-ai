// Las suites necesitan el binario del producto, y en un clon limpio no está: `bin/`
// no se versiona. Lo construyen una vez y solo si falta, para que `npm test` funcione
// sin pasos previos.

import { execFileSync } from "node:child_process";
import { renameSync } from "node:fs";
import path from "node:path";

/** La raíz del repositorio, que es la que tiene el `go.mod`. */
export const repoRoot = path.resolve(import.meta.dirname, "../..");

/** El binario que las suites ejecutan: el mismo que usa el producto desde el repositorio. */
export const binary = path.join(repoRoot, "bin", "tasks-ai");

/**
 * Devuelve el binario, construyéndolo siempre: `bin/` no se versiona, y uno viejo
 * deja que la suite mida el código de ayer sin avisar. La construcción es
 * incremental y cuesta menos que una prueba.
 */
export function ensureBinary(): string {
	// Se construye aparte y se renombra: el renombrado es atómico, así que otro
	// proceso que ejecute el binario nunca lee un fichero a medio escribir, aunque
	// dos ficheros de prueba lo construyan a la vez.
	const fresh = `${binary}.${process.pid}.tmp`;

	execFileSync("go", ["build", "-o", fresh, "./cmd/tasks-ai"], {
		cwd: repoRoot,
		stdio: "pipe",
	});
	renameSync(fresh, binary);

	return binary;
}
