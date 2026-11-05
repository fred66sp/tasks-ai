# Registro de cambios

Cambios con valor para quien usa el proyecto, por versión. El formato sigue las
recomendaciones de *Keep a Changelog* y el versionado es semántico. Lo que se anota aquí
al publicar; la historia de cada cambio, commit a commit, vive en Git.

## [Sin publicar]

### Añadido

- El sistema de diseño incorpora la iconografía: los 72 iconos de Heroicons 24/outline viven
  en `web/public/icons/heroicons/` con su licencia, y la guía navegable los cataloga por
  grupos con su origen y sus reglas de uso.
- Los ADR del proyecto viven en el tablero, con las mismas superficies que las tareas: el
  CLI (`adr`), la API HTTP (`/api/adrs`) y la vista de Decisiones de la interfaz.
  `docs/decisions/` pasa a ser su exportación generada, y `adr export --check` detecta la
  deriva entre la base y el directorio versionado.

### Corregido

- La vuelta arriba de la documentación apunta a un ancla real al principio del documento y
  no a la cabecera sticky, que siempre estaba en pantalla y dejaba la página donde estaba.

### Cambiado

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
