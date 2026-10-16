# tasks-ai — reglas del repositorio

Reglas que valen en cada turno dentro de este repositorio. Esto es un router: el
detalle canónico vive en los documentos enlazados y no se copia aquí.

## Contratos

- [`docs/CONTRATO_DOCUMENTACION.md`](docs/CONTRATO_DOCUMENTACION.md): qué escribe cada
  documento y dónde vive cada dato.
- [`docs/CONTRATO_FLUJO_GIT_VERSIONADO.md`](docs/CONTRATO_FLUJO_GIT_VERSIONADO.md):
  ramas, commits, verificación, integración, CHANGELOG y versionado.

## Tablero del trabajo

El trabajo se registra en la herramienta de desarrollo, instalada en `task-manager/`.
Toda llamada pasa por su guardia, con base explícita y salida JSON:

```bash
cd task-manager
npm run odd:tk -- stats
npm run odd:tk -- ready --limit 1
npm run odd:tk -- show tk-XXXX
```

- La base de tareas vive en `task-manager/tk/.tacks/tacks.db`. No se edita con SQL:
  solo por su CLI.
- Ciclo: crear o seleccionar la tarea antes de tocar archivos, marcarla en curso,
  mantener las notas y los comentarios al día y cerrarla tras verificar.
- Lo que no está en el tablero no entra en el trabajo. La tarea siguiente se saca de
  ahí, con su criterio; no se inventa por el camino. Lo que aparezca de nuevo —un
  arreglo, una decisión, una duda— se anota como tarea antes de tocarlo.
- Cómo se escribe y se mantiene una tarea —etiquetas, ritmo de actualización, relevo
  y trampas conocidas del tablero— lo gobierna
  [`task-manager/skills/tk/SKILL.md`](task-manager/skills/tk/SKILL.md).
- Además de lo que dice esa skill, cada tarea declara **el resultado que cierra** y **el
  criterio con el que se comprueba**, observable por la interfaz pública. Una tarea sin
  criterio no se empieza: se escribe primero.
- No se registra en el tablero el estado de Git ni el del proceso de revisión: cada
  uno vive donde vive.

## Alcance del trabajo

- Se hace lo que pide el encargo, con el mínimo que lo resuelve. Sin puertas,
  candados, permisos ni comprobaciones de adorno que nadie pidió.
- El trabajo se delega en encargos acotados: el criterio citado literalmente, las
  pruebas escritas antes del código, una lista cerrada de archivos que se pueden
  tocar, y la evidencia de lo comprobado. Quien orquesta revisa, comprueba el
  criterio contra el binario real y entrega.
- Una tarea se cierra cuando su resultado y sus comprobaciones se han observado. Lo
  que se comprueba es el criterio, no la impresión de quien hizo el trabajo.
- Una unidad a la vez sobre el mismo código: dos tareas listas no se trabajan en
  paralelo si tocan los mismos archivos, se encadenan.

## Qué se toca y qué no

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto |
| `task-manager/` | La herramienta con la que se desarrolla; no es código del producto |
| `cmd/`, `internal/` | El código del producto, en Go |

- Los proyectos de referencia están fuera de este repositorio: se leen, no se tocan.
- No se traduce la referencia funcional ni se copia su código.

## Al cambiar algo

| Cambio | Dónde se escribe |
|---|---|
| Requisito nuevo o modificado | `docs/PRD.md` |
| Decisión durable | Un documento nuevo en `docs/decisions/` |
| Forma de datos o de API | `docs/contracts.md` |
| Diseño | `docs/architecture.md` |
| Riesgo nuevo | `docs/known-risks.md` |
| Comandos o estructura | `README.md` o `docs/development.md` |

Y siempre, antes de cerrar el cambio, las comprobaciones del contrato de
documentación.

## Trabajo en Go

Antes de escribir, revisar o depurar código Go, carga las skills del taller para esa
petición:

```bash
../../.strm/bin/strm -manifest ../../.strm/skills-manifest.json -prompt "<la petición>"
```

Después lee `../../skills/golang-how-to/SKILL.md`, que es el orquestador, y las skills
que señale para la tarea. Para este producto suelen ser `golang-project-layout`,
`golang-cli`, `golang-database`, `golang-testing`, `golang-error-handling`,
`golang-code-style`, `golang-lint` y `golang-naming`.

## Idioma

- Documentación, código, comentarios e interfaz: español.
- Mensajes de commit: inglés.
