# Registro de cambios

Cambios relevantes del proyecto. El formato sigue las recomendaciones de *Keep a
Changelog* y el versionado es semántico.

## [Sin publicar]

### Modificado

- **Las superficies del producto hablan inglés**: la ayuda de todos los comandos, los
  mensajes de error del CLI, sus avisos por la salida de errores y el cuerpo `error` de
  la API se emiten en inglés, con los mismos códigos de salida y de estado. La
  documentación, los comentarios y el texto de la interfaz web siguen en español
  (`ADR-011`).

### Añadido

- **El agente opera el tablero con herramientas**: la extensión de Pi en
  `extensions/pi/` envuelve el CLI en nueve herramientas con esquema —`tasks_board`,
  `tasks_show`, `tasks_create`, `tasks_update`, `tasks_close`, `tasks_reopen`,
  `tasks_block`, `tasks_unblock` y `tasks_comment`—, pone el tablero resumido delante
  del agente en cada turno, añade el comando `/board` y una línea de estado con la tarea
  en curso, y deja los comandos a mano para la persona. Vive fuera del binario, con sus
  pruebas propias (`ADR-012`), y no toca la base ni reimplementa reglas: cada
  herramienta ejecuta su comando y devuelve su JSON, rechazos incluidos.

- **La interfaz lleva la identidad del producto**: el encabezado enseña el símbolo
  aprobado en lugar de la letra provisional, la paleta y las tipografías son las de la
  marca, y la pestaña del navegador y el manifiesto del sitio traen sus iconos. Los
  estados de la tarea se leen por encima del contraste AA, sin depender solo del color.
  Nada de lo que la interfaz hace cambia: las mismas acciones, los mismos avisos y los
  mismos datos.

### Documentación

- La guía de uso del producto en una página web: `docs/web/index.html` recorre el producto
  entero —instalarlo, modelar el trabajo, cada orden con sus opciones y la respuesta que
  devuelve, el ciclo de vida, las recetas, la interfaz, la API, la configuración, los
  límites y los errores— y sirve para aprender a manejarlo sin abrir otra fuente. Está en
  español y en inglés, con cada ejemplo en los dos idiomas y la respuesta del binario
  debajo, y viste la identidad aprobada, que llega con este cambio en `docs/tasks-ai-brand/`.
  El contrato de documentación le da su misión y su frontera, y el mapa del `README.md`
  nombra las dos carpetas.
- `docs/contracts.md` describe lo que la API responde de verdad: `parent_id` viaja como
  el número interno del que el cliente deriva la referencia del padre, el cambio de
  tarea con `status` junto a otro campo se rechaza, y el nivel de registro enumera sus
  valores junto al formato del prefijo.
- El contrato de documentación ya no nombra una suite de comprobaciones que el
  repositorio no tiene: las comprobaciones corren a mano, igual que dicen el registro
  de riesgos y el de operaciones.
- Quedan fuera de la documentación lo obsoleto: la tabla de ajustes duplicada del PRD
  ahora se enlaza a `contracts.md`, la carpeta del producto ya no anuncia skills que
  la instalación no escribe, y `init` se dice no repetible y con el JSON que devuelve.

### Añadido

- **Ver todas las tareas por CLI**: `list` responde en JSON todas las tareas del
  proyecto —las cerradas también—, cada una con su referencia visible, su estado, su
  título y si está bloqueada. Antes solo la interfaz y la API sabían listarlas, así que
  el ciclo de trabajo de un agente por CLI no se podía cerrar sin ellas.
- **Poner una tarea en curso**: una tarea pendiente se pone en curso con `update
  --status in_progress` y vuelve a pendiente con `--status open`; ponerse en curso solo
  cabe desde pendiente, y `--status done` se rechaza porque cerrar va por `close`, con
  su motivo. El cambio de estado de la API acepta `in_progress`, la interfaz ofrece
  «En curso» ya funcionando, y «Bloqueada» sigue viéndose sin poder elegirse, porque es
  un estado que se calcula.
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
  recibir el aviso de la otra, sea el CLI o la propia API. Quien abre o recarga la
  pantalla empieza en el cambio siguiente: no recibe el historial de lo que ya estaba
  antes de abrirla.
- **El tablero del proyecto es el producto**: el trabajo se registra en `.task-ai/`, la
  carpeta que instala el propio binario, y se opera con `./bin/tasks-ai` y `serve`. La
  herramienta de desarrollo anterior —un servidor en Node sobre `tk`— se retiró del
  repositorio, y sus 58 tareas se importaron en este tablero con sus descripciones,
  etiquetas, comentarios y dependencias.
