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
tasks-ai show tk-1                               # devuelve la tarea en JSON
```

Toda operación responde en JSON por la salida estándar, y los avisos y errores van por
la salida de error con la forma `{"error": "..."}`.

## Las referencias

Cada tarea tiene un número interno, que es su identidad, que es único en el proyecto y
que no se reutiliza. La referencia visible se compone con el prefijo del proyecto y ese
número: `tk-1`. Se escribe sin distinguir mayúsculas y admite espacios alrededor.
