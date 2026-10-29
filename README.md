# tasks-ai

Gestor de tareas local-first para un proyecto, escrito en Go. Las tareas viven en una
base SQLite dentro del propio proyecto y se usan por línea de comandos, con salida
JSON; el mismo binario también sirve con `serve` la API HTTP y su interfaz web: la
interfaz responde en la raíz del servidor y la API conserva sus rutas `/api/...`. Las
rutas, los códigos de estado y los avisos de cambio están en
[`docs/contracts.md`](docs/contracts.md).

Todo lo del producto vive en la carpeta `.tasks-ai/` del proyecto —la base, la
configuración y las instrucciones—, de modo que el proyecto no se ensucia.

La base guarda también las decisiones durables del proyecto, los ADR. Se escriben con el
producto —`adr create`, `adr edit`— y `docs/decisions/` es su exportación generada, que
se versiona para que Git guarde la revisión de cada decisión (`ADR-016`).

## Documentación

Cada documento responde a una pregunta. La estructura del código está en
[`docs/development.md`](docs/development.md).

| Documento | Qué escribe | Qué no |
|---|---|---|
| `README.md` | Qué es el producto y cómo arrancarlo | Estado, historia, decisiones |
| `CONTRIBUTING.md` | Ramas, commits, integración y versiones | Requisitos, arquitectura |
| `AGENTS.md` | Reglas de cada turno y enrutado al detalle | Estado, planes |
| `CHANGELOG.md` | Lo que cambió en cada versión publicada | Planes, estado |
| `PRODUCT.md` | El registro durable del producto para el trabajo de diseño | Requisitos, estado, formas de datos |
| `docs/PRD.md` | Requisitos, alcance y criterios de aceptación | Diseño técnico, estado |
| `docs/architecture.md` | Cómo está construido y por qué | Requisitos, procedimientos |
| `docs/contracts.md` | Formas de datos y API que no se pueden romper | Narrativa de producto |
| `docs/development.md` | Estructura, ejecución y cómo se trabaja | Estado, historia |
| `docs/known-risks.md` | Riesgos vivos, resueltos y deuda aceptada | Requisitos, planes |
| `docs/decisions/` | La exportación generada de las decisiones durables | El registro de lo trivial |
| `docs/web/index.html` | La guía de uso del producto | Requisitos, decisiones |
| `docs/web/landing.html` | La presentación del producto y su entrada a la guía | La referencia de comandos |
| `docs/tasks-ai-brand/` | La identidad visual aprobada y sus normas | El contenido de la guía |
| `.tasks-ai/` | El trabajo: tareas, criterios y evidencia | Documentación |

## Requisitos de entorno

- Go, para el producto.
- Node, para la herramienta con la que se desarrolla el proyecto y para construir la
  interfaz: el binario la lleva incrustada, así que Node no hace falta para compilar
  en Go ni para usar el producto (`ADR-010`).

## Comandos

```bash
go build ./cmd/tasks-ai          # compila el binario
./tasks-ai init                  # instala el producto en el proyecto
./tasks-ai create --title "..."  # crea una tarea
./tasks-ai list                  # devuelve todas las tareas
./tasks-ai adr list              # devuelve las decisiones del proyecto
./tasks-ai adr export --dir docs/decisions  # escribe la exportación versionada de los ADR
./tasks-ai serve                 # arranca la API y la interfaz web
./serve.sh                       # deja ese servidor en segundo plano
```

`tasks-ai --help` y `tasks-ai <comando> --help` son la referencia de cada comando y sus
banderas. El recorrido completo, con ejemplos y respuestas, está en la guía de uso:
[`docs/web/index.html`](docs/web/index.html). Construir la interfaz y refrescar el binario
está en [`docs/development.md`](docs/development.md).

## Integración con los agentes

### Pi

`extensions/pi/` convierte el CLI en herramientas con esquema para el agente de Pi:
`tasks_board`, `tasks_show`, `tasks_create`, `tasks_update`, `tasks_close`,
`tasks_reopen`, `tasks_block`, `tasks_unblock`, `tasks_comment`, `tasks_comment_list` y
`tasks_comment_edit` (`ADR-012`, `ADR-014`). El agente
deja de escribir comandos del tablero y recibe el tablero resumido en cada turno —lo que
está en curso, lo que se puede empezar y su árbol, sin descripciones ni comentarios—,
además del comando `/board`, un bloque junto al editor con lo que está en curso y una
línea de estado con la última tarea cerrada.

El `init` la deja instalada en el proyecto: incrustada en el binario (`ADR-014`), se
extrae a `.tasks-ai/extensions/pi/` junto con el binario en `.tasks-ai/bin/tasks-ai`. El
proyecto la carga desde su propia carpeta, sin depender del repositorio de origen:

```bash
pi install -l .tasks-ai/extensions/pi              # instala el paquete en el proyecto
pi remove -l --approve .tasks-ai/extensions/pi     # lo retira y deja la configuración como estaba
```

Con el paquete instalado se arranca Pi desde la raíz del proyecto: la extensión encuentra
el tablero subiendo desde el directorio de trabajo hasta `.tasks-ai/`, igual que el CLI, y
ejecuta el binario instalado (`.tasks-ai/bin/tasks-ai`) o, si falta, el del `PATH`. En
un proyecto sin tablero las herramientas devuelven el rechazo del producto y la sesión
sigue.

### OpenCode

`extensions/opencode/` registra las mismas herramientas en OpenCode y añade el resumen del
tablero al sistema de cada petición (`ADR-015`). El `init` la deja, como la de Pi, en
`.tasks-ai/extensions/opencode/`, y el proyecto la carga desde su `opencode.jsonc`:

```jsonc
{
  "plugins": ["./.tasks-ai/extensions/opencode"]
}
```

No hay `/board` ni bloque del TUI: el plugin de servidor de OpenCode no los ofrece. El
TODO del arnés no sustituye al tablero: la misión nombra su `tk-N` y el tablero sigue
siendo la única fuente de tareas.

### El agente de delegación

`.pi/agents/tasks-ai-worker.md` es el subagente que trabaja una tarea del tablero: lee su
resultado y su criterio con `tasks_show`, toca solo los archivos que el encargo declara y
devuelve evidencia. Vive en el proyecto, que es donde Pi busca agentes, y viaja en el
repositorio: otra máquina lo tiene al clonar, sin copiar nada.

El modelo del agente no se configura en el repositorio: se elige con `/gentle:models`,
que escribe `.pi/subagents.json`. La sesión de Pi tiene que arrancar después de cambiar
ese archivo, porque el arnés congela su configuración al inicio.
