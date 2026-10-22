# Registro de cambios

Cambios relevantes del proyecto. El formato sigue las recomendaciones de *Keep a
Changelog* y el versionado es semántico.

## [Sin publicar]

### Añadido

- **`init` completa una instalación anterior**: rechaza solo la que ya lo tiene todo
  —base, instrucciones, binario y extensión— y, a la que le falta el binario o la
  extensión, le añade lo que falta sin tocar lo que tiene. Es como migran los proyectos
  instalados antes de que el producto dejara esas dos piezas dentro de su carpeta.

- **El install deja el proyecto autocontenido**: `init` copia el binario a
  `.tasks-ai/bin/tasks-ai` y extrae la extensión del agente —incrustada en el binario con
  `go:embed`— a `.tasks-ai/extensions/pi/`. El proyecto ya no depende de la ruta del
  repositorio de origen: la extensión se instala con `pi install -l
  .tasks-ai/extensions/pi` y el binario responde desde su propia carpeta. La decisión en
  `ADR-014`, que supera a `ADR-012`.

- **El texto de una tarea no guarda el estado**: el título, la descripción, las notas y el
  cuerpo de un comentario se rechazan si nombran un hash de commit, una rama o una
  referencia a un remoto, o si declaran el estado de la tarea o de la entrega con la forma
  que el producto reconoce. El núcleo comprueba el texto antes de escribir, así que el
  rechazo no deja nada guardado y una creación rechazada no crea la tarea: sale con la
  salida `2` por el CLI, con `400` por la API y como error de la herramienta en la
  extensión del agente, con el mismo mensaje que nombra el campo y el fragmento. Lo
  entrecomillado no exime. Las listas cerradas están en `docs/contracts.md`, lo que queda
  fuera de ellas en `docs/known-risks.md`, y la decisión en `ADR-013`.

- **El padre de una tarea se puede cambiar después de crearla**: `update --parent <ref>`
  la asigna, la mueve a otro padre o, con el valor vacío, la deja en su nivel. Un ciclo
  —tomar como padre a la propia tarea o a una de sus descendientes— y una referencia que
  no resuelve se rechazan sin escribir nada. Al quedar con padre la tarea lleva la
  etiqueta `task` y al quedarse sin padre la pierde; si el catálogo no la tiene, el
  producto la crea con el color por omisión. El padre puede estar cerrado: colgar una
  subtarea de una tarea hecha está permitido. Está en el CLI (`update --parent`), la API
  (`PATCH /api/tasks/:ref` con `parent`), la interfaz web (campo «Tarea padre» en la
  edición del inspector) y la extensión de Pi (`tasks_update` con `parent`).
- **La tarea que nace con padre lleva su etiqueta `task` desde el primer momento**:
  `create --parent` la asigna en la misma creación —creándola en el catálogo si falta—,
  igual que el cambio de padre la ajusta después.

- **El buscador de la interfaz encuentra tareas por su contenido**: además del título,
  el identificador y las etiquetas, la búsqueda cubre la descripción, las notas y los
  comentarios de cada tarea. Se apoya en el nuevo endpoint `GET /api/comments`, que
  devuelve todos los comentarios del proyecto con la referencia de la tarea que los
  lleva.
- **Los comentarios se pueden leer y editar**: además de añadirlos, `comment list`
  devuelve los de la tarea del primero al último con su identidad y su fecha, y
  `comment edit <referencia> <id> --body` cambia el cuerpo de uno conservando su
  identidad y su fecha de creación. No hay borrado: lo que estaba mal se corrige
  editando. Las operaciones están en el CLI, la API (`GET` y `PATCH` sobre
  `/api/tasks/:ref/comments`), la interfaz web (identidad y fecha visibles, edición
  desde el inspector) y la extensión de Pi (`tasks_comment_list` y
  `tasks_comment_edit`).

