# Riesgos y deuda

Qué riesgo existe, con qué se cubre y qué deuda se acepta. Este documento es un
registro: puede citar hechos con su fecha.

## Riesgos vivos

### RX-1 El diseño va por delante del código

Los requisitos, la arquitectura y los contratos se escriben antes que el código que los
cumple, así que lo documentado puede separarse de lo implementado.

**Cobertura:** el requisito que cambia se corrige en el mismo cambio que lo cambia, y el
cambio se cierra con las comprobaciones de [`development.md`](development.md) y la
evidencia en la tarea.

### RX-2 Nada comprueba lo escrito

Ninguna herramienta comprueba la coherencia de los documentos: sus enlaces y sus títulos
los revisa una prueba, pero el resto se sostiene en revisiones manuales. El código no entra en este
riesgo: su suite y sus flujos de comprobación corren solos en cada envío de trabajo, con
un suelo de cobertura que impide bajarla.

**Cobertura:** las comprobaciones de [`development.md`](development.md), hechas a mano, y
la prueba que revisa enlaces y títulos de los documentos.

### RX-3 La auditoría de la interfaz de partida es de segunda mano

El análisis de la interfaz se hizo leyendo el código de otro repositorio y se verificó por
muestreo. Puede haber capacidades o acoplamientos que no se vieran.

**Cobertura:** al integrarla, sus rutas y su ciclo se compararon con el código del que
venía y se ejercitaron contra el binario real; la evidencia vive en la tarea. Sus pruebas propias no viven en este repositorio, así que no corren con
las nuestras.

### RX-7 La interfaz lleva un parche sobre un fichero de terceros

`web/public/vendor/toast-anchor.mjs` es código de terceros, minificado, y lleva un parche
nuestro: retirar la escucha de teclado del diálogo cuando se cierra. Es la única vía,
porque esa escucha la registra el propio fichero y no se puede retirar desde fuera. El
parche se pierde el día que se vuelva a copiar el fichero original.

**Cobertura:** esta entrada. Quien reemplace el fichero, la lee antes y vuelve a aplicar el
parche, o cambia de idea sobre el diálogo.

### RX-8 Las extensiones del agente dependen del contrato del CLI

Las integraciones con los agentes envuelven el CLI (`ADR-012`, `ADR-015`): cada
herramienta suya ejecuta un comando y espera su JSON. Hasta ahora el CLI no tenía consumidores fuera del repositorio,
así que cambiar un campo o el significado de un código de salida era un cambio local.

**Cobertura:** `RF-11` y `CA-12` hacen la integración parte del alcance, y el cambio que
toque el CLI comprueba también las extensiones y las dos juntas, no solo las pruebas del
producto. Mientras la extensión no exista, esto es una previsión: si el CLI cambia antes,
el consumidor nace desalineado y esta entrada es el aviso para revisarlo.

### RX-10 El texto de una tarea se dibuja como HTML

La descripción, las notas y el cuerpo de los comentarios se escriben libremente y la
interfaz los dibuja con `v-html`, así que un formato que la lista blanca deje pasar
llega al navegador. El texto entra por el CLI, por la API y por el pegado en la propia
pantalla: no hay una sola boca que lo valide.

**Cobertura:** `web/src/shared/tasks/rich-text.js` filtra el contenido con una lista
blanca de etiquetas y atributos antes de que Vue lo inserte, y una dirección sólo pasa
con un esquema que no ejecuta nada. Se comprobó con un texto que lleva `script`, `img`
con `onerror`, `iframe`, un manejador en un párrafo y un enlace `javascript:`, y ninguno
llegó a dibujarse ni ejecutó nada.

### RX-11 El validador del texto no alcanza todo lo que escribe el estado

Las listas del apartado 8 de [`contracts.md`](contracts.md) son léxicas: reconocen una
forma, no la intención de quien escribe. Se les escapa un hash escrito solo con dígitos, el
nombre de una rama que no lleva uno de los prefijos del flujo de Git —uno que también es
una carpeta del repositorio—, una declaración de estado en prosa sin marcador, y toda
escritura que no pase por el producto. Y la regla cuesta también en el otro sentido: lo
entrecomillado no exime, así que un texto que cita la frase rechazada se rechaza igual, y
el tablero no puede guardar el ejemplo de lo que prohíbe.

**Cobertura:** la revisión de quien escribe, que es la que había antes de la regla, y la
enmienda de las listas cuando una forma demuestra el mismo daño y no aparece en ningún
texto legítimo (`ADR-013`). Una escritura que no pasa por el producto solo la ve quien
lee la base.

### RX-12 La copia durable de los ADR es la exportación

La base del tablero no se versiona (`RX-9`), así que el registro de los ADR vive en la
máquina. Un ADR que no se exportó no está en Git: si la base se pierde, se pierde la
decisión, y el directorio versionado es lo único que queda.

**Cobertura:** `adr export --check` en las comprobaciones antes de cerrar un cambio que
toque una decisión, y la exportación en el mismo commit que la escribe. La base se puede
copiar junto con las tareas.

## Resueltos

- **RX-9 La base del tablero es un binario que no se fusiona.** La base viajaba en el
  repositorio y un binario en Git no se fusiona: dos máquinas que muevan el tablero
  producen un conflicto que no se resuelve sin perder las tareas de una de ellas. La base
  salió del índice y cada máquina tiene su tablero; llevar el tablero a otra máquina es un
  traslado manual que no tiene volcado. **Lección:** la cobertura prometía un volcado
  «cuando moleste» y, al molestar, se eligió la salida más barata. El volcado sigue
  faltando.

- **RX-5 El catálogo de etiquetas se reimplementaba sin pruebas heredadas.** El catálogo
  vivía en el servidor anterior y no tenía pruebas nuestras. Las escribió la fase que lo
  implementó: renombrar una etiqueta no toca ninguna tarea, borrar una avisa de las tareas
  que la usaban, y cambiar las etiquetas de una tarea no toca las de otra. **Lección:** una
  cobertura prometida para una fase futura hay que releerla cuando esa fase llega.

- **RX-6 El trabajo vivía sólo en local.** El registro aceptaba como deuda que el
  repositorio no estaba en ningún remoto. El proyecto pasó a tener remoto y el trabajo a
  vivir allí. **Lección:** lo que deja de ser cierto se corrige en el mismo cambio que lo
  dejó obsoleto, o se retira; el registro mintió hasta que se volvió a leer entero.

## Deuda aceptada

- La interfaz de partida trae una pieza sin uso: no consume `GET /api/status`. Sigue ahí, y
  todavía no se ha decidido si se retira o se usa.
