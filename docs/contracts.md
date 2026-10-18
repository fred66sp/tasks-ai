# Contratos del producto

Formas de datos y acuerdos de interfaz que no se pueden romper: el modelo, la salida
en JSON, la API, los eventos, las referencias, el settings y la integración con el
agente. Los requisitos están en `PRD.md` y el diseño en `architecture.md`.

## 1. Modelo de datos

Claves sustitutas enteras, nunca reutilizadas. Los nombres de campo son los del
código, en inglés.

### Tarea

| Campo | Tipo | Notas |
|---|---|---|
| `id` | entero | Clave. Identidad interna, no se presenta |
| `title` | texto | Obligatorio |
| `description` | texto | Opcional |
| `status` | enumerado | `open`, `in_progress`, `done`. Bloqueada no se guarda: se deriva |
| `blocked` | booleano | Se calcula desde las dependencias y no se almacena: cierto cuando la tarea tiene algún bloqueante que no está en `done` |
| `priority` | entero | Rango de `0` a `3`, por omisión `2` |
| `assignee` | texto | Opcional |
| `parent_id` | entero | Número interno de la tarea padre, opcional: es la identidad que cruza la frontera, y de él el cliente deriva la referencia visible del padre |
| `created_at` | fecha y hora | En UTC |
| `updated_at` | fecha y hora | En UTC |
| `close_reason` | enumerado | `done`, `duplicate`, `absorbed`, `stale`, `superseded` |
| `notes` | texto | Opcional |

### Etiqueta

| Campo | Tipo | Notas |
|---|---|---|
| `id` | entero | Clave. Mantiene el renombrado lejos de las tareas |
| `name` | texto | Único. Es lo que se muestra y se escribe, y se compara tal cual: `docs` y `Docs` son dos etiquetas distintas |
| `color` | texto | Uno de los colores de la paleta |
| `description` | texto | Opcional |

La paleta de colores y su color por omisión vienen de la interfaz de referencia del
proyecto (`project-task-manager`). La lista de colores aceptados vive en el núcleo
(`internal/core`) y en ningún otro sitio: este documento no la copia.

### Etiquetas de una tarea

Una fila por pareja de tarea y etiqueta. El recuento de uso de una etiqueta sale de
aquí. La fila apunta al número interno de la etiqueta, así que renombrar una etiqueta
cambia el nombre que muestran las tareas sin tocar su fila guardada, y borrar una
deja las tareas sin ella.

### Dependencia

Una fila por pareja: la tarea hija queda bloqueada por la padre. El estado bloqueado
se calcula desde estas filas y no se guarda. Bloquear la misma pareja otra vez no cambia
nada, y retirar un bloqueo que no está tampoco.

### Comentario

| Campo | Tipo | Notas |
|---|---|---|
| `id` | entero | Clave. Es la identidad del comentario y lo único que se presenta de él: un comentario no tiene referencia visible, se le nombra por su número dentro de su tarea |
| `task_id` | entero | Número interno de la tarea comentada. No cruza la frontera: la tarea se nombra por su referencia |
| `body` | texto | Obligatorio. Se guarda sin los espacios de los lados |
| `created_at` | fecha y hora | En UTC, en el momento en que se añade. No cambia al editar el cuerpo |

Un comentario se añade, se lee y se edita: al editarlo cambia solo el cuerpo, y su
identidad y su fecha se conservan. No hay borrado. Se leen del primero al último, por
fecha y, empatados, por identidad.

### Configuración de la base

Parejas de clave y valor dentro de la base, entre ellas la versión del esquema y el
prefijo con el que se compusieron las referencias de sus tareas.

## 2. Salida en JSON

Toda operación del CLI devuelve JSON. Una tarea se presenta así:

```json
{
  "id": 18,
  "ref": "tk-18",
  "title": "Preparar el análisis",
  "description": null,
  "status": "open",
  "blocked": false,
  "priority": 2,
  "assignee": null,
  "parent_id": null,
  "tags": ["docs"],
  "created_at": "2026-10-16T14:36:00Z",
  "updated_at": "2026-10-16T14:36:00Z",
  "close_reason": null,
  "notes": null
}
```

