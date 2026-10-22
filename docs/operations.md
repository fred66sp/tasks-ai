# Operación

Cómo se construye, se prueba y se ejecuta el producto, y qué se ha verificado. Las
formas de datos y de API están en `contracts.md`.

## Construir y ejecutar

| Comando | Qué hace |
|---|---|
| `go build ./cmd/...` | Compila el binario del producto |
| `go test ./...` | Ejecuta las pruebas |
| `go vet ./...` | Comprobaciones estáticas |
| `./tasks-ai init` | Instala el producto en el proyecto: crea `.task-ai/` con la base, el settings y las instrucciones, y devuelve el JSON del resultado. Con `--prefix` fija el prefijo de las referencias en la propia instalación, en vez de `tk`, y escribe las instrucciones con él. No es repetible: un proyecto ya instalado lo rechaza |
| `./tasks-ai serve` | Arranca la API y la interfaz web en la dirección y el puerto del settings: la interfaz responde en la raíz del servidor y la API en sus rutas `/api/...` |
| `./tasks-ai <comando>` | Ejecuta cualquier operación del producto con salida JSON |
| `./serve.sh` | Deja el servidor en segundo plano: compila el binario, lo arranca con `serve`, escribe su registro y su pid en `.run/` e informa de la dirección. `./serve.sh status` dice si está en marcha y `./serve.sh stop` lo para |

La interfaz viaja incrustada en el binario: si se cambia su código, hay que
construirla antes de compilar el binario. Los comandos de su construcción están en
`README.md`, y la decisión, en `ADR-010`.

## El tablero del trabajo

El trabajo del proyecto vive en `.task-ai/`, y se opera con el mismo binario:

```bash
./bin/tasks-ai list        # todas las tareas, con su estado y si están bloqueadas
./bin/tasks-ai show tk-1   # la tarea con sus hijas, sus bloqueos y sus comentarios
./bin/tasks-ai serve       # la interfaz web y su API
```

La herramienta de desarrollo anterior —un servidor en Node sobre `tk`— se retiró, y su
histórico se importó en este tablero (`ADR-005`).

## Comprobaciones antes de cerrar un cambio

- Las pruebas del proyecto (`go test ./...`).
- Las comprobaciones del contrato de documentación, que se ejecutan a mano: ninguna herramienta las automatiza.
- `make check-web`: que la interfaz construida coincida con su fuente.

## Registro de verificaciones

