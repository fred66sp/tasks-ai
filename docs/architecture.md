# Arquitectura del producto

Cómo está construido el producto y por qué. El porqué de cada elección vive en
`decisions/`; aquí se describe el diseño que resulta de ellas y se enlaza lo que ya
está decidido en otro sitio.

## 1. Vista general

```text
 persona ──> interfaz web ──HTTP──┐
                                  ├──> núcleo de negocio ──> SQLite
 agentes ──> CLI (JSON) ──────────┘                          .tasks-ai/
```

Un solo binario ofrece las dos superficies: los comandos del CLI y, cuando se le pide,
el servidor que sirve la interfaz. Ambas hablan con el mismo núcleo, y el núcleo es el
único que toca la base.

## 2. Capas y fronteras

| Capa | Responsabilidad | Lo que no hace |
|---|---|---|
| Núcleo | Reglas de negocio, identidad, consultas y escrituras | No conoce comandos, rutas ni formatos de presentación |
| Superficie CLI | Traduce comandos y banderas a llamadas del núcleo; serializa a JSON | No decide reglas ni toca la base |
| Superficie HTTP | Rutas, validación de entrada, códigos de estado y canal de cambios | No decide reglas ni toca la base |
| Interfaz web | Muestra y opera lo que le da la superficie HTTP | No conoce la base ni el núcleo |
| Almacén | Guarda el estado en un archivo SQLite | No contiene reglas: las impone el núcleo |
| Instalación | Reúne base, settings e instrucciones en una carpeta del proyecto | No escribe fuera de esa carpeta |

La frontera que ordena todo lo demás: **la base solo la toca el núcleo**. Las
superficies traducen y presentan; el núcleo decide.

## 3. Flujo de una operación

Cerrar una tarea, pedida desde el CLI o desde la interfaz:

1. La superficie valida la forma de la petición y resuelve la referencia escrita a la
   identidad interna de la tarea.
2. El núcleo comprueba las reglas: que la tarea exista, que no tenga subtareas
   abiertas y que el cierre no alcance a su padre.
3. El núcleo escribe el cambio en una sola transacción.
4. La superficie responde: JSON por el CLI, código de estado y cuerpo por HTTP.
5. Si la operación cambió datos, la superficie lo anuncia; las demás sesiones
   recargan lo que no han cambiado ellas.

Las dos superficies recorren los mismos pasos intermedios porque comparten el núcleo;
lo único que cambia es el idioma de entrada y de salida.

## 4. Invariantes

- Ninguna regla de negocio vive fuera del núcleo (`RF-02`).
- El estado bloqueado se calcula desde las dependencias y no ocupa lugar en la base
  (`RN-04`).
- La identidad de una tarea es su número interno; la referencia visible se compone al
  mostrarla (`RN-03`).
- Un cierre con subtareas abiertas se rechaza, y cerrar una subtarea no cierra su
  padre (`RN-05`, `RN-06`).
- Los errores se responden con el código de estado que corresponde (`RF-09`).
- Nada del producto vive fuera de su carpeta dentro del proyecto (`RF-03`).
- El número interno aparece siempre que un dato cruza la frontera del núcleo
  (`RN-03`).
- La ayuda, los errores y los avisos del producto se escriben en inglés; la
  documentación, los comentarios y el texto de la interfaz web, en español (`ADR-011`).

## 5. Dónde vive el estado

Dentro de `.tasks-ai/`, en la raíz del proyecto:

| Elemento | Qué guarda |
|---|---|
| `tasks.db` | Tareas, dependencias, comentarios y el catálogo de etiquetas con color y descripción |
| `settings.json` | Prefijo, ruta de la base, dirección y puerto de la interfaz, y registro |
| `AGENTS.md` | Instrucciones para las personas y los agentes que trabajan en el proyecto |

Fuera de esa carpeta el producto no escribe: solo ofrece el texto que el usuario pueda
querer añadir en sus propios archivos.

## 6. Almacén

- Acceso directo a SQLite, sin intermediarios (`ADR-007`).
- La base ordena las escrituras y la interfaz puede leer mientras se escribe
  (`ADR-008`).
- El esquema se versiona dentro de la propia base y las migraciones son aditivas:
  añadir columnas o tablas nuevas con valor por omisión, sin reescribir lo existente.
  Un producto local no puede permitirse perder datos al actualizarse.
- Cada operación del núcleo escribe en una sola transacción: lo que se ve, se ve
  entero.

## 7. Identidad y referencias

- La clave de una tarea es su número interno, entero, único dentro del proyecto y no
  reutilizado.
- Las relaciones —padre, dependencias, comentarios— usan ese número.
- La referencia visible se compone al vuelo con el prefijo del proyecto y el número.
- Cuando un dato cruza la frontera del núcleo viaja el número; cuando se presenta a
  una persona se compone la referencia.
- Cambiar el prefijo no toca ninguna fila: cambia lo que se escribe, no lo que se
  guarda (`ADR-004`).

## 8. Superficie HTTP

- Rutas por recurso, con la referencia visible en la petición y el número interno en
  la respuesta.
- Los cambios se anuncian por un canal de eventos, para que las demás sesiones
  recarguen.
- La validación de la entrada vive en esta frontera; las reglas, no.

## 9. Interfaz web

La interfaz existente se reutiliza (`ADR-005`). Su capa de peticiones apunta a la
superficie HTTP del núcleo; el árbol, los filtros, la búsqueda, las preferencias y las
hojas de estilo no se reescriben. Lo que el servidor anterior calculaba —el estado
bloqueado, las reglas de cierre y la cascada del catálogo de etiquetas— pasa al
núcleo. Las capacidades nuevas, el progreso de épicas y el recuento, se añaden sobre
la misma base.

Su código vive en `web/` y su construcción es suya, no del compilador de Go: el
artefacto construido se versiona y viaja incrustado en el binario, de modo que `serve`
reparte las dos superficies sin depender de nada más (`ADR-010`).

## 10. Integración con el agente

El agente que trabaja en el proyecto llega al tablero por una extensión del agente Pi,
que vive en `extensions/pi/` y se incrusta en el binario para que el install la copie
(`ADR-012`, `ADR-014`). No es una tercera
superficie del producto: envuelve el CLI, ejecuta sus comandos y devuelve su JSON.

- No toca la base ni reimplementa reglas: lo que el núcleo rechaza llega rechazado.
- El proyecto lo encuentra el CLI subiendo hasta `.tasks-ai/`, no la extensión.
- Una herramienta por operación que el CLI ya tenga; ninguna que no exista.
- Las etiquetas se asignan después de crear, porque `create` no las acepta.

```text
 agente ──> extensión (extensions/pi/) ──CLI──> núcleo ──> SQLite
```

## 11. De qué decisión sale cada pieza

| Pieza del diseño | Decisión |
|---|---|
| Producto en Go, local y de un solo proyecto | `ADR-001` |
| Un binario con las dos superficies y un núcleo | `ADR-002` |
| Instalación en una sola carpeta del proyecto | `ADR-003` |
| Identidad interna y referencia visible | `ADR-004` |
| Interfaz reutilizada con servidor nuevo | `ADR-005` |
| Reglas heredadas conservadas y descartadas | `ADR-006` |
| SQLite directo | `ADR-007` |
| Escrituras ordenadas por la base | `ADR-008` |
| Forma del código y dependencias | `ADR-009` |
| Interfaz en `web/`, incrustada en el binario | `ADR-010` |
| Integración con el agente: incrustada para distribuirla | `ADR-012`, `ADR-014` |