- `id` es la identidad interna y `ref` la referencia visible compuesta.
- Las etiquetas viajan por su nombre, ordenadas por él: las lleva cada tarea, y también
  las tareas de las listas del detalle.
- `tags` responde el catálogo ordenado por nombre, cada etiqueta con su identidad
  interna, su nombre, su color, su descripción y el número de tareas que la usan.
- `tag create` crea una etiqueta y responde con la guardada: el color que falta es el
  de por omisión de la paleta, un color desconocido es entrada inválida —código `2`— y
  un nombre que ya existe lo rechaza la regla que lo impone —código `1`—.
- `tag edit` cambia solo lo que se pide, como `update` cambia una tarea, y responde con
  la etiqueta tal como queda. Un cambio sin ningún campo es entrada inválida, un nombre
  que no existe se rechaza con el código `1` igual que una referencia que no resuelve,
  y renombrar sobre un nombre que ya lleva otra etiqueta se rechaza con el código `1`
  sin tocar ni el catálogo ni las tareas.
- `tag delete` borra la etiqueta con sus asignaciones y responde con la etiqueta
  borrada; si había tareas que la usaban, avisa de sus referencias por la salida de
  errores. Una etiqueta sin uso se borra sin aviso, y un nombre que no existe se
  rechaza con el código `1`.
- `update --tags "uno,dos"` pone la lista entera de etiquetas de la tarea, separadas
  por comas: una lista vacía la deja sin ninguna, y un nombre que no está en el
  catálogo es entrada inválida que no toca la tarea.
- `update --status` cambia el estado: `in_progress` pone la tarea en curso y solo
  cabe desde `open`, y `open` la devuelve a pendiente —desde `done` es reabrir, y le
  quita el motivo de cierre—. `done` no se pone por el cambio: cerrar va por `close`,
  con su motivo, y pedirlo se rechaza como entrada inválida —código `2`— sin tocar la
  tarea.
- `list` responde todas las tareas del proyecto con la forma de arriba —las cerradas
  también—, cada una con su referencia visible, su estado y su `blocked`, que es
  verdadero mientras alguna de sus bloqueantes siga sin cerrar.
- El detalle de una tarea añade sus bloqueantes, sus dependientes, sus hijas y sus
  comentarios, cada uno con su identidad, su cuerpo y su fecha:
  `"comments": [{ "id": 4, "body": "...", "created_at": "2026-10-16T15:02:00Z" }]`.
- Cerrar una tarea exige un motivo, deja su estado en `done`, guarda el motivo y
  refresca `updated_at`; reabrir la devuelve a `open` y le quita el motivo. Cerrar una
  tarea ya cerrada y reabrir una no cerrada se rechazan. Cerrar una tarea con subtareas
  abiertas —estado distinto de `done`— se rechaza sin escribir nada, y cerrar una
  subtarea no cierra a su padre.
- `block` deja la primera tarea bloqueada por la segunda y `unblock` retira ese
  bloqueo; las dos responden con la primera tarea tal como queda. Una referencia que
  no resuelve se rechaza con el código `1`, igual que `show`, y bloquearse una tarea
  por sí misma es entrada inválida: código `2`. Los ciclos se permiten. Una operación
  rechazada no escribe nada.
- `comment <referencia> --body` añade un comentario a la tarea y responde con la tarea
  tal como queda. El cuerpo es obligatorio: vacío o reducido a espacios es entrada
  inválida —código `2`— y no escribe nada.
- `comment list <referencia>` responde los comentarios de la tarea, del primero al
  último, cada uno con su `id`, su `body` y su `created_at`. Una referencia que no
  resuelve se rechaza con el código `1`.
- `comment edit <referencia> <id> --body` cambia el cuerpo del comentario de esa tarea
  con ese número y responde con la tarea tal como queda: la identidad y la fecha del
  comentario no cambian. El `id` se entiende dentro de la tarea: un número que no es de
  ningún comentario de esa tarea se rechaza con el código `1` sin tocar nada, y un cuerpo
  vacío es entrada inválida —código `2`—. La herramienta no ofrece borrar un comentario:
  lo que estaba mal se corrige editando su cuerpo.
