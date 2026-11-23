# tasks-ai: mapa del repositorio

Este archivo dice dónde vive cada fuente canónica y qué leer antes de cada encargo. No
guarda el contenido: lo enlaza.

## Qué se busca → dónde vive

| Qué se busca | Dónde vive |
|---|---|
| Qué es el producto y cómo se arranca | [`README.md`](README.md) → «tasks-ai», «Requisitos de entorno», «Comandos» |
| Memoria de sesión: decisiones, aprendizajes, checkpoints y preferencias | Montaje db-ai `tasks-ai` (rw) sobre `.agent-memory/memory.db`; se abre con `acm start`, se guarda con `acm decision|learning|discovery|bugfix|preference`, se consulta con `acm recall` antes de investigar o decidir sobre algo posiblemente ya tratado —lo recuperado es contexto, no hecho: se contrasta con el repo antes de actuar sobre él— y se cierra con `acm close` (adaptador: `tools/acm.py` del ejemplo `agent-continuity` en database-virtual-filesystem) |
| Qué no hace el producto | [`docs/web/index.html`](docs/web/index.html) → «Lo que no hace» |
| Qué documento es dueño de cada dato | [`README.md`](README.md) → «Documentación» |
| Ramas, commits, verificación, integración, versiones y operaciones que piden autorización | [`CONTRIBUTING.md`](CONTRIBUTING.md) → «Modelo», «Ramas», «Commits», «Verificación», «Integración», «CHANGELOG», «Versiones y etiquetas», «Operaciones que exigen autorización» |
| Estructura del repositorio y mapa de paquetes | [`docs/development.md`](docs/development.md) → «Estructura» |
| Mobilia común de las páginas de documentación y su publicabilidad | [`docs/web/assets/`](docs/web/assets/); la decisión, en [`docs/decisions/README.md`](docs/decisions/README.md) → ADR-018; la guarda, en `docs_web_test.go` |
| Cómo se trabaja y qué se comprueba antes de cerrar | [`docs/development.md`](docs/development.md) → «Cómo se trabaja» |
| Ejecutar el producto y recorrer sus comandos | [`docs/development.md`](docs/development.md) → «Ejecutar»; [`docs/web/index.html`](docs/web/index.html) → «Empezar en un proyecto», «Las órdenes» |
| Requisitos, alcance y criterios de aceptación | [`docs/PRD.md`](docs/PRD.md) → «3. Alcance aprobado», «4. Requisitos aprobados», «5. Criterios de aceptación» |
| Modelo de tarea: resultado, criterio, notas y comentarios | [`docs/PRD.md`](docs/PRD.md) → «1. Propósito»; [`docs/contracts.md`](docs/contracts.md) → «1. Modelo de datos» |
| Cómo está construido y por qué | [`docs/architecture.md`](docs/architecture.md) → «1. Vista general», «2. Capas y fronteras», «4. Invariantes», «11. De qué decisión sale cada pieza» |
| Forma de datos y de API | [`docs/contracts.md`](docs/contracts.md) → «1. Modelo de datos», «2. Salida en JSON», «3. API HTTP», «4. Eventos», «5. Referencias», «6. Settings», «7. Integración con el agente», «8. El texto de las tareas» |
| Decisiones durables: cómo se escriben y se exportan | [`docs/development.md`](docs/development.md) → «Los ADR»; [`docs/decisions/README.md`](docs/decisions/README.md) → «Convenciones» |
| Riesgos vivos, resueltos y deuda aceptada | [`docs/known-risks.md`](docs/known-risks.md) → «Riesgos vivos», «Resueltos», «Deuda aceptada» |
| Trabajo del repositorio: crear, mover, bloquear y cerrar tareas | [`.tasks-ai/AGENTS.md`](.tasks-ai/AGENTS.md) → «Cómo se usa» |
| Referencias visibles y prefijo del proyecto | [`.tasks-ai/AGENTS.md`](.tasks-ai/AGENTS.md) → «Las referencias»; valor vivo en [`.tasks-ai/settings.json`](.tasks-ai/settings.json) |
| Etiquetas: tipos, estructura, ámbito, catálogo y cambios del contrato | [`.tasks-ai/CONTRATO_ETIQUETAS.md`](.tasks-ai/CONTRATO_ETIQUETAS.md) → «1. Las clases y su convivencia», «2. Asignación correcta a una tarea», «3. Gestión del catálogo», «4. Cómo se cambia este contrato» |
| Guía de uso: modelo, ciclo de vida, recetas, API y errores | [`docs/web/index.html`](docs/web/index.html) → «El modelo de datos», «El ciclo de vida», «Recetas», «La interfaz web», «La API HTTP», «Los agentes», «La configuración», «Errores y qué significan» |
| Presentación del producto | [`docs/web/landing.html`](docs/web/landing.html) |
| Demostración del tablero: datos de ejemplo y el ciclo de una épica | [`docs/web/demo.html`](docs/web/demo.html) → «Así se ve un agente gestionando una épica», «Qué es real y qué es simulado» |
| Sistema visual: contrato máquina-legible, prosa y muestras | [`design-system.json`](design-system.json); [`DESIGN.md`](DESIGN.md) → «Colors», «Typography», «Layout», «Components», «Do's and Don'ts»; [`docs/web/design-system.html`](docs/web/design-system.html) → «Un tablero que ordena el trabajo.» |
| Iconos del producto y la documentación: origen, licencia, hoja compartida y selección | [`web/public/icons/`](web/public/icons/); [`DESIGN.md`](DESIGN.md) → «Iconography»; [`docs/web/design-system.html`](docs/web/design-system.html) → «Iconografía» |
| Identidad visual aprobada | [`docs/tasks-ai-brand/README.md`](docs/tasks-ai-brand/README.md); [`docs/tasks-ai-brand/AGENT-INTEGRATION.md`](docs/tasks-ai-brand/AGENT-INTEGRATION.md) |
| Usuarios, propósito y principios de producto | [`PRODUCT.md`](PRODUCT.md) → «Users», «Product Purpose», «Product Principles» |
| Qué cambió en cada versión | [`CHANGELOG.md`](CHANGELOG.md) → «[Sin publicar]» |
| Skills: cuáles hay, de dónde vienen, cómo se usan y cómo se actualizan | [`skills/README.md`](skills/README.md) → «SKILLS-2 Cuáles hay y de dónde vienen», «SKILLS-3 Cómo se usan», «SKILLS-4 Cómo se actualizan» |
| Qué skill cargar para Go | [`skills/golang-how-to/SKILL.md`](skills/golang-how-to/SKILL.md); catálogo por categoría en [`by-category.md`](skills/golang-how-to/references/by-category.md) y límites en [`disambiguation.md`](skills/golang-how-to/references/disambiguation.md) |
| Agente de delegación: encargo, límites y devolución | [`.pi/agents/tasks-ai-worker.md`](.pi/agents/tasks-ai-worker.md) → «Antes de tocar nada», «Qué puedes tocar», «Límites», «Qué devuelves» |
| Integración con Pi y OpenCode | [`README.md`](README.md) → «Integración con los agentes», «Pi», «OpenCode»; [`extensions/pi/`](extensions/pi/), [`extensions/opencode/`](extensions/opencode/); carga en [`opencode.jsonc`](opencode.jsonc) y [`.pi/settings.json`](.pi/settings.json) |
| Instrucciones y contrato que el `init` copia | [`internal/install/agent.md`](internal/install/agent.md) → «Cómo se usa»; [`internal/install/contract_tags.md`](internal/install/contract_tags.md) → «1. Las clases y su convivencia» |
| Integración continua | [`.github/workflows/tests.yml`](.github/workflows/tests.yml) → «Pruebas»; [`.github/workflows/lint.yml`](.github/workflows/lint.yml) → «Linter»; [`.github/workflows/security.yml`](.github/workflows/security.yml) → «Seguridad» |
| Objetivos de `make` y qué hace cada uno | [`Makefile`](Makefile) |
| Dependencias del producto y de la interfaz | [`go.mod`](go.mod), [`go.sum`](go.sum), [`extensions/pi/package.json`](extensions/pi/package.json), [`extensions/opencode/package.json`](extensions/opencode/package.json), [`web/package.json`](web/package.json) |

