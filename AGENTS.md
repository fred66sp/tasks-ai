# tasks-ai — reglas del repositorio

Reglas que valen en cada turno dentro de este repositorio. Esto es un router: el
detalle canónico vive en los documentos enlazados y no se copia aquí.

## Contratos

- [`docs/CONTRATO_DOCUMENTACION.md`](docs/CONTRATO_DOCUMENTACION.md): qué escribe cada
  documento y dónde vive cada dato.
- [`docs/CONTRATO_FLUJO_GIT_VERSIONADO.md`](docs/CONTRATO_FLUJO_GIT_VERSIONADO.md):
  ramas, commits, verificación, integración, CHANGELOG y versionado.

## Tablero del trabajo

El trabajo se registra en la herramienta de desarrollo, instalada en `task-manager/`.
Toda llamada pasa por su guardia, con base explícita y salida JSON:

```bash
cd task-manager
npm run odd:tk -- stats
npm run odd:tk -- ready --limit 1
npm run odd:tk -- show tk-XXXX
```

- La base de tareas vive en `task-manager/tk/.tacks/tacks.db`. No se edita con SQL:
  solo por su CLI.
- Ciclo: crear o seleccionar la tarea antes de tocar archivos, marcarla en curso,
  mantener las notas y los comentarios al día y cerrarla tras verificar.
- No se registra en el tablero el estado de Git ni el del proceso de revisión: cada
  uno vive donde vive.

## Qué se toca y qué no

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto |
| `task-manager/` | La herramienta con la que se desarrolla; no es código del producto |
| `cmd/`, `internal/` | El código del producto, en Go |

- Los proyectos de referencia están fuera de este repositorio: se leen, no se tocan.
- No se traduce la referencia funcional ni se copia su código.

## Al cambiar algo

| Cambio | Dónde se escribe |
|---|---|
| Requisito nuevo o modificado | `docs/PRD.md` |
| Decisión durable | Un documento nuevo en `docs/decisions/` |
| Forma de datos o de API | `docs/contracts.md` |
| Diseño | `docs/architecture.md` |
| Riesgo nuevo | `docs/known-risks.md` |
| Comandos o estructura | `README.md` o `docs/development.md` |

Y siempre, antes de cerrar el cambio, las comprobaciones del contrato de
documentación.

## Idioma

- Documentación, código, comentarios e interfaz: español.
- Mensajes de commit: inglés.