- `init` instala el producto y devuelve en JSON lo que dejó: `dir`, `db`, `instructions`, `prefix` y `pointer` —el texto que hay que añadir fuera para apuntar a las instrucciones—. No es repetible: un proyecto que ya tiene la base y las instrucciones lo rechaza, y lo que una intenta anterior dejó a medias se completa sin borrar nada.
- Los errores del CLI son un objeto con `error` y salida distinta de cero: `2` cuando
  el uso de la línea de comandos no es el correcto o la entrada no es válida —lo que la
  API responde con `400`—, y `1` en cualquier otro caso. El texto de `error`, como la
  ayuda de los comandos, va en inglés (`ADR-011`).

## 3. API HTTP

| Método y ruta | Para qué |
|---|---|
| `GET /api/tasks` | Lista de tareas |
| `GET /api/tasks/:ref` | Detalle con bloqueantes, dependientes, hijas y comentarios |
| `POST /api/tasks` | Crear tarea o subtarea |
| `PATCH /api/tasks/:ref` | Cambiar título, descripción, notas, responsable, etiquetas, prioridad o estado —`open` o `in_progress`—, y devolver la tarea a `open`; cerrarla tiene su propia ruta, con su motivo. El estado va solo en el cambio: pedirlo junto a cualquier otro campo se rechaza como entrada inválida |
| `POST /api/tasks/:ref/close` | Cerrar con motivo |
| `POST /api/tasks/:ref/blockers` | Añadir un bloqueo |
| `DELETE /api/tasks/:ref/blockers/:blockerRef` | Retirar un bloqueo |
| `POST /api/tasks/:ref/comments` | Añadir un comentario |
| `GET /api/tasks/:ref/comments` | Los comentarios de la tarea, del primero al último, con su identidad |
| `PATCH /api/tasks/:ref/comments/:id` | Cambiar el cuerpo de un comentario, conservando su identidad y su fecha |
| `GET /api/tags` | Catálogo de etiquetas con su uso |
| `POST /api/tags` | Crear etiqueta |
| `PATCH /api/tags/:name` | Renombrar, recolorear o describir |
| `DELETE /api/tags/:name` | Borrar etiqueta |
| `GET /api/events` | Canal de cambios |
| `GET /api/status` | Estado del producto: versión, esquema y prefijo en uso |

Códigos de estado:

| Código | Cuándo |
|---|---|
| `200` | Operación resuelta |
| `201` | Recurso creado |
| `400` | Entrada inválida: prioridad fuera de rango, motivo de cierre desconocido, campos que faltan |
| `404` | La referencia no corresponde a ninguna tarea o etiqueta |
| `409` | La regla lo impide: cerrar una tarea con subtareas abiertas |
| `500` | Fallo interno |

Crear y editar un comentario responden con `200` y la tarea tal como queda, igual que el
CLI; `GET /api/tasks/:ref/comments` responde la lista con la forma de la sección 1. El
`id` de la ruta es el número del comentario dentro de la tarea de la ruta: un número que
no pertenece a esa tarea es `404`, y un cuerpo vacío es `400`.

El cuerpo de un error es `{"error": "..."}` con el código que corresponde, nunca un
error dentro de una respuesta correcta. Cada petición de escritura contiene un único
valor JSON; cualquier contenido posterior, aunque empiece por otro JSON válido, es
entrada inválida y no modifica el proyecto. El mensaje va en inglés y es el mismo que
el CLI escribe en su JSON de error: lo compone el núcleo, no cada superficie (`ADR-011`).

## 4. Eventos

Cada cambio deja un aviso escrito en la misma transacción que el cambio, sea la
superficie que lo pidió. `GET /api/events` mantiene un canal abierto
(`text/event-stream; charset=utf-8`, sin caché) y emite un evento por aviso, en el
orden en que ocurrieron. Cada evento viaja con el nombre `workspace.changed`, el
mensaje que la interfaz escucha:

```
id: 3
event: workspace.changed
data: {"resource":"tasks","changes":[{"id":"tk-3","action":"creada","title":"Preparar el análisis"}]}
```

- `id` es el número del aviso y `data` es el aviso en JSON: el recurso que cambió
  con el nombre en plural que la interfaz compara (`tasks` o `tags`). El aviso no
  dice quién pidió el cambio: el estado y el aviso son los mismos venga de donde
  venga, y la ventana que hizo el cambio no repite el suyo, lo evita con su propio
  estado local.