- **La descripción, las notas y los comentarios se dibujan con su formato**: el
  inspector los presenta con el formato que traigan —énfasis, listas, enlaces, tablas—
  en lugar de mostrar sus etiquetas, y conserva sus saltos de línea. Lo que ejecuta o
  carga algo se descarta antes de dibujarlo, con una lista blanca de etiquetas y
  atributos, porque el texto llega de cualquier sitio: se pega, se escribe por el CLI o
  se manda por la API. Está en la interfaz
  web: la descripción y las notas del detalle y el cuerpo de cada comentario.

### Modificado

- **La carpeta del producto se llama `.tasks-ai/`**: se corrige la errata del nombre —le
  faltaba la letra ese— en el código, los documentos y las instalaciones. El producto
  busca y crea `.tasks-ai/`, y las carpetas existentes se renombran en consecuencia.

- **La línea de estado del agente enseña la última tarea cerrada**: la extensión de Pi
  deja de repetir en el estado del TUI lo que ya enseña el bloque del tablero, y nombra
  la tarea cerrada más reciente —`🦊 última cerrada: tk-158 · título`—, con «sin tarea
  cerrada» mientras no haya ninguna. El bloque junto al editor y `tasks_board` siguen
  enseñando lo que está en curso, que es el trabajo abierto.

- **El tablero deja de viajar en el repositorio**: `.tasks-ai/tasks.db` sale del índice y
  el `.gitignore` la deja fuera, así que cada máquina tiene su base y el conflicto de un
  binario de SQLite deja de ser posible. Las instrucciones del producto siguen
  versionadas; lo que viaja con el trabajo nuevo es el código, no las tareas. Llevar el
  tablero a otra máquina pasa a ser un traslado manual, que todavía no tiene volcado.

- **El tablero y el estado de Pi viajan en el repositorio**: `.tasks-ai/tasks.db` y `.pi/`
  se versionan para que otra máquina arranque con las tareas y el agente de delegación ya
  puestos. El agente vive solo en `.pi/agents/`, que es donde Pi lo lee, sin una copia que
  mantener sincronizada. La base se publica con el contenido del WAL integrado, y
  `tasks.db-wal` y `tasks.db-shm` no viajan porque son transitorios y copiarlos corrompe
  lecturas. Un binario no se fusiona: el conflicto entre dos máquinas está en
  `docs/known-risks.md`.

- **Los settings nuevos quedan privados**: la escritura atómica usa permisos `0600`
  cuando todavía no existe un archivo y conserva los permisos de uno existente.

- **Settings rechaza claves desconocidas**: una errata como `prefx` deja de aplicar
  silenciosamente el valor por omisión y muestra el nombre de la clave no reconocida.

- **Guardar settings sustituye el archivo de forma atómica**: una escritura o
  sustitución fallida conserva el JSON anterior y evita que el producto vuelva a leer
  un archivo truncado.

- **Las reconexiones SSE recorren el historial en lotes**: los nombres de las
  etiquetas se leen agrupados y un aviso de una etiqueta borrada sigue llegando sin
  romper el canal.

- **Abrir una base ya migrada evita transacciones de comprobación**: las migraciones
  pendientes conservan su transacción y la protección ante aperturas concurrentes.

- **Las bases existentes crean índices para las consultas de detalle**: la migración
  añade índices sobre `comments.task_id` y `dependencies.blocker_id` sin alterar los
  comentarios, dependencias ni respuestas existentes.

- **La API rechaza contenido posterior al JSON de una escritura**: un cuerpo debe
  contener un único valor JSON; si trae basura o un segundo valor, la operación
  responde con error sin modificar el proyecto.

- **Los avisos SSE conservan cada cambio de tarea**: crear, cerrar, actualizar y
  comentar una tarea llevan su propia acción y título, aunque varias escrituras hayan
  ocurrido dentro del mismo sondeo. Las bases existentes migran su historial sin
  perder avisos; los avisos antiguos se presentan como actualizaciones.

- **`init` avisa al completar una instalación con prefijo distinto**: cuando la base ya
  conserva referencias escritas con otro prefijo, completar `settings.json` e
  instrucciones mantiene la instalación y comunica que esas referencias dejan de
  resolver.

