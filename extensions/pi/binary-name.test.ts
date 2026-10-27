import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { binaryFileName } from "./cli.ts";

// La copia que deja `init` y el fichero que busca el adaptador tienen que
// llamarse igual: en Windows, sin `.exe`, el sistema no ejecuta la copia.
describe("binaryFileName", () => {
	it("en Windows lleva la extensión que el sistema necesita para ejecutarlo", () => {
		assert.equal(binaryFileName("win32"), "tasks-ai.exe");
	});

	it("en las demás plataformas deja el nombre sin extensión", () => {
		assert.equal(binaryFileName("linux"), "tasks-ai");
		assert.equal(binaryFileName("darwin"), "tasks-ai");
	});
});