- Cuando el cambio toca una tarea, `changes` nombra a esa tarea como la imprime el
  aviso de la interfaz: su referencia visible (`id`), lo que le pasó (`creada`,
  `cerrada` o `actualizada`) y el título que tiene al emitir el aviso. La acción se
  guarda junto con el cambio, así que varios cambios de la misma tarea dentro de un
  sondeo conservan su propia descripción y un comentario también se anuncia como
  `actualizada`: lo es tanto cuando se añade como cuando se edita su cuerpo. Leer los
  comentarios no deja ningún aviso. El aviso de una etiqueta no lleva `changes`; los avisos
  antiguos escritos antes de guardar la acción se publican como `actualizada`.
- Quien lee continúa donde se quedó: en la primera lectura pide `?after=<id>` en la
  consulta, y al reconectar retoma del aviso que trae la cabecera `Last-Event-ID`.
  Sin ninguna de las dos, el canal empieza por el cambio siguiente: quien acaba de
  abrir la pantalla ya tiene delante el estado del proyecto, así que abrir o recargar
  no suelta los avisos de lo que pasó antes, y el canal cuenta solo lo que pasa
  mientras mira.
- El aviso se lee de la base, no de la memoria del servidor: un cambio pedido por el
  CLI llega al canal igual que uno pedido por la API, y eso es lo que hace que dos
  sesiones se vean. La interfaz recarga lo suyo y avisa de cualquier cambio de
  tarea; la ventana que lo pidió no se lo repite, porque lo evita con su propio
  estado local.
- El listado de tareas de la sección 3 entrega `parent_id` como el número interno del
  padre, igual que el modelo de la sección 1. La interfaz transforma ese número en la
  referencia visible al construir su árbol; la API, el CLI y la extensión Pi conservan
  la identidad numérica y no dependen del prefijo para enlazar tareas.
- Un canal pedido por alguien que no admite un canal abierto se rechaza con el código
  de entrada inválida de la sección 3, con el cuerpo de error de la API.

## 5. Referencias

- Formato: el prefijo del proyecto, un guion y el número interno. Por ejemplo `tk-18`.
- Se resuelven dentro del proyecto; el prefijo sale del settings.
- Aceptan un espacio alrededor y se comparan sin distinguir mayúsculas.
- Un cambio de prefijo no toca la base: cambia lo que se escribe.

## 6. Settings

Archivo `settings.json` dentro de la carpeta del producto, con valores anidados:

| Clave | Tipo | Por omisión |
|---|---|---|
| `prefix` | texto | `tk` |
| `db` | ruta | `tasks.db`, dentro de la carpeta del producto |
| `http.host` | texto | `127.0.0.1` |
| `http.port` | entero | `8890` |
| `log.level` | enumerado: `debug`, `info`, `warn`, `error` | `info` |

- El prefijo empieza por letra o número y sigue con letras, números, guiones o guiones bajos, hasta treinta y dos caracteres.

- El producto escribe el archivo al instalar y no reescribe lo que el usuario haya
  puesto. Cada sustitución prepara el JSON en la misma carpeta y lo renombra al final:
  si la escritura falla, el archivo anterior sigue intacto. Un archivo nuevo queda con
  permisos privados (`0600`); uno existente conserva sus permisos.
- Una clave desconocida también invalida el archivo y dice qué nombre no reconoce;
  no se aplica silenciosamente el valor por omisión.
- Un valor inválido detiene el arranque con un mensaje claro.
- Si el prefijo del archivo no coincide con el que la base tiene registrado, el
  producto avisa de que las referencias escritas con el prefijo anterior dejan de
  resolver, y continúa con el nuevo.

## 7. Integración con el agente

La extensión del agente Pi (`ADR-012`) no es una superficie del producto: envuelve el
CLI. Cada herramienta ejecuta un comando y devuelve su JSON; ninguna toca la base ni
recompone un mensaje del producto.

### Herramientas