- **La línea del tablero en el TUI lleva el símbolo de la marca**: el estado de la
  extensión empieza por el zorro, como el widget de otras extensiones empieza por el
  suyo, y cuando hay varias tareas en curso las nombra todas con su recuento en vez de
  enseñar solo la primera. El recorte sigue en 100 caracteres, y con una sola tarea el
  texto no cambia. Además, junto al editor se ve el tablero como bloque: el título del
  producto y una línea por tarea en curso, sin recortar.

- **Las superficies del producto hablan inglés**: la ayuda de todos los comandos, los
  mensajes de error del CLI, sus avisos por la salida de errores y el cuerpo `error` de
  la API se emiten en inglés, con los mismos códigos de salida y de estado. La
  documentación, los comentarios y el texto de la interfaz web siguen en español
  (`ADR-011`).

### Añadido

- **`init` fija el prefijo del proyecto**: `./tasks-ai init --prefix tai` compone las
  referencias con ese prefijo en vez de `tk`, lo graba en la base que crea y escribe las
  instrucciones con él en sus ejemplos, así que el proyecto no arrastra el aviso de que
  la base se compuso con otro. Sin la bandera la instalación nace con `tk` como antes, y
  pedir un prefijo distinto del que ya tienen los ajustes del proyecto se rechaza en vez
  de sobrescribirlo.

- **La extensión se instala como paquete de Pi**: `extensions/pi` declara su manifiesto y
  se instala en el proyecto con `pi install -l ./extensions/pi`, que deja la ruta relativa
  en `.pi/settings.json` sin copiar el fuente fuera del repositorio; `pi remove -l
  --approve ./extensions/pi` la retira y deja la configuración como estaba. Una sesión
  nueva de Pi carga sus herramientas sin pasar `--extension`, y como el paquete de
  proyecto lo lee después de conceder confianza, la primera vez hay que aceptarla.

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
  el número interno del que la interfaz deriva la referencia del padre, el cambio de
  tarea con `status` junto a otro campo se rechaza, y el nivel de registro enumera sus
  valores junto al formato del prefijo.
- El contrato de documentación ya no nombra una suite de comprobaciones que el
  repositorio no tiene: las comprobaciones corren a mano, igual que dicen el registro
  de riesgos y el de operaciones.
- Quedan fuera de la documentación lo obsoleto: la tabla de ajustes duplicada del PRD
  ahora se enlaza a `contracts.md`, la carpeta del producto ya no anuncia skills que
  la instalación no escribe, y `init` se dice no repetible y con el JSON que devuelve.
- La guía de uso describe la línea de estado como la extensión la escribe: nombra la
  última tarea cerrada —recortada a 100 caracteres— y, sin ninguna cerrada, lo dice con
  una frase propia; las tareas en curso siguen en el bloque junto al editor. En español
  y en inglés, con el diccionario de la página sin claves huérfanas.
- El contrato de etiquetas reparte la autoría de las estructurales: `ODD` se asigna a
  mano, `epic` la pone quien crea la tarea y el producto no la añade ni la retira, y
  `task` la pone y la retira el producto al cambiar la forma. El documento dice también
  que una lista `--tags` sin cambio de padre sustituye las etiquetas que la tarea tenía.

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
- **El tablero del proyecto es el producto**: el trabajo se registra en `.tasks-ai/`, la
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
  `parent_id` como número interno; la interfaz lo transforma en la referencia visible
  para que el árbol anide y cada épica muestre el progreso de sus subtareas.

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
- **El tipo de una tarea en la interfaz mira su forma, no solo su etiqueta**
  (`tk-159`): una tarea con subtareas se presenta como «Épica» aunque nadie le ponga
  `epic`, y una con la etiqueta y sin subtareas sigue siendo «Épica». El padre distingue
  la subtarea de la tarea suelta, y el rótulo, el icono y el inspector dicen lo mismo.
