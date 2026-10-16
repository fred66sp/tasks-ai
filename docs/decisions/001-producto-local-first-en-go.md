# ADR-001: Producto local-first en Go, de un solo proyecto

- **Estado:** Aceptada
- **Fecha:** 2026-10-16

## Contexto

El proyecto vive en un taller orientado al desarrollo en Go. Existe un producto
anterior, `project-task-manager`, escrito en Node y Vue, que ya resolvió la
extracción de un gestor de tareas local desde una aplicación mayor; de él se
conserva la interfaz. La referencia funcional es Tacks, un gestor de tareas en Rust,
de binario único y SQLite.

El Product Owner fijó el sustrato y el alcance: local, escrito en Go y de un solo
proyecto.

## Decisión

- El producto se escribe en Go.
- Es local-first: los datos y el servicio viven en la máquina de la persona, sin
  servicio remoto.
- Las tareas viven en una base SQLite.
- Atiende a un único proyecto: sin vista conjunta y con un único ámbito de
  configuración.

## Consecuencias

- El producto no registra proyectos, no agrega vistas y no resuelve precedencia
  entre ajustes de ámbitos distintos.
- Quedan fuera cualquier contrato de servicio remoto, cuentas o sincronización.
- El alcance del producto es el de un proyecto y sus tareas.
- Requisitos afectados: `O-01`, `O-03`, `NO-03`, `RNF-02`.

## Alternativas consideradas

- **Continuar el producto existente en Node y Vue.** Rechazada: el sustrato elegido
  es Go.
- **Producto multiproyecto, con vista conjunta y configuración global.** Se planteó
  y el Product Owner la descartó por complicación frente al producto que necesita.
- **Servicio remoto o multiusuario.** No se contempla: la condición local-first lo
  excluye.
