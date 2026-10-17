# ODD-89 — Resolver hallazgos de la auditoría ponytail
Rama: `fix/audit-findings`.
Épica: `tk-89`; cada tarea se implementa y verifica por separado.

## Specs

- S1: “pues adelante tienes autorización. Crea una rama para estas tareas, haz los commits necesarios con mi autorización. No puedes hacer push ni integración con main hasta que yo lo revise.”
- S2: “tk-89 pasa a subtarea épica, crea las subtareas hijas para solucionar todo esto.”
- S3: “Resultado: `init` informa cuando una base existente usa un prefijo distinto del settings, también al completar una instalación incompleta. Criterio: reproducir la recuperación con una base `px` sin settings ni instrucciones y comprobar por la salida de error el aviso exacto, además de conservar la operación y no escribir fuera de `.task-ai/`.”
- S4: “Resultado: cada cambio de tarea emite una descripción correcta, incluidos comentarios y varias escrituras dentro del mismo intervalo de sondeo. Criterio: comprobar por `GET /api/events` la creación, cierre, actualización, comentario y dos cambios consecutivos, verificando cada `id`, `action`, título y `changes` mediante la interfaz pública.”
- S5: “Resultado: la API rechaza cuerpos con JSON válido seguido de cualquier contenido adicional. Criterio: enviar una petición de escritura con `{"title":"válida"} basura`, comprobar HTTP 400 con `{"error":...}` y confirmar mediante la API que no se creó ninguna tarea.”
- S6: “Resultado: el listado HTTP entrega `parent_id` en la forma documentada —referencia visible del padre— o actualiza de forma explícita el contrato y todos sus consumidores. Criterio: crear padre e hija, consultar `GET /api/tasks` y comprobar la forma elegida en la respuesta y en la interfaz web, sin romper CLI ni extensión Pi.”
- S7: “Resultado: las consultas de detalle por `comments.task_id` y `dependencies.blocker_id` usan índices. Criterio: migrar una base existente sin perder datos, ejecutar `show` con comentarios y dependencias, y comprobar por la API/CLI que las respuestas son iguales antes y después de la migración.”
- S8: “Resultado: abrir una base cuyo esquema ya está al día no inicia una transacción por cada migración aplicada. Criterio: abrir una base nueva y otra ya migrada, ejecutar comandos de lectura y comprobar que el esquema y las respuestas permanecen iguales, conservando la seguridad cuando dos procesos inicializan a la vez.”
- S9: “Resultado: reconectar el canal SSE desde un ID antiguo no carga todo el historial en memoria ni ejecuta una consulta por etiqueta. Criterio: generar un historial largo, reconectar con `after`, comprobar que los eventos llegan completos y ordenados por lotes, y verificar que las etiquetas borradas no rompen el canal.”
- S10: “Resultado: guardar settings no puede dejar un JSON truncado si el proceso falla durante la escritura. Criterio: conservar el archivo anterior si la sustitución no se completa y comprobar que una instalación y los comandos posteriores siguen leyendo settings válidos.”
- S11: “Resultado: un settings con una clave mal escrita se rechaza con un error claro en lugar de aplicar silenciosamente el valor por omisión. Criterio: arrancar con una clave desconocida, comprobar el error de CLI y HTTP cuando corresponda, y verificar que un settings válido parcial conserva sus valores por omisión.”
- S12: “Resultado: retirar el método interno sin callers y conservar el comportamiento del catálogo de etiquetas. Criterio: compilar, pasar la suite Go y comprobar por CLI/API que crear, editar, listar y borrar etiquetas siguen funcionando.”
- S13: “Resultado: `install.Root` y `install.Project` comparten el recorrido ascendente sin cambiar dónde encuentra el producto. Criterio: comprobar desde raíz, subdirectorio, repositorio con `.git` como archivo y proyecto sin instalación que `init` y los comandos localizan exactamente las mismas carpetas que antes.”

## Tasks

- T1 — `tk-90`, S3, inline, commit `915dad5`: conservar el aviso de prefijo durante `init`.
- T2 — `tk-91`, S4, inline, commit pendiente: corregir descripciones de eventos SSE.
- T3 — `tk-92`, S5, inline, commit pendiente: rechazar JSON con contenido posterior.
- T4 — `tk-93`, S6, inline, commit pendiente: alinear `parent_id` de la API con el contrato.
- T5 — `tk-94`, S7, inline, commit pendiente: añadir índices a comentarios y dependencias.
- T6 — `tk-95`, S8, inline, commit pendiente: evitar transacciones de migración innecesarias.
- T7 — `tk-96`, S9, inline, commit pendiente: paginar y agrupar el historial de eventos.
- T8 — `tk-97`, S10, inline, commit pendiente: escribir settings de forma atómica.
- T9 — `tk-98`, S11, inline, commit pendiente: rechazar claves desconocidas en settings.
- T10 — `tk-99`, S12, inline, commit pendiente: eliminar `Store.Tag` sin uso.
- T11 — `tk-100`, S13, inline, commit pendiente: unificar el recorrido de raíces de instalación.

## Log

- L1: “pues adelante tienes autorización. Crea una rama para estas tareas, haz los commits necesarios con mi autorización. No puedes hacer push ni integración con main hasta que yo lo revise.”
- L2: “tk-89 pasa a subtarea épica, crea las subtareas hijas para solucionar todo esto.”
- L3: Se reabrió `tk-89` como épica en curso y se crearon `tk-90` a `tk-100` como subtareas abiertas.