| Herramienta | Comando | Parámetros | Devuelve |
|---|---|---|---|
| `tasks_board` | `list` | ninguno | el tablero derivado: contadores, en curso, listas para empezar y árbol |
| `tasks_show` | `show` | `ref` | la tarea con sus hijas, sus bloqueantes, sus dependientes y sus comentarios |
| `tasks_create` | `create` | `title` obligatorio; `description`, `notes`, `assignee`, `priority` y `parent` opcionales | la tarea creada |
| `tasks_update` | `update` | `ref`, más los campos que cambian entre `title`, `description`, `notes`, `assignee`, `priority`, `status` y `tags` | la tarea tal como queda |
| `tasks_close` | `close` | `ref` y `reason` | la tarea cerrada |
| `tasks_reopen` | `reopen` | `ref` | la tarea reabierta |
| `tasks_block` | `block` | `ref` y `blocker` | la tarea bloqueada |
| `tasks_unblock` | `unblock` | `ref` y `blocker` | la tarea sin ese bloqueo |
| `tasks_comment` | `comment` | `ref` y `body` | la tarea con el comentario añadido |

- `tasks_board` no tiene comando propio: sale de `list`, que devuelve todas las tareas
  con todos sus campos. La derivación es esta: contadores por estado; en curso, las que
  están en `in_progress`; listas para empezar, las `open` que no están bloqueadas; árbol,
  por `parent_id`.
- `tasks_create` no acepta etiquetas, porque `create` no las tiene: se ponen después con
  `tasks_update`.
- `tasks_unblock` acompaña a `tasks_block` porque el CLI tiene las dos y una herramienta
  sin su contraria deja el vocabulario a medias.
- `init`, `serve`, `tags` y `tag` no son herramientas: instalar, servir y mantener el
  catálogo no son operaciones del ciclo de trabajo del agente. Las etiquetas de una tarea
  se leen dentro de la tarea, por su nombre.

### Parámetros y validación

- Las referencias se escriben tal cual las escribe el producto —`tk-18`—: la extensión no
  compone ninguna ni acepta el número interno suelto.
- Lo que la extensión declara en su esquema es la misma regla que impone el CLI: una
  prioridad de `0` a `3`, un motivo de cierre de los cinco conocidos, un cuerpo de
  comentario no vacío. El producto lo vuelve a comprobar, y su rechazo es el que vale.
- Un campo que no se nombra no cambia; `tasks_update` no manda lo que no se le ha dado.

### Lo que llega al modelo

El contexto es el recurso escaso: la extensión proyecta el tablero para que el agente
decida, no para que lo lea entero.

- De cada tarea viajan la referencia, el título, el estado, si está bloqueada, la
  prioridad, el padre y las etiquetas. **Nunca** descripciones, notas ni comentarios: eso
  solo sale en `tasks_show`, que es la puerta al detalle y se pide a propósito.
- El árbol del tablero se recorta a 40 tareas y lo que queda fuera se cuenta en una
  línea, para que el agente sepa que hay más y lo pida.
- Las listas de en curso y de lo que se puede empezar se recortan a 8, con su recuento.
- El detalle de una tarea trae su descripción entera y los 10 comentarios más recientes,
  con el total a la vista.

Los límites viven en la extensión, no en el producto: el CLI sigue devolviendo todo, y
quien lea su JSON recibe la tarea completa.

### Errores

- La salida `2` es el mal uso: una bandera que no encaja o una entrada que el núcleo
  rechaza como inválida. La salida `1` es el fallo: el proyecto no está instalado, la
  referencia no resuelve, una regla lo impide.
- Las dos llegan como error de la herramienta con el mensaje del producto tal cual, y con
  su `{"error": "..."}` delante cuando lo trae. La extensión no traduce ni reinterpreta:
  si el texto cambia en el producto, cambia en la herramienta.
- Un rechazo no deja nada escrito, y la herramienta no vuelve a leer el estado para
  arreglarlo por su cuenta.

### Fuera del contrato

- Un comando o una herramienta `ready`: lo que se puede empezar se deriva en
  `tasks_board`.
- Un importador: traer el tablero anterior fue una pasada de retirada, no una capacidad
  del producto.
- Un servidor MCP: la forma elegida es la extensión (`ADR-012`), y su condición de
  revisión está en esa decisión.
