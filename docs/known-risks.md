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
nombre de una rama que no lleva uno de los prefijos del contrato de Git —uno que también es
una carpeta del repositorio—, una declaración de estado en prosa sin marcador, y toda
escritura que no pase por el producto. Y la regla cuesta también en el otro sentido: lo
entrecomillado no exime, así que un texto que cita la frase rechazada se rechaza igual, y
el tablero no puede guardar el ejemplo de lo que prohíbe.

**Cobertura:** la revisión de quien escribe, que es la que había antes de la regla, y la
enmienda de las listas cuando una forma demuestra el mismo daño y no aparece en ningún
texto legítimo (`ADR-013`). Una escritura que no pasa por el producto solo la ve quien
lee la base.

## Resueltos

### RX-9 La base del tablero es un binario que no se fusiona

La base viajaba en el repositorio para llevarla a otra máquina, y un binario en Git no se
fusiona: dos máquinas que muevan el tablero producen un conflicto que no se resuelve
eligiendo un lado sin perder las tareas del otro, y cada cambio de tarea engorda la
historia.

**Qué resultó:** la base dejó de viajar. `.task-ai/tasks.db` sale del índice y el
`.gitignore` la deja fuera, así que cada máquina tiene su tablero y el conflicto no
existe; lo que viaja con el trabajo nuevo es el código. Los commits anteriores a ese
cambio conservan la base, y llevar el tablero a otra máquina es ahora un traslado manual
que todavía no tiene volcado.

**Lección:** la cobertura de esta entrada prometía un volcado de texto «cuando moleste»,
y al molestar se eligió la salida más barata —retirar la base del repositorio— en vez de
escribirlo. El volcado sigue faltando.

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
