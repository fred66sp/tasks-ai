# ADR-012: La integración con el agente vive fuera del binario

- **Estado:** Aceptada
- **Fecha:** 2026-10-17

## Contexto

El producto tiene dos públicos: la persona, que trabaja desde la interfaz web, y los
agentes, que trabajan por el CLI y consumen su JSON (`O-03`). Hoy un agente opera el
tablero escribiendo comandos: tiene que recordar la bandera de cada operación, componer
las referencias a mano y volver a leer el estado después de cada cambio. El resultado
depende de que se acuerde de todo a la vez, y el tablero se queda atrás mientras el
trabajo avanza.

El CLI ya es una superficie de agente: responde JSON por la salida estándar, distingue el
fallo del mal uso con códigos de salida propios y encuentra el proyecto subiendo desde el
directorio de trabajo hasta `.task-ai/` (`RF-01`, `RF-04`, `RN-03`). Lo que falta no es
una superficie nueva, sino que el agente llegue a ella con parámetros validados y sin
tener que escribir comandos.

`RF-11` aprobó la integración y `CA-12` fijó su criterio de aceptación. Esta decisión
elige su forma y su sitio.

## Decisión

- La integración es una **extensión del agente Pi**: un módulo en TypeScript que registra
  herramientas y las ejecuta invocando el CLI del producto.
- Vive en **`extensions/pi/`**, versionada en este repositorio y **fuera del binario**: no
  entra en el producto instalable y no añade dependencias a Go.
- **No toca SQLite y no reimplementa reglas**: cada herramienta ejecuta el comando que ya
  existe y devuelve su JSON; los rechazos del núcleo llegan tal cual, sin reinterpretar.
- El proyecto lo encuentra **el CLI**, no la extensión: ella trabaja en el directorio de
  la sesión y deja que el producto suba hasta `.task-ai/`. Tampoco compone referencias.
- **Una herramienta por operación que ya exista en el CLI**, y ninguna que no exista. El
  vocabulario del tablero no se amplía desde la extensión.
- Las pruebas usan el runner que trae Node (`node --test`), que ejecuta TypeScript sin
  compilar en el Node que el proyecto ya necesita; no se añade ningún framework.
- **MCP se descarta** mientras no haya otro cliente que lo pida. La condición que reabre
  la decisión es esa, y no una preferencia de forma.

## Consecuencias

- La extensión es un **consumidor del contrato del CLI**: cambiarlo deja de ser un cambio
  local, y quien lo cambie tiene que comprobar la extensión además de las pruebas del
  producto (`RX-8`).
- El producto no gana superficie: `RF-01` a `RF-10` siguen siendo suyos. `RF-11` es del
  repositorio, y el binario queda como la única superficie que se instala.
- La extensión necesita Node; el producto no. Quien use el producto sin agente no instala
  nada más.
- Si más adelante otro cliente pide MCP, será una segunda puerta al mismo núcleo: habrá
  que decidir si acompaña a la extensión o la sustituye, y esta decisión es el sitio donde
  se anota.

## Alternativas consideradas

- **Un servidor MCP dentro del binario.** Rechazada por ahora: añade protocolo, superficie
  y pruebas al producto para servir a un solo cliente, cuando el CLI ya es una API de
  agente. La condición que la reabriría está en la decisión.
- **Instrucciones o una skill en texto.** Es lo que hay hoy, y es lo que no funciona: no
  valida parámetros, no trae el estado del tablero y depende de que el agente recuerde los
  comandos.
- **Herramientas que escriban en la base directamente.** Rechazada: rompe la frontera de
  `RF-02` —la base solo la toca el núcleo— y duplicaría cada regla en dos lenguajes.
- **Que el producto traiga la integración dentro.** Rechazada: mete Node en un binario Go
  que hoy no lo necesita y rompe la promesa de que el producto se instala en una carpeta
  sin dependencias.
