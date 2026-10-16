# Gestor de tareas del proyecto

Este proyecto guarda sus tareas con el gestor local que vive en esta carpeta.

## Qué hay aquí

| Elemento | Qué es |
|---|---|
| `tasks.db` | Las tareas del proyecto |
| `settings.json` | El prefijo de las referencias, la base y los ajustes de la interfaz |
| `AGENTS.md` | Estas instrucciones |

## Cómo se usa

Desde cualquier punto del proyecto:

```bash
tasks-ai create --title "Preparar el análisis"   # crea una tarea y devuelve su JSON
tasks-ai create --title "Revisar el análisis" --parent tk-1  # crea una subtarea de tk-1
tasks-ai show tk-1                               # devuelve la tarea en JSON
tasks-ai update tk-1 --priority 1                # cambia los campos que se piden y devuelve la tarea
tasks-ai close tk-1 --reason done                # cierra la tarea con su motivo y devuelve la tarea
tasks-ai reopen tk-1                             # reabre la tarea cerrada y devuelve la tarea
```

El motivo de cierre es obligatorio y debe ser uno de: `done`, `duplicate`, `absorbed`,
`stale` o `superseded`. Cerrar una tarea con subtareas abiertas se rechaza sin escribir
nada, cerrar una subtarea no cierra su padre, y una tarea ya cerrada no se cierra otra
vez ni una abierta se reabre.

Toda operación responde en JSON por la salida estándar, y los avisos y errores van por
la salida de error con la forma `{"error": "..."}`.

## Las referencias

Cada tarea tiene un número interno, que es su identidad, que es único en el proyecto y
que no se reutiliza. La referencia visible se compone con el prefijo del proyecto y ese
número: `tk-1`. Se escribe sin distinguir mayúsculas y admite espacios alrededor.
