# Riesgos y deuda

Qué riesgo existe, con qué se cubre y qué deuda se acepta. Este documento es un
registro: puede citar hechos con su fecha.

## Riesgos vivos

### RX-1 El diseño va por delante del código

Los requisitos, la arquitectura y los contratos se escriben antes que el código que los
cumple, así que lo documentado puede separarse de lo implementado.

**Cobertura:** el requisito que cambia se corrige en el mismo cambio que lo cambia, y el
cambio se cierra con las comprobaciones del contrato de documentación y su entrada en el
registro de verificaciones de [`operations.md`](operations.md).

### RX-2 Nada comprueba lo escrito

Ninguna herramienta comprueba la coherencia de los documentos: no se compilan ni se
prueban, y su conjunto se sostiene en revisiones manuales. El código no entra en este
riesgo: su suite y sus flujos de comprobación corren solos en cada envío de trabajo, con
un suelo de cobertura que impide bajarla.

**Cobertura:** las comprobaciones del contrato de documentación, hechas a mano, con su
entrada en el registro de verificaciones de [`operations.md`](operations.md).

### RX-3 La auditoría de la interfaz de partida es de segunda mano

El análisis de la interfaz se hizo leyendo el código de otro repositorio y se verificó por
muestreo. Puede haber capacidades o acoplamientos que no se vieran.

**Cobertura:** al integrarla, sus rutas y su ciclo se compararon con el código del que
venía y se ejercitaron contra el binario real, con su entrada en el registro de
verificaciones. Sus pruebas propias no viven en este repositorio, así que no corren con
las nuestras.

### RX-4 El contrato de flujo cita un perfil que no existe aquí

El contrato de flujo nombra `strict_confirmation`, un valor de un perfil de confirmación
que este repositorio no tiene, y la skill `commit-preparation`, que tampoco vive aquí. El
contrato de documentación no cita nada de fuera.

**Cobertura:** deuda aceptada. Se revisa al usarlo y, si estorba, se decide si se adapta o
se retira, con su documento de decisión.

### RX-7 La interfaz lleva un parche sobre un fichero de terceros

`web/public/vendor/toast-anchor.mjs` es código de terceros, minificado, y lleva un parche
nuestro: retirar la escucha de teclado del diálogo cuando se cierra. Es la única vía,
porque esa escucha la registra el propio fichero y no se puede retirar desde fuera. El
parche se pierde el día que se vuelva a copiar el fichero original.

**Cobertura:** esta entrada. Quien reemplace el fichero, la lee antes y vuelve a aplicar el
parche, o cambia de idea sobre el diálogo.

### RX-8 La extensión del agente depende del contrato del CLI

La integración con el agente envuelve el CLI (`ADR-012`): cada herramienta suya ejecuta un
comando y espera su JSON. Hasta ahora el CLI no tenía consumidores fuera del repositorio,
así que cambiar un campo o el significado de un código de salida era un cambio local.

**Cobertura:** `RF-11` y `CA-12` hacen la integración parte del alcance, y el cambio que
toque el CLI comprueba también la extensión y las dos juntas, no solo las pruebas del
producto. Mientras la extensión no exista, esto es una previsión: si el CLI cambia antes,
el consumidor nace desalineado y esta entrada es el aviso para revisarlo.

### RX-9 La copia del agente de delegación puede quedarse atrás

El agente de delegación vive en dos sitios del repositorio —la fuente en
`extensions/pi/agents/` y la copia que Pi lee en `.pi/agents/`—, y nada compara las dos.
Si una cambia y la otra no, el agente trabaja con un contrato viejo sin que nada lo avise.

**Cobertura:** el contrato del agente es corto y cambia poco; cuando cambie, el mismo
cambio actualiza las dos. Ante un comportamiento raro en un encargo, se comparan antes de
buscar otra causa.

### RX-10 La base del tablero es un binario que no se fusiona

La base viaja en el repositorio para llevarla a otra máquina. Un binario en Git no se
fusiona: dos máquinas que muevan el tablero producen un conflicto que no se resuelve
eligiendo un lado sin perder las tareas del otro, y cada cambio de tarea engorda la
historia del repositorio.

**Cobertura:** el tablero se mueve desde una máquina cada vez; antes de integrar se
comprueba que la otra no tiene tareas nuevas, y una tarea creada en la máquina que se
queda se vuelve a crear a mano si el conflicto ya ocurrió. Cuando moleste de verdad, la
salida es un volcado de texto exportable, no la base binaria.

## Resueltos

### RX-5 El catálogo de etiquetas se reimplementaba sin pruebas heredadas

El catálogo con color, descripción y aviso de uso vivía en el servidor anterior, y se
dudaba de su cascada al renombrar o borrar, que no tenía pruebas nuestras.

**Qué resultó:** la fase que lo implementó escribió sus pruebas, y la duda quedó
respondida por ellas: renombrar una etiqueta no toca ninguna tarea, borrar una avisa de
las tareas que la usaban, y cambiar las etiquetas de una tarea no toca las de otra.

**Lección:** una cobertura que se promete para una fase futura hay que releerla cuando esa
fase llega, o el registro sigue anunciando un hueco que ya no existe.

### RX-6 El trabajo vivía sólo en esta máquina

El registro aceptaba como deuda que el repositorio no estaba en ningún remoto y que el
trabajo vivía sólo en local.

**Qué resultó:** dejó de ser cierto cuando el proyecto pasó a tener su remoto y el trabajo
a vivir allí.

**Lección:** la entrada dejó de ser cierta y nadie la corrigió, así que el registro mintió
hasta que se volvió a leer entero. Lo que deja de ser cierto se corrige en el mismo
cambio que lo dejó obsoleto, o se retira.

## Deuda aceptada

- La interfaz de partida trae una pieza sin uso: no consume `GET /api/status`. Sigue ahí, y
  todavía no se ha decidido si se retira o se usa.
