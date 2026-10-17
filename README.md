# tasks-ai

Gestor de tareas local-first para un proyecto, escrito en Go. Las tareas viven en una
base SQLite dentro del propio proyecto y se usan por línea de comandos, con salida
JSON; el mismo binario también sirve con `serve` la API HTTP y su interfaz web: la
interfaz responde en la raíz del servidor y la API conserva sus rutas `/api/...`. Las
rutas, los códigos de estado y los avisos de cambio están en
[`docs/contracts.md`](docs/contracts.md).

Todo lo del producto vive en la carpeta `.task-ai/` del proyecto —la base, la
configuración y las instrucciones—, de modo que el proyecto no se ensucia.

## Mapa de carpetas

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto |
| `docs/decisions/` | Una decisión durable por documento |
| `docs/tasks-ai-brand/` | La identidad visual aprobada: el símbolo, los iconos, los tokens de color y tipografía, y sus normas de uso |
| `docs/web/` | La guía de uso del producto, en una página web |
| `cmd/` | El binario del producto |
| `internal/` | Su núcleo y sus superficies |
| `web/` | Su interfaz web: el código de la de partida, ajustado a la API, y lo que el navegador descarga en `web/public/` |
| `.task-ai/` | El tablero del proyecto: la base de tareas, sus ajustes y sus instrucciones |

El contrato de documentación dice qué contiene cada documento de `docs/`.

## Requisitos de entorno

- Go, para el producto.
- Node, para la herramienta con la que se desarrolla el proyecto y para construir la
  interfaz: el binario la lleva incrustada, así que Node no hace falta para compilar
  en Go ni para usar el producto (`ADR-010`).

## Comandos

Del producto:

```bash
go build ./cmd/tasks-ai  # compila el binario

./tasks-ai init                  # instala el producto en el proyecto: deja su carpeta .task-ai/ con la base, los ajustes y las instrucciones; no es repetible, un proyecto ya instalado lo rechaza
./tasks-ai create --title "..."  # crea una tarea y devuelve su JSON, con su identidad y su referencia
./tasks-ai create --title "..." --parent tk-1  # crea una subtarea de la tarea padre que se le da
./tasks-ai list                 # devuelve todas las tareas del proyecto, las cerradas también, y cuál está bloqueada
./tasks-ai show <referencia>     # devuelve en JSON la tarea a la que apunta la referencia
./tasks-ai update <referencia> --priority 1  # cambia los campos que se piden: título, descripción, notas, responsable, prioridad, estado o etiquetas
./tasks-ai update <referencia> --status in_progress  # pone la tarea en curso: solo desde pendiente; --status open la devuelve a pendiente
./tasks-ai update <referencia> --tags "docs,urgente"  # pone la lista entera de etiquetas de la tarea: una lista vacía la deja sin ninguna
./tasks-ai tags                 # devuelve el catálogo de etiquetas con el uso de cada una
./tasks-ai tag create --name "docs" --color blue  # crea una etiqueta; el color que falta es blue
./tasks-ai tag edit <nombre> --name <nuevo>  # cambia de la etiqueta lo que se pide y nada más; renombrar no toca ninguna tarea
./tasks-ai tag delete <nombre>  # borra la etiqueta con sus asignaciones y avisa de las tareas que la usaban
./tasks-ai close <referencia> --reason done  # cierra una tarea con su motivo: done, duplicate, absorbed, stale o superseded
./tasks-ai reopen <referencia>   # reabre una tarea cerrada: vuelve a open y quita el motivo de cierre
./tasks-ai block <referencia> <referencia-bloqueante>   # deja la primera tarea bloqueada por la segunda
./tasks-ai unblock <referencia> <referencia-bloqueante>  # retira el bloqueo entre las dos tareas
./tasks-ai comment <referencia> --body "..."  # añade un comentario a la tarea: los comentarios solo se añaden
./tasks-ai serve                # arranca la API HTTP y la interfaz en la dirección y el puerto del settings: la interfaz responde en la raíz, y los cambios se anuncian por un canal abierto
```

La interfaz se reconstruye —`npm run build` dentro de `web/`— y el binario se vuelve
a compilar para que el artefacto nuevo viaje dentro (`ADR-010`).

De la interfaz:

```bash
cd web && npm ci      # instala su herramienta de construcción
cd web && npm run build  # construye la interfaz: deja el artefacto en web/public/task-manager/, que viaja incrustado en el binario
cd web && npm run watch  # la reconstruye al cambiar
```

De la herramienta de desarrollo —la de partida, que el producto sustituye—:

Se retiró del repositorio al pasar el trabajo a `.task-ai/`. Era un servidor en Node sobre
el binario de terceros [tacks](https://github.com/srmccray/tacks) —`tk`—, con un catálogo
de etiquetas propio. Su histórico de tareas se importó en el tablero del producto, y las
decisiones que la describen viven en `ADR-005` y `ADR-010`. No queda nada que arrancar.
