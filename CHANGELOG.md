# Registro de cambios

Cambios con valor para quien usa el proyecto, por versión. El formato sigue las
recomendaciones de *Keep a Changelog* y el versionado es semántico. Lo que se anota aquí
al publicar; la historia de cada cambio, commit a commit, vive en Git.

## [Sin publicar]

## [0.2.0] - 2026-11-27

### Añadido

- `tasks-ai upgrade` actualiza la instalación desde el propio ejecutable en ejecución,
  sin descargas: reemplaza el binario y los recursos gestionados, conserva los datos y las
  personalizaciones, y retira las referencias exactas de host (`opencode.jsonc` y
  `.pi/settings.json`) solo cuando la porción no lleva comentarios y nunca una extensión
  referenciada. De las extensiones retiradas quita solo los ficheros que la v0.1.0
  distribuyó —lo demás del proyecto permanece, sin impedir que la actualización
  complete—, y deja el contrato de etiquetas a salvo si el proyecto lo editó. Escribe
  `.tasks-ai/VERSION` solo al completar; un upgrade incompleto sale con código distinto de
  cero y se completa reejecutando. Rechaza el downgrade —un ejecutable más viejo que la
  instalación—.
- `tasks-ai version` publica la versión del ejecutable en JSON.
- `.tasks-ai/VERSION` registra la versión de la instalación y es el marcador que lee
  `upgrade`.
- `tasks-ai aif` es la única superficie de agente (`ADR-021`): el perfil CLI de AIF 0.2 se
  sirve desde el propio binario con las once operaciones del tablero, los errores
  reparables por campo y la proyección recortada del tablero.

### Cambiado

- `init` ya no instala ni devuelve extensiones: su JSON pierde los campos `extension` y
  `opencode`, y deja de crear `.tasks-ai/extensions/`. **Breaking de la v0.2.0** para quien
  lea esos campos o las rutas (`docs/contracts.md` §2).
- `AGENTS.md` instalado es agéntico: instruye el descubrimiento de `aif`, el tablero y las
  operaciones reparables, sin el manual del CLI convencional, que sigue en `README.md` y en
  `docs/development.md`.

### Eliminado

- Las extensiones del agente de Pi y OpenCode, con su código TypeScript, sus pruebas, su
  integración continua, sus objetivos del `Makefile` y las configuraciones de host que las
  registraban (`ADR-021`; `ADR-012` queda sustituido, `ADR-014` y `ADR-015` deprecated).

## [0.1.0] - 2026-11-26

### Añadido

- La línea de comandos sirve el perfil CLI de AIF 0.2 con `tasks-ai aif`: una
  petición JSON por `stdin`, una respuesta y el código de salida que fija su
  resultado, sin estado entre invocaciones. Cubre las once herramientas del
  contrato §7 —incluidas las lecturas del tablero— con los errores clasificados
  por significado (referencia reparable, regla denegada y fallo real), y el
  tablero que recibe el agente es su proyección recortada.

- Las instrucciones que `init` distribuye incorporan la convención para encargos: un
  encargo sustancial nace con objetivo, alcance y criterios de aceptación en la
  descripción de su tarea; un cambio pequeño, con un título claro; al dejar el trabajo a
  medias queda un comentario que señala el siguiente paso.

- El responsable del proyecto se guarda en `person` de los ajustes: `init` lo sugiere una
  sola vez desde `git config user.name` y las tareas creadas sin `assignee` nacen con él.

- `docs/web/` se publica por sí sola (`ADR-018`): nace `docs/web/assets/` con el CSS común
  (`css/tokens.css` y `css/chrome.css`), el guion de mobilia (`js/docs.js`, islas Vue sin
  router ni construcción) y las copias locales de todo lo que las tres páginas consumían
  fuera (`vendor/`: hoja de iconos, CSS del tablero, Vue, toast-anchor y MicroLighter con
  sus licencias; `img/`: activos de marca). Copiada la carpeta a cualquier sitio, abre en
  `file://` o en un servidor estático sin red. La guarda `docs_web_test.go` corta la
  deriva con los dueños y prohíbe rutas relativas que salgan de la carpeta; `make
  docs-vendor` refresca las copias.
