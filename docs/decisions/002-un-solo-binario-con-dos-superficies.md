# ADR-002: Un solo binario con dos superficies

- **Estado:** Aceptada
- **Fecha:** 2026-10-16

## Contexto

El producto tiene dos formas de uso: los agentes operan por línea de comandos y
consumen datos en JSON, y la persona usa una interfaz web. La referencia funcional
resuelve las dos cosas con un único binario: el mismo programa ofrece los comandos y
levanta el servidor de la interfaz.

El Product Owner confirmó esa misma forma: el CLI y el servidor son el mismo
programa.

## Decisión

- Un único programa con dos superficies sobre el mismo núcleo de negocio: el CLI y el
  servidor de la interfaz.
- El servidor se levanta desde el propio CLI, con un comando.
- Ninguna regla de negocio vive en una sola de las dos superficies.
- La salida JSON es de primera clase: es lo que consumen los agentes.
- Toda operación del producto se puede ejecutar desde el CLI.

## Consecuencias

- Una sola instalación y un solo artefacto.
- Las reglas de negocio tienen un único lugar donde vivir; la interfaz y el CLI no
  las duplican.
- El binario arranca en modo comando o en modo servidor según lo que se le pida.
- Requisitos afectados: `O-02`, `O-03`, `RF-01`, `RF-02`, `RF-10`, `CA-01`, `CA-11`.

## Alternativas consideradas

- **Dos programas, uno para el CLI y otro para el servidor.** Rechazada: dos
  instalaciones y dos sitios donde duplicar las reglas.
- **CLI como cliente HTTP del servidor.** Rechazada: cualquier operación exigiría el
  servidor arrancado, incluso dentro de un guion.
- **Solo CLI, sin interfaz.** Rechazada: la interfaz existente forma parte del
  producto.
