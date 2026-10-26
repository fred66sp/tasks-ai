# ADR-011: Las superficies del producto hablan inglés

- **Estado:** Aceptada
- **Fecha:** 2026-10-17

## Contexto

El producto tiene dos públicos: la persona, que trabaja desde la interfaz web, y los
agentes, que trabajan por el CLI y consumen su JSON (`O-03`). La ayuda de los comandos,
los mensajes de error de las dos superficies, lo que el CLI avisa por la salida de
errores y el cuerpo `error` de la API se escribían en español, igual que la
documentación y el texto de la interfaz web.

El idioma de los mensajes no estaba en el `PRD.md`: la convención vivía en
`docs/development.md` y en las reglas de documentación, y decía que todo el texto que
ve una persona va en español. Un agente que lee la ayuda y los errores del CLI paga una
traducción en cada llamada, y sus propias interfaces ya están en inglés; los
identificadores del código y los mensajes de commit también lo están.

## Decisión

- La ayuda de todos los comandos, los mensajes de error del CLI, lo que el producto
  avisa por la salida de errores y el cuerpo `error` de las respuestas HTTP se escriben
  en inglés.
- La documentación del repositorio, los comentarios del código, el texto de la interfaz
  web y el contenido que la instalación deja en el proyecto siguen en español: son
  texto para la persona, no mensajes de la superficie.
- El vocabulario que la interfaz web consume —las acciones `creada`, `cerrada` y
  `actualizada` de los avisos— no cambia: es la forma del contrato de eventos de
  `docs/contracts.md` y traducirlo rompería la interfaz.
- No se añade i18n ni una bandera de idioma: cada superficie se escribe en un solo
  idioma, y el que se elige es el de su público.

## Consecuencias

- `RNF-04` del `PRD.md` fija la regla, y las reglas de documentación de `AGENTS.md`
  y `docs/development.md` la recogen para el texto nuevo.
- Los mensajes de error los compone el núcleo y los dos públicos los leen igual: el
  idioma no se decide en cada superficie, se decide al escribir el mensaje.
- Los ejemplos de `docs/web/index.html` reproducen lo que imprime el binario; cuando el
  binario cambia de idioma, la guía se pone al día en el mismo cambio.
- Una frase nueva en una superficie se escribe en inglés; en la documentación y en el
  código, en español.

## Alternativas consideradas

- **Mantener el español y dejar que cada agente traduzca.** Rechazada: la traducción se
  paga en cada llamada, y el público del CLI es el que ya trabaja en inglés.
- **Añadir i18n con una bandera de idioma.** Rechazada: nadie la pidió, duplica cada
  mensaje y sus pruebas, y el producto tiene un público por superficie.
- **Traducir también la interfaz web y la documentación.** Rechazada: el texto de la
  interfaz y la documentación son para la persona que lee en español, y su traducción
  es otro cambio, con su propio alcance.
