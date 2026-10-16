# tasks-ai

Gestor de tareas local-first para un proyecto, escrito en Go. Las tareas viven en una
base SQLite dentro del propio proyecto y hoy se usan por línea de comandos, con salida
JSON. La interfaz web que sirve el mismo binario está por llegar; su diseño está en
[`docs/PRD.md`](docs/PRD.md).

Todo lo del producto vive en la carpeta `.task-ai/` del proyecto —la base, la
configuración y las instrucciones—, de modo que el proyecto no se ensucia.

## Mapa de carpetas

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto |
| `docs/decisions/` | Una decisión durable por documento |
| `cmd/` | El binario del producto |
| `internal/` | Su núcleo y sus superficies |
| `task-manager/` | La herramienta con la que se desarrolla el proyecto, con su tablero |

El contrato de documentación dice qué contiene cada documento de `docs/`.

## Requisitos de entorno

- Go, para el producto.
- Node, para la herramienta con la que se desarrolla el proyecto.

## Comandos

Del producto:

```bash
go build ./cmd/tasks-ai  # compila el binario

./tasks-ai init                  # instala el producto en el proyecto: deja su carpeta .task-ai/ con la base, los ajustes y las instrucciones
./tasks-ai create --title "..."  # crea una tarea y devuelve su JSON, con su identidad y su referencia
./tasks-ai create --title "..." --parent tk-1  # crea una subtarea de la tarea padre que se le da
./tasks-ai show <referencia>     # devuelve en JSON la tarea a la que apunta la referencia
./tasks-ai update <referencia> --priority 1  # cambia los campos que se piden: título, descripción, notas, responsable o prioridad
./tasks-ai close <referencia> --reason done  # cierra una tarea con su motivo: done, duplicate, absorbed, stale o superseded
./tasks-ai reopen <referencia>   # reabre una tarea cerrada: vuelve a open y quita el motivo de cierre
```

De la herramienta de desarrollo:

```bash
cd task-manager
npm ci                 # instala la herramienta
npm run odd:tk -- stats  # estado del trabajo
npm run server         # interfaz de la herramienta
```
