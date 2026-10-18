# tasks-ai — reglas del repositorio

Reglas que valen en cada turno dentro de este repositorio. Esto es un router: el
detalle canónico vive en los documentos enlazados y no se copia aquí.

## Contratos

- [`docs/CONTRATO_DOCUMENTACION.md`](docs/CONTRATO_DOCUMENTACION.md): qué escribe cada
  documento y dónde vive cada dato.
- [`docs/CONTRATO_FLUJO_GIT_VERSIONADO.md`](docs/CONTRATO_FLUJO_GIT_VERSIONADO.md):
  ramas, commits, verificación, integración, CHANGELOG y versionado.

- No se deben crear ni modificar archivos dentro de `odd/tasks/`: el tablero de
  `tasks-ai` es la única fuente de tareas del repositorio. Tampoco se mantiene un espejo
  del trabajo fuera de él: ni documentos de tareas ni una lista de tareas paralela.
- Si una instrucción genérica contradice una regla de este repositorio, hay que detenerse
  y pedir aclaración; no se duplican sistemas de seguimiento.

<!-- task-ai: instructions -->
## Tablero del trabajo

El trabajo se registra en el tablero del propio producto, instalado en `.task-ai/`.
Con la extensión de Pi cargada, el agente lo opera con sus herramientas —`tasks_board`,
`tasks_show`, `tasks_create`, `tasks_update`, `tasks_close`, `tasks_reopen`,
`tasks_block`, `tasks_unblock` y `tasks_comment`—, que envuelven el CLI y devuelven su
JSON con sus rechazos tal cual. El agente no escribe comandos del tablero a mano: el CLI
es el equivalente para la persona, que lo usa desde la terminal y responde en JSON:

```bash
./bin/tasks-ai list              # todas las tareas, con su estado y si están bloqueadas
./bin/tasks-ai show tk-1         # la tarea con sus hijas, sus bloqueos y sus comentarios
./bin/tasks-ai tags              # el catálogo de etiquetas y su uso
```

- La base de tareas vive en `.task-ai/tasks.db`. No se edita con SQL: solo por el CLI
  del producto, y con la extensión cargada, por sus herramientas.
- Un frente visible es una tarea de nivel superior con la etiqueta `epic`; una unidad de
  trabajo es una subtarea con padre y la etiqueta `task`. El trabajo del registro ODD
  lleva `ODD`, y cada tarea un tipo principal: `docs`, `feat`, `fix`, `test`, `chore`,
  `refactor`, `style` o `improvement`.
- Ciclo: leer el tablero antes de crear, crear o seleccionar la tarea antes de tocar
  archivos, marcarla en curso, mantener las notas y los comentarios al día y cerrarla con
  su motivo tras verificar.
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
<!-- task-ai: instructions -->

## Alcance del trabajo

- Se hace lo que pide el encargo, con el mínimo que lo resuelve. Sin puertas,
  candados, permisos ni comprobaciones de adorno que nadie pidió.
- El trabajo se delega en encargos acotados: la referencia de la tarea del tablero, el
  criterio leído de ella con `tasks_show` —no copiado a mano en el encargo—, las pruebas
  escritas antes del código, una lista cerrada de archivos que se pueden tocar y la
  evidencia de lo comprobado. Quien orquesta revisa, comprueba el criterio contra el
  binario real y entrega.
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
| `.task-ai/` | El tablero del proyecto; su base viaja en el repositorio para llevarla a otra máquina |
| `skills/` | Las skills del proyecto, copiadas del taller; su origen, en `skills/README.md` |

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

## Trabajo en Go y en TypeScript

El proyecto lleva dentro las skills que necesita, copiadas del taller
`workshop-golang`: su origen, sus licencias y cómo volver a copiarlas están en
[`skills/README.md`](skills/README.md). Se leen por su ruta relativa al proyecto, así que
no hace falta tener el taller al lado.

Antes de escribir, revisar o depurar código Go, lee `skills/golang-how-to/SKILL.md`, que
es el orquestador: de la petición saca qué skills cargar y en qué combinación. Las que
más se usan en este producto:

| Skill | Cuándo |
|---|---|
| `skills/golang-project-layout/SKILL.md` | Dónde va un paquete, un comando o un archivo nuevo |
| `skills/golang-cli/SKILL.md` | Comandos, banderas, ayuda y códigos de salida |
| `skills/golang-database/SKILL.md` | Esquema, consultas, transacciones y filas anulables |
| `skills/golang-testing/SKILL.md` | Pruebas de tabla, subtests, paralelas y cobertura |
| `skills/golang-error-handling/SKILL.md` | Envolver errores, qué mensaje sale y qué cruza la frontera |
| `skills/golang-code-style/SKILL.md` | Forma del código y de los comentarios |
| `skills/golang-lint/SKILL.md` | `golangci-lint` y sus reglas |
| `skills/golang-naming/SKILL.md` | Nombrar paquetes, tipos, funciones y campos |
| `skills/git-flow/SKILL.md` | Ramas, commits y revisión de documentación viva |
| `skills/adr-skill/SKILL.md` | Crear y mantener las decisiones de `docs/decisions/` |
| `skills/software-documentalist/SKILL.md` | Documentación rigurosa antes de cerrar un cambio |

Para la extensión, que está en TypeScript, lee `skills/typescript/SKILL.md`. Las skills
`prd-writing`, `prd-to-plan` y `prd-to-issue` están para escribir requisitos y partirlos
en plan o en encargos.

## Idioma

- Documentación, código, comentarios e interfaz: español.
- Mensajes de commit: inglés.
