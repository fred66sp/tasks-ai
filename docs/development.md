# Desarrollo

Cómo está organizado el repositorio y cómo se trabaja en él. Las formas de datos y de
API están en `contracts.md`; el diseño, en `architecture.md`.

## Estructura

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto: requisitos, decisiones, diseño, contratos y riesgos |
| `docs/decisions/` | Un documento por decisión durable |
| `cmd/tasks-ai/` | El punto de entrada del binario |
| `internal/settings/` | Los ajustes del proyecto: su archivo en la carpeta del producto y sus valores por omisión |
| `internal/store/` | La base SQLite: su esquema versionado y lo que guarda de las tareas |
| `internal/core/` | Las reglas del producto: crear tareas, darles identidad y referencia, y leerlas |
| `internal/install/` | La instalación en un proyecto: su carpeta con la base, los ajustes y las instrucciones |
| `internal/cli/` | La línea de comandos: comandos, banderas y respuesta en JSON |
| `internal/` | El núcleo del producto y sus superficies |
| `web/` | La interfaz web: el código de la de partida en `web/src/`, lo que el navegador descarga en `web/public/` —con el artefacto construido versionado— y `web/embed.go`, que la incrusta en el binario |
| `.task-ai/` | El tablero del proyecto: la base de tareas, sus ajustes y sus instrucciones |
| `.pi/` | El estado de Pi del proyecto, versionado: el paquete instalado en `.pi/settings.json` y el agente de delegación en `.pi/agents/`, cuyo modelo se elige con `/gentle:models` ([`README.md`](../README.md)) |
| `extensions/pi/` | La integración con el agente de Pi: herramientas en TypeScript sobre el CLI del producto, con sus pruebas propias |
| `skills/` | Las skills del proyecto, copiadas del taller: Go, TypeScript y las locales |

Las pruebas de la extensión se ejecutan con `cd extensions/pi && npm test`, que usa el runner
de Node y construye el binario si falta. Ninguna prueba escribe en el tablero del
repositorio: cada una monta su proyecto temporal y lo borra al terminar.

La extensión no viaja en el binario: es un paquete de Pi que vive en el repositorio y se
instala en el proyecto con los comandos del host, que dejan su ruta relativa en el archivo
de ajustes de Pi del proyecto, versionado (`.pi/settings.json`):

```bash
pi install -l ./extensions/pi              # instala el paquete en el proyecto
pi remove -l --approve ./extensions/pi     # lo retira y deja la configuración como estaba
```

Con el paquete instalado se arranca Pi desde la raíz del proyecto, y la extensión
encuentra el tablero subiendo desde el directorio de trabajo hasta `.task-ai/`, igual que
el CLI, ejecutando el binario del proyecto (`bin/tasks-ai`) o, si falta, el del `PATH`. Sus
herramientas y lo que llega al modelo están en `contracts.md`; qué se gana con ellas, en
el `README.md`; la forma y el riesgo, en `ADR-012`.

## Fronteras del código

- El núcleo contiene las reglas y el acceso a datos; no conoce comandos ni rutas.
- Cada superficie traduce su idioma al del núcleo: el CLI con sus comandos y su JSON,
  el servidor HTTP con sus rutas y sus códigos.
- La interfaz web es la de partida, ajustada a la API del núcleo. Vive en `web/`, la
  construye su propia herramienta de desarrollo y viaja incrustada en el binario:
  `serve` la reparte en la raíz y la API conserva sus rutas (`ADR-010`).
- La base solo la toca el núcleo.

## Convenciones

- Un módulo de Go en la raíz del repositorio.
- Capas sencillas con cableado manual por constructor, árbol de comandos con Cobra y
  sin Viper, y acceso directo a SQLite con migraciones aditivas propias (`ADR-007`,
  `ADR-009`).
- Sin ORM y sin capas de indirección sin justificación; una capa nueva es una decisión,
  no una comodidad.
- Los identificadores del código van en inglés; la documentación, los comentarios y el
  texto de la interfaz web, en español; la ayuda, los errores y los avisos del
  producto, en inglés (`ADR-011`).
- Los mensajes de commit van en inglés, con el formato del contrato de flujo.
- Los identificadores de los registros son de solo añadir: requisitos en `PRD.md`,
  riesgos en `known-risks.md` y decisiones en sus documentos.

## Cómo se trabaja

1. La tarea en el tablero antes de tocar archivos.
2. Un cambio pequeño se hace y se verifica; un cambio grande va por fases, con la
   documentación al día en cada una.
3. Antes de cerrar: las pruebas del proyecto y las comprobaciones del contrato de
   documentación.
4. Un cambio de diseño se documenta antes de implementarlo, y su documento de decisión
   explica qué se descartó.

Los comandos del día a día son objetivos de `make`, con su equivalente de `go`:

```bash
make build      # go build -o bin/tasks-ai ./cmd/tasks-ai
make test       # go test ./...
make test-race  # go test -race ./...
make lint       # golangci-lint run ./...
make fmt        # golangci-lint fmt ./...
```

## Lo que no se hace

- No se edita la base de tareas con SQL.
- No se modifican los proyectos de referencia.
- No se añade una capa, una dependencia o una abstracción «por si acaso».