## Encargo → qué leer antes → con qué se cierra

| Encargo | Qué leer antes | Con qué se cierra |
|---|---|---|
| Trabajar una tarea del tablero | [`.tasks-ai/AGENTS.md`](.tasks-ai/AGENTS.md) → «Cómo se usa», «Las referencias»; [`docs/development.md`](docs/development.md) → «Cómo se trabaja» | El criterio escrito en la propia tarea, observado; el cierre y su motivo, en [`.tasks-ai/AGENTS.md`](.tasks-ai/AGENTS.md) → «Cómo se usa» |
| Cambiar código Go | [`skills/golang-how-to/SKILL.md`](skills/golang-how-to/SKILL.md) y las skills que señale; [`docs/development.md`](docs/development.md) → «Fronteras del código», «Convenciones» | Las comprobaciones de [`docs/development.md`](docs/development.md) → «Cómo se trabaja»; el contrato, si cambia la superficie, en [`docs/contracts.md`](docs/contracts.md) |
| Cambiar una extensión del agente | [`docs/contracts.md`](docs/contracts.md) → «7. Integración con el agente»; [`skills/typescript-idiomatic-style/SKILL.md`](skills/typescript-idiomatic-style/SKILL.md); [`skills/typescript-testing-and-verification/SKILL.md`](skills/typescript-testing-and-verification/SKILL.md) | Las pruebas y el typecheck de la extensión, en [`docs/development.md`](docs/development.md) → «Estructura»; la integración continua, en [`.github/workflows/tests.yml`](.github/workflows/tests.yml) → «Pruebas» |
| Cambiar la interfaz o el sistema visual | [`DESIGN.md`](DESIGN.md) → «Components», «Do's and Don'ts»; [`design-system.json`](design-system.json); [`docs/web/design-system.html`](docs/web/design-system.html) | La comprobación de la interfaz del [`Makefile`](Makefile) y la prueba de tokens de [`design-tokens.test.js`](web/src/task-manager/design-tokens.test.js) |
| Escribir o cambiar una decisión durable | [`docs/development.md`](docs/development.md) → «Los ADR»; [`docs/decisions/README.md`](docs/decisions/README.md) → «Convenciones»; [`skills/adr-software/SKILL.md`](skills/adr-software/SKILL.md) | La comprobación de deriva de la exportación, en [`docs/development.md`](docs/development.md) → «Los ADR», y la puerta de [`skills/adr-software/SKILL.md`](skills/adr-software/SKILL.md) |
| Cambiar un requisito, un contrato o un riesgo | [`docs/PRD.md`](docs/PRD.md) → «4. Requisitos aprobados»; [`docs/contracts.md`](docs/contracts.md) → «1. Modelo de datos»; [`docs/known-risks.md`](docs/known-risks.md) → «Riesgos vivos» | La documentación viva y las comprobaciones de [`docs/development.md`](docs/development.md) → «Cómo se trabaja»; la prueba de enlaces y títulos de [`docs_test.go`](docs_test.go) |
| Delegar una tarea a otro agente | [`.pi/agents/tasks-ai-worker.md`](.pi/agents/tasks-ai-worker.md) → «Antes de tocar nada», «Límites», «Qué devuelves»; la tarea del tablero | La evidencia que devuelve, contrastada con el criterio de la tarea |

## Autoridad

Este mapa no es fuente de nada: enlaza. Si una ruta, un título o una afirmación de este
archivo contradice a la fuente que enlaza, manda la fuente y se corrige aquí. La prueba de
[`docs_test.go`](docs_test.go) comprueba que cada documento tiene título y que sus enlaces
resuelven.