- **Interfaz web en el binario** (`tk-4d8f.10`, primera rebanada): la interfaz de
  partida del proyecto de referencia vive ahora en `web/`, con su capa de peticiones
  ajustada a la API de este producto —identidad por referencia, cierre con motivo,
  bloqueos, etiquetas— y sus pantallas intactas. `serve` la reparte incrustada en el
  binario: la raíz del servidor abre la interfaz y la API conserva sus rutas
  `/api/...`. Desde la interfaz se crea, se ve, se cierra y se reabre una tarea.
  Decisión en [`docs/decisions/010-la-interfaz-vive-en-web-y-viaja-en-el-binario.md`](docs/decisions/010-la-interfaz-vive-en-web-y-viaja-en-el-binario.md).
- **La pantalla se entera sola de los cambios** (`tk-4962.16`): cada aviso del canal
  `GET /api/events` viaja con el nombre `workspace.changed`, el mensaje que la interfaz
  escucha, y su carga nombra el recurso en plural y, cuando toca una tarea, la propia
  tarea con su referencia, lo que le pasó y su título. Con la pantalla abierta, un
  cambio hecho desde la línea de comandos
  aparece solo, con su aviso de qué tarea cambió, y el
  progreso de las épicas y el recuento lo reflejan. El listado de tareas entrega
  además cada `parent_id` resuelto a la referencia visible del padre, así que el
  árbol anida y cada épica muestra el progreso de sus subtareas.

### Corregido

- **La extensión no deja al agente sin sitio donde anotar**: en un proyecto sin tablero
  ya no retira la herramienta `todo` del arnés; la deja en su sitio y el resumen lo dice
  en una línea. Con tablero, `todo` sigue fuera mientras la extensión está cargada, y
  vuelve al descargarla.

- **El aviso de un cambio ya no depende de quién lo pidió** (`tk-4962.21`): el aviso
  del canal `GET /api/events` no lleva la superficie que pidió el cambio —la interfaz
  enviaba una identidad de sesión (`?clientId=`, `x-client-id`) que el servidor ni
  leía ni devolvía, así que un cambio hecho desde otra ventana recargaba la pantalla
  sin avisar—. Ahora el aviso sale para cualquier cambio de tarea, venga de la línea
  de comandos, de otra ventana o de la misma, y la ventana que pidió el cambio no se
  lo repite: lo evita con su propio estado local, sin identidad viajando al
  servidor.
- **El servidor se apaga en el acto aunque queden conexiones de más** (`tk-95b5.1`): con
  una sesión leyendo el canal de avisos, apagarlo podía tardar cinco segundos y salir con
  error, porque una conexión aceptada que todavía no ha pedido nada —el navegador abre
  más de las que usa— retiene el apagado ordenado de Go hasta los cinco segundos que él
  le concede. El plazo propio baja a un segundo, de sobra para las peticiones de un
  proyecto local, y lo que siga abierto se cierra sin quejarse.
- **El suelo de cobertura del flujo de pruebas deja de ser inalcanzable** (`tk-95b5.2`):
  el flujo pedía una cobertura que la suite nunca alcanzaba, así que el paso fallaba con
  todas las pruebas en verde. El número pasa a lo que la suite mide de verdad y queda
  como trinquete: a partir de ahí solo puede subir.
- **El flujo de pruebas corre con una sola versión de Go** (`tk-95b5.12`): la matriz
  clavaba una versión concreta además de la estable, y el suelo de cobertura colgaba de
  la clavada. Con una sola versión, el suelo se comprueba contra la que se instala.
- **El aviso de un cambio de tarea no puede ejecutar HTML** (`tk-95b5.3`): el título de
  la tarea llegaba al aviso y el diálogo de confirmación lo insertaba como HTML. Ahora
  todo texto que viene del servidor se escapa antes de mostrarse.
- **La interfaz ya no pierde la escucha del diálogo ni cierra lo de debajo**
  (`tk-95b5.8`): cerrar una confirmación dejaba viva su escucha de teclado, y Escape
  cerraba además el cajón o el formulario que quedaban detrás.
- **Buscar «null» ya no devuelve las etiquetas sin descripción** (`tk-95b5.10`), y
  renombrar una etiqueta conserva el filtro que la seguía.
- **El listado de tareas deja de consultar la base tarea a tarea** (`tk-95b5.11`):
  resuelve los bloqueos y las etiquetas en dos consultas en lote, con el mismo JSON.
- **El flujo de pruebas vuelve a pasar**: el listado por lotes traía consultas nuevas sin
  ninguna prueba que las recorriera, así que la cobertura bajaba del suelo. Ahora el
  listado tiene su prueba —el bloqueo calculado, las etiquetas ordenadas y un proyecto sin
  tareas— y el suelo se mide con margen sobre lo que mide el servidor, no pegado a la
  última cifra.
- **El repositorio deja de versionar los binarios de la herramienta de desarrollo**
  (`tk-95b5.7`): `tk` y `tk.exe` son de terceros y se instalan aparte, y la base del
  tablero nunca vivió aquí, así que tenerlos versionados no hacía el clon más útil. Se
  van también el Vue de desarrollo, que no lo cargaba ninguna página, y las órdenes del
  manifiesto de la herramienta que apuntaban a un `src/` y unos `tests/` que no están en
  el repositorio, con sus dependencias.