- El sistema de diseño incorpora la iconografía: el tablero y su documentación usan 23
  iconos de Heroicons 24/outline desde `web/public/icons/heroicons/`, con su licencia y una
  única hoja compartida, `web/public/icons/icons.css`. Los componentes y las tres páginas de
  `docs/web` ya no dibujan trazos propios: consumen el conjunto con una máscara que hereda el
  color del texto. La guía navegable documenta la selección, dónde se usa cada icono y sus
  reglas.
- Los ADR del proyecto viven en el tablero, con las mismas superficies que las tareas: el
  CLI (`adr`), la API HTTP (`/api/adrs`) y la vista de Decisiones de la interfaz.
  `docs/decisions/` pasa a ser su exportación generada, y `adr export --check` detecta la
  deriva entre la base y el directorio versionado.

### Cambiado

- El contrato de los textos del tablero cierra dos formas más de escribir el estado de
  Git: una referencia con nombre (`main`, `master` o `HEAD`) junto a un verbo de Git, y
  un sustantivo de Git (`rama`, `commit`, `merge`, `push`, `hash`…) con su identificador
  designado a continuación. `docs/<slug>` cuenta como rama, sin confundir las rutas de
  la documentación. Cuando un texto cae en varias listas, el rechazo nombra todos los
  fragmentos y no solo el primero.
- Las herramientas del tablero dicen cuándo se usan, no solo qué hacen: poner la tarea en
  curso al empezar el trabajo, comentar el avance cuando surge y cerrar con `done` solo
  tras la revisión o el pedido expreso de la persona. El ciclo viaja además en el resumen
  que se inyecta al agente en cada turno, y las instrucciones que copia `init` explican
  qué operación corresponde a cada momento del trabajo.
- Cerrar como hecha exige que la tarea esté en curso: una tarea pendiente se rechaza con
  un error que dice el atajo, y los demás motivos de cierre siguen valiendo desde
  pendiente (`ADR-019`).
- La portada presenta tasks-ai como producto para desarrollo agéntico: el agente gestiona
  tareas y procesos por la CLI y la persona revisa, hace seguimiento y conserva el control
  desde la interfaz web. Tres paneles ilustrados en SVG explican el reparto —el agente
  prepara, el proyecto responde, la persona decide— y mencionan los sistemas de
  documentación con los que convive el tablero, como SDD y ODD.
- La vista de Decisiones es un acordeón: cada ADR se pliega a tres líneas de su contexto y
  se lee y se edita entero en su propia tarjeta, sin panel lateral ni formulario aparte.
  El buscador pregunta al servidor y encuentra el texto en todos los campos del ADR, sin
  distinguir mayúsculas ni acentos. El cuerpo se dibuja como markdown, el registro pagina
  a 10, 20, 40 o 60 por página —10 por omisión—, vuelve arriba con un botón flotante y
  alterna el tono de las tarjetas. `adr list` y `GET /api/adrs` incluyen el contexto de
  cada decisión.
- La cabecera del gestor y las entradas de etiquetas y decisiones usan botones compactos
  con icono; Cerrar tarea pasa al detalle de la tarea seleccionada, junto a Editar y su
  estado, y en pantalla estrecha la cabecera deja las acciones en icono para no perder la
  entrada a Decisiones.

### Corregido

- El `pointer` que devuelve `init` pide también enlazar `CONTRATO_ETIQUETAS.md`:
  el archivo de instrucciones del proyecto enlaza así el contrato que dice qué tipo y
  qué etiqueta estructural lleva toda tarea, y no solo `AGENTS.md`.
- Los heroes de la portada, la guía y el sistema de diseño comparten los colores de los
  halos y la cuadrícula, con una única definición de cada fondo en el CSS común.
- La cabecera de la documentación conserva sus iconos en pantallas estrechas y la insignia
  de estado del tablero muestra solo su palabra: se retiró el trazo propio que quedaba
  negro en los dos temas.
- El desplegable de una tarea gira solo su chevron; antes giraban el botón y el icono, y la
  flecha quedaba al revés al abrir la tarea.
- La vuelta arriba de la documentación apunta a un ancla real al principio del documento y
  no a la cabecera sticky, que siempre estaba en pantalla y dejaba la página donde estaba.
