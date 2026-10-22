# Gestor de tareas del proyecto

Este proyecto guarda sus tareas con el gestor local que vive en esta carpeta.

## Qué hay aquí

| Elemento | Qué es |
|---|---|
| `tasks.db` | Las tareas del proyecto |
| `settings.json` | El prefijo de las referencias, la base y los ajustes de la interfaz |
| `AGENTS.md` | Estas instrucciones |
| `CONTRATO_ETIQUETAS.md` | El contrato de las etiquetas: clases, asignación y gestión del catálogo |

## Cómo se usa

Con la extensión de Pi cargada, el agente no escribe estos comandos: opera el tablero
con sus herramientas (`tasks_board`, `tasks_show`, `tasks_create`, `tasks_update`,
`tasks_close`, `tasks_reopen`, `tasks_block`, `tasks_unblock`, `tasks_comment`,
`tasks_comment_list` y `tasks_comment_edit`). Los comandos quedan para la persona. Desde
cualquier punto del proyecto:

```bash
tasks-ai create --title "Preparar el análisis"   # crea una tarea y devuelve su JSON
tasks-ai create --title "Revisar el análisis" --parent tk-1  # crea una subtarea de tk-1
tasks-ai show tk-1                               # devuelve la tarea en JSON, con sus hijas, sus bloqueantes, sus dependientes y sus comentarios
tasks-ai update tk-1 --priority 1                # cambia los campos que se piden y devuelve la tarea
tasks-ai update tk-1 --tags "docs,urgente"       # pone la lista entera de etiquetas de tk-1: una lista vacía la deja sin ninguna
tasks-ai tags                                    # devuelve el catálogo de etiquetas con el uso de cada una
tasks-ai tag create --name "docs" --color blue   # crea una etiqueta; el color que falta es blue
tasks-ai tag edit docs --name "documentacion"    # cambia de la etiqueta lo que se pide y nada más; renombrar no toca ninguna tarea
tasks-ai tag delete docs                         # borra la etiqueta con sus asignaciones y avisa de las tareas que la usaban
tasks-ai close tk-1 --reason done                # cierra la tarea con su motivo y devuelve la tarea
tasks-ai reopen tk-1                             # reabre la tarea cerrada y devuelve la tarea
tasks-ai block tk-2 tk-1                         # deja tk-2 bloqueada por tk-1 y devuelve tk-2
tasks-ai unblock tk-2 tk-1                       # retira el bloqueo entre tk-2 y tk-1 y devuelve tk-2
tasks-ai comment tk-1 --body "..."               # añade un comentario a tk-1 y devuelve la tarea
tasks-ai serve                                   # arranca la API HTTP en la dirección y el puerto del settings y anuncia los cambios por un canal abierto
```

El motivo de cierre es obligatorio y debe ser uno de: `done`, `duplicate`, `absorbed`,
`stale` o `superseded`. Cerrar una tarea con subtareas abiertas se rechaza sin escribir
nada, cerrar una subtarea no cierra su padre, y una tarea ya cerrada no se cierra otra
vez ni una abierta se reabre.

El estado bloqueado se calcula y no se guarda: una tarea aparece bloqueada mientras
tenga algún bloqueante que no esté `done`, y deja de estarlo al cerrar al bloqueante o
al retirar el bloqueo. Una referencia que no resuelve se rechaza y una tarea no puede
bloquearse por sí misma. Los comentarios se pueden añadir y editar por su identidad
dentro de la tarea; su cuerpo es obligatorio y no hay operación de borrado.

Las etiquetas se gestionan por su nombre en el catálogo de `tags`; el cómo y el cuándo
los gobierna [`CONTRATO_ETIQUETAS.md`](CONTRATO_ETIQUETAS.md). El color, si se da,
debe ser uno de la paleta que la propia ayuda del comando enumera (`tasks-ai tag create
--help`); un color que falte es `blue` y un color desconocido se rechaza. Renombrar una
etiqueta cambia el nombre que muestran las tareas sin tocar su fila guardada, y borrar
una retira sus asignaciones avisando de las tareas que la usaban.

Toda operación responde en JSON por la salida estándar, y los avisos y errores van por
la salida de error con la forma `{"error": "..."}`. `serve` no es una operación: no
devuelve JSON, habla por la salida de error y se detiene con Ctrl+C o con SIGTERM.

## Las referencias

Cada tarea tiene un número interno, que es su identidad, que es único en el proyecto y
que no se reutiliza. La referencia visible se compone con el prefijo del proyecto y ese
número: `tk-1`. Se escribe sin distinguir mayúsculas y admite espacios alrededor.

## El trabajo del proyecto

Lo que sigue lo escribe el proyecto, no la instalación: `init` no vuelve a escribir
este archivo, así que aquí se corrige lo que haga falta.

