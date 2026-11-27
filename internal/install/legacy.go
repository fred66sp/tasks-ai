package install

// legacyAgentTemplate is the exact instruction template v0.1.0 shipped: the
// content of agent.md at 5ef79fd, kept to recognise a project that still carries
// it and replace it with the current one. It is history, not a template that is
// written again.
const legacyAgentTemplate = "# Gestor de tareas del proyecto\n" +
	"\n" +
	"Este proyecto guarda sus tareas con el gestor local que vive en esta carpeta.\n" +
	"\n" +
	"## Qué hay aquí\n" +
	"\n" +
	"| Elemento | Qué es |\n" +
	"|---|---|\n" +
	"| `tasks.db` | Las tareas y las decisiones del proyecto |\n" +
	"| `settings.json` | El prefijo de las referencias, la base y los ajustes de la interfaz |\n" +
	"| `AGENTS.md` | Estas instrucciones |\n" +
	"| `CONTRATO_ETIQUETAS.md` | El contrato de las etiquetas: clases, asignación y gestión del catálogo |\n" +
	"\n" +
	"## Cómo se usa\n" +
	"\n" +
	"Desde cualquier punto del proyecto:\n" +
	"\n" +
	"```bash\n" +
	"tasks-ai create --title \"Preparar el análisis\"   # crea una tarea y devuelve su JSON\n" +
	"tasks-ai create --title \"Revisar el análisis\" --parent tk-1  # crea una subtarea de tk-1\n" +
	"tasks-ai show tk-1                               # devuelve la tarea en JSON, con sus hijas, sus bloqueantes, sus dependientes y sus comentarios\n" +
	"tasks-ai update tk-1 --priority 1                # cambia los campos que se piden y devuelve la tarea\n" +
	"tasks-ai update tk-1 --tags \"docs,urgente\"       # pone la lista entera de etiquetas de tk-1: una lista vacía la deja sin ninguna\n" +
	"tasks-ai tags                                    # devuelve el catálogo de etiquetas con el uso de cada una\n" +
	"tasks-ai tag create --name \"docs\" --color blue   # crea una etiqueta; el color que falta es blue\n" +
	"tasks-ai tag edit docs --name \"documentacion\"    # cambia de la etiqueta lo que se pide y nada más; renombrar no toca ninguna tarea\n" +
	"tasks-ai tag delete docs                         # borra la etiqueta con sus asignaciones y avisa de las tareas que la usaban\n" +
	"tasks-ai adr create --title \"Adoptar SQLite\" --deciders \"Ana\"  # crea una decisión durable y devuelve su número\n" +
	"tasks-ai adr edit 1 --status accepted            # cambia lo que se pide y devuelve el ADR; el cuerpo se congela al salir de proposed\n" +
	"tasks-ai adr list                                # devuelve los ADR del proyecto, sin sus cuerpos\n" +
	"tasks-ai adr show ADR-001                        # devuelve el ADR entero, con su cuerpo por secciones\n" +
	"tasks-ai adr import --dir docs/decisions         # guarda los ADR de un directorio que aún no estén, sin duplicar\n" +
	"tasks-ai adr export --dir docs/decisions         # escribe la exportación generada de los ADR y su índice\n" +
	"tasks-ai close tk-1 --reason done                # cierra la tarea con su motivo y devuelve la tarea\n" +
	"tasks-ai reopen tk-1                             # reabre la tarea cerrada y devuelve la tarea\n" +
	"tasks-ai block tk-2 tk-1                         # deja tk-2 bloqueada por tk-1 y devuelve tk-2\n" +
	"tasks-ai unblock tk-2 tk-1                       # retira el bloqueo entre tk-2 y tk-1 y devuelve tk-2\n" +
	"tasks-ai comment tk-1 --body \"...\"               # añade un comentario a tk-1 y devuelve la tarea\n" +
	"tasks-ai comment list tk-1                       # devuelve los comentarios de tk-1, del primero al último, cada uno con su identidad\n" +
	"tasks-ai comment edit tk-1 1 --body \"...\"        # cambia el cuerpo del comentario 1 de tk-1: conserva su identidad y su fecha\n" +
	"tasks-ai serve                                   # arranca la API HTTP en la dirección y el puerto del settings y anuncia los cambios por un canal abierto\n" +
	"```\n" +
	"\n" +
	"El motivo de cierre es obligatorio y debe ser uno de: `done`, `duplicate`, `absorbed`,\n" +
	"`stale` o `superseded`. Cerrar una tarea con subtareas abiertas se rechaza sin escribir\n" +
	"nada, cerrar una subtarea no cierra su padre, y una tarea ya cerrada no se cierra otra\n" +
	"vez ni una abierta se reabre.\n" +
	"\n" +
	"## Cuándo se usa qué\n" +
	"\n" +
	"El tablero es el único registro del trabajo. Cada momento del ciclo tiene su operación, y\n" +
	"se hace cuando toca, no al final:\n" +
	"\n" +
	"| Momento | Operación |\n" +
	"|---|---|\n" +
	"| Abrir el trabajo | `tasks-ai create --title \"...\"`; la tarea nace `open`. Un encargo sustancial nace con objetivo, alcance y criterios de aceptación en su descripción; un cambio pequeño, con un título claro |\n" +
	"| Abrir el trabajo de una épica | `tasks-ai update <ref> --status in_progress` en la épica cuando empieza su primera hija; su cierre espera a que cierren todas |\n" +
	"| Empezar el trabajo | `tasks-ai update <ref> --status in_progress` |\n" +
	"| Surgir un avance, una decisión o un bloqueo | `tasks-ai comment <ref> --body \"...\"` en ese momento |\n" +
	"| Dejar el trabajo a medias | `tasks-ai update <ref> --status open`, y un comentario que señale el siguiente paso |\n" +
	"| Terminar | informe a la persona y `tasks-ai close <ref> --reason done`, con su revisión o su pedido expreso |\n" +
	"| Dejar de necesitarla sin hacerla | `tasks-ai close <ref> --reason duplicate`, `absorbed`, `stale` o `superseded` |\n" +
	"\n" +
	"El tablero no habla de Git: el estado de Git solo vive en Git.\n" +
	"\n" +
	"El estado bloqueado se calcula y no se guarda: una tarea aparece bloqueada mientras\n" +
	"tenga algún bloqueante que no esté `done`, y deja de estarlo al cerrar al bloqueante o\n" +
	"al retirar el bloqueo. Una referencia que no resuelve se rechaza y una tarea no puede\n" +
	"bloquearse por sí misma. Los comentarios se pueden añadir y editar por su identidad\n" +
	"dentro de la tarea; su cuerpo es obligatorio y no hay operación de borrado.\n" +
	"\n" +
	"Las etiquetas se gestionan por su nombre en el catálogo de `tags`; toda tarea lleva desde\n" +
	"que nace un tipo y un estructural (`task` o `epic`), y el vocabulario con el que nace el\n" +
	"proyecto y las reglas para asignarlo están en\n" +
	"[`CONTRATO_ETIQUETAS.md`](CONTRATO_ETIQUETAS.md). El color, si se da,\n" +
	"debe ser uno de la paleta que la propia ayuda del comando enumera (`tasks-ai tag create\n" +
	"--help`); un color que falte es `blue` y un color desconocido se rechaza. Renombrar una\n" +
	"etiqueta cambia el nombre que muestran las tareas sin tocar su fila guardada, y borrar\n" +
	"una retira sus asignaciones avisando de las tareas que la usaban.\n" +
	"\n" +
	"Las decisiones durables del proyecto viven en la misma base, cada una con su número y su\n" +
	"referencia `ADR-NNN`. Se escriben con `adr`, no editando el markdown: `docs/decisions/`\n" +
	"es una exportación generada desde la base y `adr export --check` dice si hay deriva. El\n" +
	"cuerpo se congela al salir de `proposed`; después solo se admiten añadidos fechados en\n" +
	"«Más información», y un cambio de decisión se publica como un ADR nuevo que sustituye al\n" +
	"anterior.\n" +
	"\n" +
	"Toda operación responde en JSON por la salida estándar, y los avisos y errores van por\n" +
	"la salida de error con la forma `{\"error\": \"...\"}`. `serve` no es una operación: no\n" +
	"devuelve JSON, habla por la salida de error y se detiene con Ctrl+C o con SIGTERM.\n" +
	"\n" +
	"## Las referencias\n" +
	"\n" +
	"Cada tarea tiene un número interno, que es su identidad, que es único en el proyecto y\n" +
	"que no se reutiliza. La referencia visible se compone con el prefijo del proyecto y ese\n" +
	"número: `tk-1`. Se escribe sin distinguir mayúsculas y admite espacios alrededor.\n"
