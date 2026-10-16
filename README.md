# tasks-ai

Gestor de tareas local-first para un proyecto, escrito en Go. Las tareas viven en una
base SQLite dentro del propio proyecto y se usan de dos maneras: por línea de comandos,
con salida JSON, y por una interfaz web que sirve el mismo binario.

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
go build ./cmd/...     # compila el binario
go test ./...          # ejecuta las pruebas
./tasks-ai init        # instala el producto en el proyecto: crea .task-ai/
./tasks-ai serve       # arranca la interfaz web
```

De la herramienta de desarrollo:

```bash
cd task-manager
npm ci                 # instala la herramienta
npm run odd:tk -- stats  # estado del trabajo
npm run server         # interfaz de la herramienta
```
