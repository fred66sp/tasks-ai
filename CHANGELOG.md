# Registro de cambios

Cambios relevantes del proyecto. El formato sigue las recomendaciones de *Keep a
Changelog* y el versionado es semántico.

## [Sin publicar]

### Añadido

- **Primera rebanada del producto**: se instala en la carpeta del proyecto, guarda su
  estado en una base SQLite con esquema versionado, y crea y lee tareas desde la línea
  de comandos, en JSON.
- **Cambio de una tarea**: la línea de comandos cambia el título, la descripción, las
  notas, el responsable y la prioridad de una tarea guardada, y rechaza la prioridad que
  se salga de su rango.
- **Ciclo de vida de una tarea**: la línea de comandos cierra una tarea con su motivo
  obligatorio —`done`, `duplicate`, `absorbed`, `stale` o `superseded`— y la reabre.
  Cerrar una tarea con subtareas abiertas se rechaza sin escribir nada, cerrar una
  subtarea no cierra su padre, y una tarea ya cerrada no se cierra otra vez ni una
  abierta se reabre. `create` acepta `--parent` para dar a la tarea nueva su tarea
  padre.
- **Jerarquía, dependencias y comentarios**: `block` deja una tarea bloqueada por otra
  y `unblock` retira el bloqueo; el estado bloqueado se calcula desde los bloqueos y no
  se guarda, así que la tarea deja de aparecer bloqueada al cerrar al bloqueante o al
  retirar el bloqueo. `comment` añade un comentario —su cuerpo es obligatorio— y los
  comentarios solo se añaden: no se editan ni se borran. `show` responde con el detalle
  de la tarea: sus hijas, sus bloqueantes, sus dependientes y sus comentarios.
- **Catálogo de etiquetas**: `tags` responde el catálogo ordenado por nombre, cada
  etiqueta con su color, su descripción y el número de tareas que la usan; `tag create`,
  `tag edit` y `tag delete` la crean, la cambian —solo lo que se pide— y la borran con
  sus asignaciones. Renombrar una etiqueta no modifica ninguna tarea, porque las
  asignaciones apuntan al número interno de la etiqueta, y borrar una avisa de las
  tareas que la usaban. `update` acepta `--tags "uno,dos"` para poner la lista entera de
  etiquetas de una tarea, y una lista vacía la deja sin ninguna. La paleta de colores y
  su color por omisión vienen de la interfaz de referencia del proyecto.
- **Documentación del producto**: requisitos y criterios de aceptación en
  [`docs/PRD.md`](docs/PRD.md); decisiones durables en
  [`docs/decisions/`](docs/decisions/); diseño en
  [`docs/architecture.md`](docs/architecture.md); y formas de datos, API, eventos,
  referencias y configuración en [`docs/contracts.md`](docs/contracts.md).
- **Reglas de trabajo**: el contrato de documentación y el contrato de flujo, y las
  instrucciones para agentes en [`AGENTS.md`](AGENTS.md).
- **Estructura, operación y riesgos**: [`docs/development.md`](docs/development.md),
  [`docs/operations.md`](docs/operations.md) y
  [`docs/known-risks.md`](docs/known-risks.md), junto con este registro y el
  [`README.md`](README.md).
- **Auditoría de las referencias**: análisis de la referencia funcional y de la
  interfaz de partida, con sus hallazgos registrados en el tablero de trabajo.
- **Servidor HTTP y avisos de cambio**: `serve` arranca la API HTTP del mismo binario
  sobre el mismo núcleo que la línea de comandos, con las rutas y los códigos de estado
  de [`docs/contracts.md`](docs/contracts.md). Cada cambio deja un aviso escrito en la
  misma transacción que el cambio, y `GET /api/events` los emite por un canal abierto:
  quien lee continúa donde se quedó, así que dos sesiones se ven y una recarga al
  recibir el aviso de la otra, sea el CLI o la propia API.
- **Herramienta de desarrollo** instalada en `task-manager/`, con su tablero y su
  catálogo de etiquetas.
- **Interfaz web en el binario** (`tk-4d8f.10`, primera rebanada): la interfaz de
  partida del proyecto de referencia vive ahora en `web/`, con su capa de peticiones
  ajustada a la API de este producto —identidad por referencia, cierre con motivo,
  bloqueos, etiquetas— y sus pantallas intactas. `serve` la reparte incrustada en el
  binario: la raíz del servidor abre la interfaz y la API conserva sus rutas
  `/api/...`. Desde la interfaz se crea, se ve, se cierra y se reabre una tarea.
  Decisión en [`docs/decisions/010-la-interfaz-vive-en-web-y-viaja-en-el-binario.md`](docs/decisions/010-la-interfaz-vive-en-web-y-viaja-en-el-binario.md).
