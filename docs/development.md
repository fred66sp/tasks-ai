# Desarrollo

Cómo está organizado el repositorio y cómo se trabaja en él. Las formas de datos y de
API están en `contracts.md`; el diseño, en `architecture.md`.

## Estructura

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto: requisitos, decisiones, diseño, contratos y riesgos |
| `docs/decisions/` | Un documento por decisión durable |
| `cmd/` | El punto de entrada del binario |
| `internal/` | El núcleo del producto y sus superficies |
| `task-manager/` | La herramienta con la que se desarrolla el proyecto, con su tablero |

## Fronteras del código

- El núcleo contiene las reglas y el acceso a datos; no conoce comandos ni rutas.
- Cada superficie traduce su idioma al del núcleo: el CLI con sus comandos y su JSON,
  el servidor HTTP con sus rutas y sus códigos.
- La interfaz web es la de partida, ajustada a la API del núcleo.
- La base solo la toca el núcleo.

## Convenciones

- Un módulo de Go en la raíz del repositorio.
- Sin ORM y sin capas de indirección sin justificación; una capa nueva es una decisión,
  no una comodidad.
- Los identificadores del código van en inglés; el texto que ve una persona, en
  español.
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

## Lo que no se hace

- No se edita la base de tareas con SQL.
- No se modifican los proyectos de referencia.
- No se añade una capa, una dependencia o una abstracción «por si acaso».
