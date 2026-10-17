# tasks-ai — reglas del repositorio

Reglas que valen en cada turno dentro de este repositorio. Esto es un router: el
detalle canónico vive en los documentos enlazados y no se copia aquí.

## Contratos

- [`docs/CONTRATO_DOCUMENTACION.md`](docs/CONTRATO_DOCUMENTACION.md): qué escribe cada
  documento y dónde vive cada dato.
- [`docs/CONTRATO_FLUJO_GIT_VERSIONADO.md`](docs/CONTRATO_FLUJO_GIT_VERSIONADO.md):
  ramas, commits, verificación, integración, CHANGELOG y versionado.

## Tablero del trabajo

El trabajo se registra en el tablero del propio producto, instalado en `.task-ai/`.
Toda llamada va por el binario del producto y responde en JSON:

```bash
./bin/tasks-ai list              # todas las tareas, con su estado y si están bloqueadas
./bin/tasks-ai show tk-1         # la tarea con sus hijas, sus bloqueos y sus comentarios
./bin/tasks-ai tags              # el catálogo de etiquetas y su uso
```

- La base de tareas vive en `.task-ai/tasks.db`. No se edita con SQL: solo por el CLI
  del producto.
- Ciclo: crear o seleccionar la tarea antes de tocar archivos, marcarla en curso,
  mantener las notas y los comentarios al día y cerrarla tras verificar.
- Lo que no está en el tablero no entra en el trabajo. La tarea siguiente se saca de
  ahí, con su criterio; no se inventa por el camino. Lo que aparezca de nuevo —un
  arreglo, una decisión, una duda— se anota como tarea antes de tocarlo.
- Cómo se escribe y se mantiene una tarea —etiquetas, ritmo de actualización y relevo—
  lo gobierna [`.task-ai/AGENTS.md`](.task-ai/AGENTS.md).
- Cada tarea declara **el resultado que cierra** y **el criterio con el que se
  comprueba**, observable por la interfaz pública. Una tarea sin criterio no se
  empieza: se escribe primero.
- Cuando el tablero o sus instrucciones no digan algo que haga falta, se corrige ahí en
  el momento: si algo cuesta entenderlo, el hueco es de la documentación, no de quien lo
  sufre.
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
  paralelo si tocan los mismos archivos, se encadenan. Tampoco se entrega un cambio
  mientras una verificación independiente recorre el árbol: se espera su veredicto, o
  la verificación acabará describiendo algo distinto de lo entregado.

## Qué se toca y qué no

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto |
| `cmd/`, `internal/` | El código del producto, en Go |
| `.task-ai/` | El tablero del proyecto; la base no se versiona |

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