| Fecha | Qué se verificó | Resultado |
|---|---|---|
| 2026-10-16 | Interfaz servida por el binario: el ciclo de crear, ver, cerrar y reabrir una tarea desde el navegador contra el binario real, y las rutas de la API comparadas con las del código anterior | Comprobado: el ciclo completo por la interfaz, y sin diferencias en las rutas documentadas |
| 2026-10-16 | Comprobaciones del contrato de documentación sobre `PRD.md`, `architecture.md` y `contracts.md`: recuentos, medidas, fechas, tablas de progreso, afirmaciones de presente e identificadores ajenos | Sin hallazgos |
| 2026-10-16 | Mapa de identificadores: los requisitos citados desde los documentos de decisión existen en el PRD y ningún documento cita identificadores de otro registro | Sin hallazgos |
| 2026-10-16 | Comprobación de las reglas heredadas en la fuente de la referencia funcional: el cierre de una tarea sincroniza el estado del padre | Confirmado en el código, y descartado por decisión |
| 2026-10-17 | Ayuda y errores del binario real: la ayuda de todos los comandos y los casos de bandera desconocida, argumento ausente, estado inválido, etiqueta desconocida, transición no permitida, motivo no válido, tarea inexistente y etiqueta inexistente, más los cuerpos de error de la API en sus códigos de estado | Comprobado: en inglés, con los mismos códigos de salida y de estado; el mismo recorrido contra el binario anterior solo difiere en el texto de los mensajes |
| 2026-10-17 | La suite del producto, las comprobaciones estáticas, la suite con detector de carreras y orden aleatorio, y el linter del repositorio en la versión que fija la integración continua | Comprobado: pasan; el linter, sin hallazgos |
| 2026-10-17 | El recorrido de punta a punta de la extensión desde una sesión de Pi, sin leer su código ni escribir comandos: crear una épica y dos subtareas con sus etiquetas, poner una en curso, comentar una decisión, cerrar la épica con las subtareas abiertas y cerrarlas después | Comprobado: la épica se rechaza con el mensaje del producto y no escribe nada, las herramientas responden con el JSON del producto, y `serve` muestra las tres tareas cerradas con su jerarquía; las pruebas de la extensión pasan |
| 2026-10-17 | Las salidas del binario que reproduce `docs/web/index.html`: cada mensaje de error de la guía, el aviso de la etiqueta borrada y la línea de registro de `serve`, comparados con la ejecución real del mismo comando | Comprobado: coinciden; las únicas diferencias son la ruta del proyecto y la referencia que la guía elige como ejemplo |
| 2026-10-18 | El cambio de padre de una tarea de punta a punta contra el binario real: asignar, mover y quitar por CLI (salidas 1 y 2 en los rechazos del contrato), por la API HTTP (404 y 400), por la interfaz web en navegador real (el campo «Tarea padre» actualiza árbol y kicker) y por la extensión de Pi (suite con los casos de la unidad CLI, 50 pruebas) | Comprobado: los cuatro recorridos en verde, sin nada escrito en los rechazos, y la etiqueta `task` entrando y saliendo con el padre |
| 2026-10-18 | La búsqueda de la interfaz contra el binario reconstruido y el fichero servido: tres tareas cuyo texto buscado vive solo en la descripción, solo en las notas y solo en un comentario, más un texto que no aparece en ningún campo | Comprobado: cada texto lista su tarea y el que no aparece no lista nada; la guía de uso nombra los campos en sus dos idiomas y su bloque de traducciones es válido |
| 2026-10-18 | El agente de delegación de Pi: descubrimiento como agente de proyecto, lectura del criterio de una tarea del tablero en un encargo real, negativa ante una referencia inexistente y resolución del modelo por el routing del arnés | Comprobado: el criterio se leyó del tablero y no del encargo, la referencia inexistente devolvió la decisión pendiente sin tocar archivos, y el routing del proyecto resolvió el perfil; el modelo de los lanzamientos de la sesión en curso siguió siendo el del arnés, porque su configuración se congela al arrancar |
| 2026-10-20 | El formato del texto de una tarea contra el binario real, en un tablero de usar y tirar servido en otro puerto y abierto en un navegador de verdad: una tarea con énfasis, lista y enlace; otra con `script`, `img` con `onerror`, `iframe`, un manejador en un párrafo y un enlace `javascript:`; y otra con texto llano de dos líneas | Comprobado: la descripción, las notas y el comentario se dibujan con su formato y conservan sus saltos de línea en los tres casos; del texto peligroso no queda ninguna etiqueta que ejecute ni cargue, ni un atributo que empiece por `on`, y la variable que el guion escribía nunca se define |
| 2026-10-21 | El alfabeto del validador del texto de las tareas —el apartado 8 de `docs/contracts.md`— aplicado a los 579 textos guardados entonces en la base del tablero: títulos, descripciones, notas y comentarios, aplicando las cinco expresiones del apartado 8 a un volcado de `select id, title, description, notes from tasks` y `select task_id, body from comments`, y revisando cada resultado | Comprobado: 21 textos caen en alguna lista —3 hashes de commit, 1 referencia a un remoto, 16 declaraciones de estado con marcador y 1 declaración del estado de la entrega—, y los 21 se revisaron uno por uno sin encontrar ningún falso positivo; una fecha escrita sin guiones queda fuera de la lista de hashes por no llevar ninguna letra |
| 2026-10-22 | Que el producto no toque las etiquetas estructurales: el mismo recorrido de `init`, `create`, `create --parent`, `update --tags`, `update --parent` y `close` contra un tablero desechable, ejecutado con el binario anterior y con el nuevo y comparado salida, error y código de salida de cada comando; y el criterio de la tarea comprobado sobre el binario real | Comprobado: la única diferencia es la etiqueta `task` y su entrada en el catálogo, que desaparecen; los códigos de salida y el resto de campos coinciden. `create --parent` deja la tarea sin etiquetas, el cambio de padre no las toca, y `epic` se rechaza como nombre desconocido y se asigna igual que cualquier otra cuando existe |
