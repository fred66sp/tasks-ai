---
status: accepted
date: 2026-10-26
decision-makers: "Alfredo González Portero"
---

# ADR-016: Los ADR viven en la base y se exportan a texto

## Contexto

El producto guarda sus tareas en `.tasks-ai/tasks.db` y no conoce sus propias
decisiones: los quince ADR del proyecto viven como archivos markdown en
`docs/decisions/`. Esa forma conserva lo que el proyecto necesita —revisión en el
Pull Request, la prueba de enlaces y títulos de `go test ./...` y la puerta de
`adr-software`— y a la vez deja las decisiones fuera del producto: no se listan ni se
editan desde la interfaz, no participan del canal de cambios y el índice lo mantiene
una herramienta que el binario no ejecuta.

El tablero ya es la única fuente del trabajo del proyecto, y la persona ha decidido que
las decisiones durables sigan el mismo camino. La pregunta es dónde vive el registro,
qué se versiona y cómo se conserva lo que hoy funciona.

## Decisión

- **La base es la fuente de los ADR.** Una tabla `adrs` en `tasks.db`, gobernada por el
  núcleo: ninguna superficie toca SQLite, y las reglas de un ADR —número, estado,
  sustitución y congelación del cuerpo— se imponen donde se imponen las de una tarea.
- **`docs/decisions/` es una exportación generada y versionada.** Se escribe con
  `adr export --dir docs/decisions` y se comprueba con `adr export --check`. Cada archivo
  lleva una marca de que el producto lo generó; editarlo a mano no cambia el ADR, porque
  la siguiente exportación lo revierte.
- **La exportación existe para conservar tres cosas que el tablero no da**: el diff y la
  revisión en el Pull Request, la prueba de enlaces y títulos de `go test ./...`, y la
  puerta determinista de `adr-software`, que sigue siendo la revisora del texto.
- **El producto no impone la puerta editorial.** No exige un ADR completo para
  guardarlo: un `proposed` puede estar a medias. La puerta dice si el ADR permite
  implementar la decisión; el producto dice si la operación respeta sus reglas.
- **Las tres superficies del producto crean y editan ADR**: CLI, API HTTP e interfaz
  web. Las extensiones del agente no ganan herramientas de ADR.
- **Un ADR tiene número correlativo, `slug` inmutable y referencia `ADR-NNN`.** El
  número se asigna dentro de la transacción de escritura y no se reutiliza; el `slug`
  tampoco cambia, porque es el nombre del archivo exportado.
- **Los estados son `proposed`, `accepted`, `rejected`, `deprecated` y `superseded`.**
  Sustituir exige un sustituto existente y distinto y no admite ciclos.
- **El cuerpo se congela al salir de `proposed`.** Después solo se admiten añadidos
  fechados en «Más información»; un cambio de decisión se publica como ADR nuevo que
  sustituye al anterior. El cuerpo del anterior no se reescribe.
- **No hay borrado, ni versionado del cuerpo, ni tabla de relación con las tareas.** La
  referencia `tk-N` de un ADR vive en su texto, donde ya está, y Git guarda la historia
  del texto exportado.
- **La base no se versiona.** `tasks.db` y sus transitorios siguen fuera de Git: el
  binario no se fusiona y el WAL deja el archivo principal atrasado.

Alcance: el registro en la base, sus reglas, las tres superficies, la importación de los
ADR existentes y la exportación. No objetivos: migrar otros documentos al tablero;
enlazar ADR y tareas con una tabla; borrar ADR o versionar su cuerpo; router o URLs
profundas en la interfaz; herramientas de ADR en las extensiones.

La decisión se revisa si el proyecto deja de versionar `docs/decisions/` o si el
producto pasa a servir varios proyectos con directorios de ADR distintos: las dos cosas
rompen la exportación como espejo fiable.

## Consecuencias

- Cambiar un ADR deja dos escrituras que mantener coherentes: la base y su exportación.
  `adr export --check` es la comprobación que lo detecta, y entra en la lista de
  comprobaciones antes de cerrar un cambio.
- Quien edite un archivo de `docs/decisions/` a mano pierde el cambio en la siguiente
  exportación. La marca del principio del archivo lo dice.
- El número es del proyecto entero y no de una carpeta: el proyecto tiene un solo
  directorio de ADR. Un proyecto que quiera categorías tendrá que decidirlo aparte.
- Un cuerpo congelado desplaza las correcciones a «Más información» o a un ADR nuevo.
  Es más rígido que editar el markdown, y es lo que impide que un ADR aceptado cambie de
  significado sin que nadie lo revise.
- La base no versionada significa que la copia durable de los ADR es la exportación: un
  ADR que no se exportó no está en Git. El riesgo queda escrito en
  [`docs/known-risks.md`](../known-risks.md).
- La importación de los quince ADR existentes es una pasada única, idempotente por
  número: los archivos conservan su texto y ganan la marca de generado.
- Requisitos afectados: `RF-15` y `CA-19`. El contrato de formas y comandos está en
  [`docs/contracts.md`](../contracts.md).

## Alternativas consideradas

- **Base única, sin exportación.** Rechazada: el ADR desaparece del repositorio, y con él
  el diff en el Pull Request, la prueba de enlaces y la puerta de `adr-software`. El
  registro dejaría de revisarse al escribirse.
- **Mantener el markdown como fuente y que la interfaz edite los archivos.** Rechazada:
  el producto solo escribe dentro de `.tasks-ai/` (`RF-03`), y el servidor tendría que
  escribir en una ruta arbitraria del proyecto. Además, dos superficies escribiendo el
  mismo texto sin reglas compartidas acaban en formas distintas.
- **Versionar `tasks.db`.** Rechazada: es un binario, no se fusiona, ensucia el árbol y
  el WAL puede dejar el archivo principal atrasado respecto de lo que se creyó guardar.
- **Versionar el cuerpo de cada ADR en la base.** Rechazada: convierte el registro en un
  historial de ediciones que hay que consultar y podar. Git ya guarda la historia del
  texto exportado, y la congelación impide que un ADR aceptado cambie de significado.
- **Una tabla de relación entre ADR y tareas.** Rechazada: la referencia `tk-N` ya vive
  en el texto del ADR, y una tabla la duplicaría con dos sitios que mantener. El precio
  es que el producto no puede listar las tareas de un ADR; nadie lo ha pedido.
- **Un directorio de ADR por categoría, con numeración propia.** Rechazada: el proyecto
  tiene un solo directorio y la numeración repetida por categoría complica la referencia
  `ADR-NNN` sin resolver nada que hoy exista.
