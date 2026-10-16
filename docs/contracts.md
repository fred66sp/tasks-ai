# Contratos del producto

Formas de datos y acuerdos de interfaz que no se pueden romper: el modelo, la salida
en JSON, la API, los eventos, las referencias y el settings. Los requisitos están en
`PRD.md` y el diseño en `architecture.md`.

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
| `priority` | entero | Rango de `0` a `3`, por omisión `2` |
| `assignee` | texto | Opcional |
| `parent_id` | entero | Tarea padre, opcional |
| `created_at` | fecha y hora | En UTC |
| `updated_at` | fecha y hora | En UTC |
| `close_reason` | enumerado | `done`, `duplicate`, `absorbed`, `stale`, `superseded` |
| `notes` | texto | Opcional |

### Etiqueta

| Campo | Tipo | Notas |
|---|---|---|
| `id` | entero | Clave. Mantiene el renombrado lejos de las tareas |
| `name` | texto | Único. Es lo que se muestra y se escribe |
| `color` | texto | Uno de los colores de la paleta |
| `description` | texto | Opcional |

### Etiquetas de una tarea

Una fila por pareja de tarea y etiqueta. El recuento de uso de una etiqueta sale de
aquí.

### Dependencia

Una fila por pareja: la tarea hija queda bloqueada por la padre. El estado bloqueado
se calcula desde estas filas y no se guarda.

### Comentario

Tarea, cuerpo y fecha. Solo se añaden: no se editan ni se borran.

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
- Las etiquetas viajan por su nombre.
- El detalle de una tarea añade sus bloqueantes, sus dependientes, sus hijas y sus
  comentarios.
- Cerrar una tarea exige un motivo, deja su estado en `done`, guarda el motivo y
  refresca `updated_at`; reabrir la devuelve a `open` y le quita el motivo. Cerrar una
  tarea ya cerrada y reabrir una no cerrada se rechazan. Cerrar una tarea con subtareas
  abiertas —estado distinto de `done`— se rechaza sin escribir nada, y cerrar una
  subtarea no cierra a su padre.
- Los errores del CLI son un objeto con `error` y salida distinta de cero: `2` cuando
  el uso de la línea de comandos no es el correcto o la entrada no es válida —lo que la
  API responde con `400`—, y `1` en cualquier otro caso.

## 3. API HTTP

| Método y ruta | Para qué |
|---|---|
| `GET /api/tasks` | Lista de tareas |
| `GET /api/tasks/:ref` | Detalle con bloqueantes, dependientes, hijas y comentarios |
| `POST /api/tasks` | Crear tarea o subtarea |
| `PATCH /api/tasks/:ref` | Cambiar título, descripción, notas, responsable, etiquetas, prioridad o estado |
| `POST /api/tasks/:ref/close` | Cerrar con motivo y comentario |
| `POST /api/tasks/:ref/blockers` | Añadir un bloqueo |
| `DELETE /api/tasks/:ref/blockers/:blockerRef` | Retirar un bloqueo |
| `POST /api/tasks/:ref/comments` | Añadir un comentario |
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

El cuerpo de un error es `{"error": "..."}` con el código que corresponde, nunca un
error dentro de una respuesta correcta.

## 4. Eventos

`GET /api/events` mantiene un canal abierto y emite un evento por cambio, con el
recurso afectado, el origen del cambio y las referencias que han cambiado. La interfaz
recarga lo suyo y avisa cuando el origen no es ella.

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
| `log.level` | enumerado | `info` |

- El producto escribe el archivo al instalar y no reescribe lo que el usuario haya
  puesto.
- Un valor inválido detiene el arranque con un mensaje claro.
- Si el prefijo del archivo no coincide con el que la base tiene registrado, el
  producto avisa de que las referencias escritas con el prefijo anterior dejan de
  resolver, y continúa con el nuevo.
