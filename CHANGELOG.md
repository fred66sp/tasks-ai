# Registro de cambios

Cambios relevantes del proyecto. El formato sigue las recomendaciones de *Keep a
Changelog* y el versionado es semántico.

## [Sin publicar]

### Añadido

- **Primera rebanada del producto**: se instala en la carpeta del proyecto, guarda su
  estado en una base SQLite con esquema versionado, y crea y lee tareas desde la línea
  de comandos, en JSON.
- **Cambio de una tarea**: la línea de comandos cambia el título, la descripción, las
  notas, el responsable y la prioridad de una tarea guardada, y rechaza la prioridad que
  se salga de su rango.
- **Documentación del producto**: requisitos y criterios de aceptación en
  [`docs/PRD.md`](docs/PRD.md); decisiones durables en
  [`docs/decisions/`](docs/decisions/); diseño en
  [`docs/architecture.md`](docs/architecture.md); y formas de datos, API, eventos,
  referencias y configuración en [`docs/contracts.md`](docs/contracts.md).
- **Reglas de trabajo**: el contrato de documentación y el contrato de flujo, y las
  instrucciones para agentes en [`AGENTS.md`](AGENTS.md).
- **Estructura, operación y riesgos**: [`docs/development.md`](docs/development.md),
  [`docs/operations.md`](docs/operations.md) y
  [`docs/known-risks.md`](docs/known-risks.md), junto con este registro y el
  [`README.md`](README.md).
- **Auditoría de las referencias**: análisis de la referencia funcional y de la
  interfaz de partida, con sus hallazgos registrados en el tablero de trabajo.
- **Herramienta de desarrollo** instalada en `task-manager/`, con su tablero y su
  catálogo de etiquetas.