### Qué es una tarea

- Una **épica** es un frente de trabajo y lleva `epic`. Se reconoce por sus subtareas, no
  por su nivel: una épica puede colgar de otra tarea.
- Una **tarea** es una unidad de trabajo que puede atravesar sesiones y lleva `task`. No
  tiene subtareas; tener padre o no es otra cosa y no cambia lo que es.
- La etiqueta la pone quien crea la tarea: el producto no la añade ni la retira sola.
- Cada tarea lleva además un único tipo principal. Las clases de etiquetas, su asignación
  y la gestión del catálogo las gobierna [`CONTRATO_ETIQUETAS.md`](CONTRATO_ETIQUETAS.md).
  El trabajo del registro ODD lleva `ODD`.
- Toda tarea declara en su descripción **el resultado que la cierra** y **el criterio con
  el que se comprueba**, observable por la interfaz pública. Una tarea sin criterio no se
  empieza: se escribe primero.
- Además del resultado y el criterio, la tarea lleva los pasos con los que se aborda y,
  como comentarios, los hallazgos y las decisiones con su razón. Eso es lo que no caduca;
  el estado caduca siempre.

### El ciclo

1. Antes de crear, se lee el tablero: crear sin mirar deja duplicados que no se pueden
   borrar, solo cerrar.
2. Ninguna tarea la crea el agente por su cuenta: la propone con el resultado que cierra
   y el criterio con el que se comprueba, y espera la confirmación de la persona. Vale
   igual para lo que aparezca por el camino —un arreglo, una decisión, una duda—: se
   propone y se espera; no se anota por cuenta propia.
3. La tarea existe y está en curso antes de tocar archivos.
4. Las **notas** son el plan de la tarea: los pasos con los que se aborda y lo que hay que
   tener presente al retomarla. No guardan el estado —ni lo hecho, ni lo que queda, ni lo
   que sigue—: eso lo dicen la propia tarea, Git y la sesión del agente. Se sobrescriben.
5. Los **comentarios** son decisiones con su razón, hallazgos y evidencias. Se corrigen
   editando su cuerpo (`tasks_comment_edit`) y no se borran, así que se escribe lo que se
   querrá leer después.
6. Antes del commit se comprueba que la tarea está terminada, revisada y correcta: se lee
   la tarea entera —descripción, notas y comentarios—, se repasa el criterio punto por
   punto sobre lo entregado y no sobre lo que se recuerda, y se repasa el contrato de Git
   (`docs/CONTRATO_FLUJO_GIT_VERSIONADO.md`): la rama, la unidad de commit, la
   confirmación, la verificación y el CHANGELOG. El criterio es el mínimo: lo que se
   prometió en una nota o en un comentario y no se hizo, se hace. Nada se declara fuera de
   alcance, ni se abre una tarea nueva, sin la conformidad y la autorización de la persona.
   Solo entonces se commitea; después se comprueba que el commit fue bien y se cierra con
   su motivo. Commitear algo que no está terminado obliga a enmendarlo con otro commit y a
   abrir tareas para lo que debió estar dentro, y eso no se hace. El cierre escribe además
   en la base del tablero, así que cerrar antes deja ese cambio suelto o mezclado con el
   trabajo que venga. Cerrar una tarea con subtareas abiertas lo rechaza el producto.
7. Quien delega es quien cierra: un subagente trabaja la tarea y devuelve evidencia, pero
   no la cierra ni escribe el cierre. Lo cierra quien delegó, después de comprobarlo.

### Lo que no se escribe en el tablero

- El estado de la tarea: lo marca la herramienta y no se escribe. Se cambia con
  `tasks_update`, `tasks_close` y `tasks_block`, y su estado, su motivo de cierre y sus
  bloqueos ya lo dicen. Ni «pendiente», ni «a la espera», ni «abierta» o «cerrada», ni
  «reabierta». Una copia en prosa envejece sin que nadie la corrija y hace creer que el
  trabajo está donde no está.
- El estado de Git: ramas, hashes, commits pendientes. Se consulta con Git. La atadura
  entre una tarea y sus commits va en el mensaje del commit, con `Refs: tk-<n>`, y se
  resuelve con `git log --grep='Refs: tk-<n>'`: el tablero no hospeda hashes porque para
  decir qué se entregó le basta ese remitente.
- Las referencias que envejecen: un número de línea, el número de un comentario, un
  recuento o un «hoy» dejan de ser ciertos sin que nadie los corrija y hacen creer que el
  trabajo está donde no está. Se enuncia el hecho: qué fichero, qué función, qué se
  comprobó.
- El proceso de revisión. Cada cosa vive donde vive.
- Métricas, listas de archivos ni resúmenes de la documentación: la documentación tiene
  su sitio, y el tablero solo dice qué se está haciendo.
