# ADR-014: La extensión se distribuye incrustada y el install la copia

- **Estado:** Aceptada
- **Fecha:** 2026-10-22

## Contexto

ADR-012 dejó la extensión del agente fuera del binario y fuera del producto instalable:
el proyecto la instalaba por ruta (`pi install -l ./extensions/pi`), dejando en sus
ajustes una ruta relativa al repositorio de origen. Un proyecto que se lleva a otra
máquina sin ese repositorio pierde la integración, y el binario tampoco viaja: se
recompilaba en su repositorio y se enlazaba.

El install debe dejar el proyecto autocontenido: el binario y la extensión dentro de
`.tasks-ai/`, sin depender de la ruta del repositorio de origen.

## Decisión

- La extensión se **incrusta en el binario** con `go:embed` —solo los ficheros que
  ejecuta: `index.ts`, `board.ts`, `cli.ts` y `package.json`— para que el install
  pueda copiarla sin conocer el repositorio de origen.
- `init`, además de la carpeta con la base, los ajustes y las instrucciones, copia el
  binario a `.tasks-ai/bin/tasks-ai` y extrae la extensión a `.tasks-ai/extensions/pi/`.
- La extensión instalada la carga el proyecto con `pi install -l
  .tasks-ai/extensions/pi`. `typebox` es dependencia par que aporta el host (Pi), así
  que la copia no lleva `node_modules`.

## Consecuencias

- El proyecto instalado es autocontenido: moverlo de máquina no rompe el binario ni la
  extensión.
- La extensión sigue siendo TypeScript y fuera del núcleo: incrustarla es solo el canal
  de distribución, no una reimplementación en Go.
- ADR-012 queda superada en su punto «fuera del binario / no entra en el producto
  instalable»; el resto —envolver el CLI, no tocar SQLite, una herramienta por
  operación— sigue en pie.
- Requisitos afectados: RF-03 (la carpeta guarda también el binario y la extensión) y
  RF-11 (la extensión se incrusta para distribuirla).
