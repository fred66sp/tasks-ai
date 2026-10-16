# Operación

Cómo se construye, se prueba y se ejecuta el producto, y qué se ha verificado. Las
formas de datos y de API están en `contracts.md`.

## Construir y ejecutar

| Comando | Qué hace |
|---|---|
| `go build ./cmd/...` | Compila el binario del producto |
| `go test ./...` | Ejecuta las pruebas |
| `go vet ./...` | Comprobaciones estáticas |
| `./tasks-ai init` | Instala el producto en el proyecto: crea `.task-ai/` con la base, el settings y las instrucciones |
| `./tasks-ai serve` | Arranca la API y la interfaz web en la dirección y el puerto del settings: la interfaz responde en la raíz del servidor y la API en sus rutas `/api/...` |
| `./tasks-ai <comando>` | Ejecuta cualquier operación del producto con salida JSON |

La interfaz viaja incrustada en el binario: si se cambia su código, hay que
construirla antes de compilar el binario. Los comandos de su construcción están en
`README.md`, y la decisión, en `ADR-010`.

## La herramienta de desarrollo

```bash
cd task-manager
npm ci                     # instala la herramienta
npm run odd:tk -- stats    # estado del trabajo
npm run server             # interfaz de la herramienta
```

## Comprobaciones antes de cerrar un cambio

- Las pruebas del proyecto, cuando existan.
- Las comprobaciones del contrato de documentación, que se ejecutan a mano hasta que
  haya una suite que las haga suyas.

## Registro de verificaciones

| Fecha | Qué se verificó | Resultado |
|---|---|---|
| 2026-10-16 | Interfaz servida por el binario: el ciclo de crear, ver, cerrar y reabrir una tarea desde el navegador contra el binario real, y las rutas de la API comparadas con las del código anterior | Comprobado: el ciclo completo por la interfaz, y sin diferencias en las rutas documentadas |
| 2026-10-16 | Comprobaciones del contrato de documentación sobre `PRD.md`, `architecture.md` y `contracts.md`: recuentos, medidas, fechas, tablas de progreso, afirmaciones de presente e identificadores ajenos | Sin hallazgos |
| 2026-10-16 | Mapa de identificadores: los requisitos citados desde los documentos de decisión existen en el PRD y ningún documento cita identificadores de otro registro | Sin hallazgos |
| 2026-10-16 | Comprobación de las reglas heredadas en la fuente de la referencia funcional: el cierre de una tarea sincroniza el estado del padre | Confirmado en el código, y descartado por decisión |
