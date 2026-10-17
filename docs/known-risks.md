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

- La interfaz de partida trae piezas sin uso: `web/src/task-manager/components/toast.js` no
  lo importa nadie, y la interfaz no consume `GET /api/status`. Siguen ahí, y todavía no se
  ha decidido si se retiran o se usan.
