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
| `AGENTS.md` | Mapa de navegación a las fuentes canónicas | Reglas, estado, planes |
| `CHANGELOG.md` | Lo que cambió en cada versión publicada | Planes, estado |
| `PRODUCT.md` | El registro durable del producto para el trabajo de diseño | Requisitos, estado, formas de datos |
| `DESIGN.md` | Cómo se aplica la identidad visual en el código: tokens, forma y componentes | Identidad aprobada, requisitos |
| `docs/PRD.md` | Requisitos, alcance y criterios de aceptación | Diseño técnico, estado |
| `docs/architecture.md` | Cómo está construido y por qué | Requisitos, procedimientos |
| `docs/contracts.md` | Formas de datos y API que no se pueden romper | Narrativa de producto |
| `docs/development.md` | Estructura, ejecución y cómo se trabaja | Estado, historia |
| `docs/known-risks.md` | Riesgos vivos, resueltos y deuda aceptada | Requisitos, planes |
| `docs/decisions/` | La exportación generada de las decisiones durables | El registro de lo trivial |
| `docs/web/index.html` | La guía de uso del producto | Requisitos, decisiones |
| `docs/web/landing.html` | La presentación del producto y su entrada a la guía | La referencia de comandos |
| `docs/web/design-system.html` | La referencia navegable del sistema visual: muestras, estados y medidas | El contrato y los valores: viven en `DESIGN.md` y en el CSS |
| `docs/web/demo.html` | La demostración en bucle del tablero: un agente crea una subtarea, la recorre por sus estados y cierra la épica | El producto: es una réplica estática con datos de ejemplo que no escribe nada |
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
./serve-web.sh                   # deja en segundo plano el servidor de la documentación
```

`tasks-ai --help` y `tasks-ai <comando> --help` son la referencia de cada comando y sus
banderas. El recorrido completo, con ejemplos y respuestas, está en la guía de uso:
[`docs/web/index.html`](docs/web/index.html). Construir la interfaz y refrescar el binario
está en [`docs/development.md`](docs/development.md).

## Integración con los agentes

El agente habla con el producto por el subcomando `tasks-ai aif`: una petición JSON por
la entrada estándar y exactamente una respuesta JSON por la salida estándar. El contrato
que el `init` copia al proyecto —[`internal/install/agent.md`](internal/install/agent.md)—
explica el vocabulario, el descubrimiento y los códigos de salida; el agente lo lee del
propio proyecto, sin paquetes ni plugins del host: las extensiones por host de la v0.1.0
quedaron retiradas (`ADR-021`).

Para actualizar una instalación, `tasks-ai upgrade` renueva el binario y los recursos
gestionados desde el ejecutable que lo invoca, conserva los datos y las personalizaciones
y escribe `.tasks-ai/VERSION` al completo; `tasks-ai version` devuelve la versión del
producto.

### El agente de delegación

`.pi/agents/tasks-ai-worker.md` es el subagente que trabaja una tarea del tablero: lee su
resultado y su criterio con `tasks_show`, toca solo los archivos que el encargo declara y
devuelve evidencia. Vive en el proyecto, que es donde Pi busca agentes, y viaja en el
repositorio: otra máquina lo tiene al clonar, sin copiar nada.

El modelo del agente no se configura en el repositorio: se elige con `/gentle:models`,
que escribe `.pi/subagents.json`. La sesión de Pi tiene que arrancar después de cambiar
ese archivo, porque el arnés congela su configuración al inicio.

## Licencia

Distribuido bajo la licencia [MIT](LICENSE). Los textos de las skills de terceros y
derivadas conservan sus licencias y atribuciones propias en `skills/LICENSES/`.
