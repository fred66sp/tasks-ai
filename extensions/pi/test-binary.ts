// Las suites necesitan el binario del producto y no pueden usar el que el tablero tiene
// instalado: uno viejo deja que la suite mida el código de ayer sin avisar. Lo construyen
// en un directorio temporal, así que `npm test` funciona sin pasos previos y sin tocar
// las copias del proyecto.

import { execFileSync } from "node:child_process";
import { mkdirSync, renameSync } from "node:fs";
import os from "node:os";
import path from "node:path";

/** La raíz del repositorio, que es la que tiene el `go.mod`. */
export const repoRoot = path.resolve(import.meta.dirname, "../..");

/** El binario que las suites ejecutan: fuera del repositorio, para no confundirlo con el instalado. */
export const binary = path.join(os.tmpdir(), "tasks-ai-test", "tasks-ai");

/**
 * Devuelve el binario, construyéndolo siempre: la construcción es incremental y cuesta
 * menos que una prueba.
 */
export function ensureBinary(): string {
	mkdirSync(path.dirname(binary), { recursive: true });

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
