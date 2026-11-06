# Desarrollo

Cómo está organizado el repositorio, cómo se trabaja en él y cómo se ejecuta el producto.
Las formas de datos y de API están en `contracts.md`; el diseño, en `architecture.md`.

## Estructura

| Ruta | Qué es |
|---|---|
| `docs/` | La documentación del producto: requisitos, decisiones, diseño, contratos y riesgos |
| `docs/decisions/` | La exportación generada de los ADR: un archivo por decisión durable |
| `docs/web/` | Las páginas de documentación —portada, guía y sistema de diseño—: se publican por sí solas, copiadas a cualquier sitio funcionan en `file://` o en un servidor estático sin red (`ADR-018`) |
| `docs/web/assets/` | Lo común de esas páginas: `css/` (tokens y mobilia), `js/` (el guion clásico `docs.js` con las islas Vue), `vendor/` (copias locales de `icons.css`, `app-vue.css`, Vue, toast-anchor y MicroLighter, cada una con su licencia) e `img/` (activos de marca); la guarda del paquete raíz corta la deriva con sus dueños |
| `cmd/tasks-ai/` | El punto de entrada del binario |
| `internal/settings/` | Los ajustes del proyecto: su archivo en la carpeta del producto y sus valores por omisión |
| `internal/store/` | La base SQLite: su esquema versionado y lo que guarda de las tareas |
| `internal/core/` | Las reglas del producto: crear tareas, darles identidad y referencia, y leerlas |
| `internal/install/` | La instalación en un proyecto: su carpeta con la base, los ajustes y las instrucciones |
| `internal/cli/` | La línea de comandos: comandos, banderas y respuesta en JSON |
| `internal/` | El núcleo del producto y sus superficies |
| `web/` | La interfaz web: el código de la de partida en `web/src/`, lo que el navegador descarga en `web/public/` —con el artefacto construido versionado, la selección de iconos de `icons/heroicons/` y su hoja compartida `icons/icons.css`— y `web/embed.go`, que la incrusta en el binario |
| `.impeccable/` | El sidecar del sistema visual: rampas, sombras, movimiento y los componentes que un panel de diseño dibuja; acompaña a [`DESIGN.md`](../DESIGN.md) |
| `design-system.json` | El contrato máquina-legible del sistema visual —identidad, tokens del tablero, componentes y gobernanza—; su prosa es [`DESIGN.md`](../DESIGN.md), sus muestras la guía de diseño y lo contrasta `web/src/task-manager/design-tokens.test.js` |
| `.tasks-ai/` | El tablero del proyecto: la base de tareas, sus ajustes y sus instrucciones |
| `.pi/` | El estado de Pi del proyecto, versionado: el paquete instalado en `.pi/settings.json` y el agente de delegación en `.pi/agents/`, cuyo modelo se elige con `/gentle:models` ([`README.md`](../README.md)) |
| `extensions/pi/` | La integración con el agente de Pi: herramientas en TypeScript sobre el CLI del producto, con sus pruebas propias |
| `extensions/opencode/` | La integración con el agente OpenCode: las mismas herramientas sobre el CLI, con sus pruebas propias |
| `skills/` | Las skills del proyecto, copiadas de su origen: el taller y `SKILLS-AGP` |

Las dos extensiones se comprueban con `npm run typecheck` y `npm test`. Sus dependencias se
instalan una vez con `npm install` y quedan en `extensions/*/node_modules/`, que no se
versiona. Hacen falta porque el typecheck necesita TypeScript y los tipos de Node, y porque
`index.ts` de la extensión de Pi importa `typebox`, que en el paquete es `peerDependency` —lo
inyecta Pi al cargar la extensión, no Node al probarla— y sin él la suite no arranca.

```bash
cd extensions/pi
npm install
npm run typecheck
npm test
```

`npm test` usa el runner de Node y construye el binario si falta. Ninguna prueba escribe en
el tablero del repositorio: cada una monta su proyecto temporal y lo borra al terminar.
`npm run typecheck` es `tsc --noEmit` sobre el `tsconfig.json` de la extensión: no emite
nada, porque Node ejecuta los `.ts` tal cual. No activa `noUncheckedIndexedAccess`: las
pruebas indexan los mapas del arnés y cada acceso pasaría a ser un aviso que no comprueba
más código. El paquete que aporta el host a Pi —`@earendil-works/pi-coding-agent`— está como
devDependency, así que el typecheck ve sus tipos reales.

El plugin de OpenCode no trae dependencias de ejecución —el host le aporta su paquete—, pero
sí las de la comprobación:

```bash
cd extensions/opencode
npm install
npm run typecheck
npm test
```

`@opencode/plugin` no declara la superficie que usa el adaptador, así que sus tipos viven en
`types/opencode-plugin.d.ts`; el contexto que el host entrega lo tipa `tasks.ts`.
Reutiliza el tablero y el CLI de la extensión de Pi, y sustituye el CLI por uno falso:
comprueba el adaptador, no el producto.

Las dos extensiones viajan incrustadas en el binario para que el `init` las copie al
proyecto (`ADR-014`, `ADR-015`). Este repositorio es también su propio tablero, y lo lleva
con una instalación real: el binario en `.tasks-ai/bin/tasks-ai`, la extensión de Pi en
`.tasks-ai/extensions/pi` y el plugin de OpenCode en `.tasks-ai/extensions/opencode`. Son
copias de lo que hay en el repositorio, así que se refrescan con una orden —recompila el
binario y vuelve a copiar las extensiones—:

```bash
make install                               # refresca el binario, las extensiones y el contrato
```

`make install` copia también el contrato de etiquetas del repositorio, que tiene su
original en `internal/install/contract_tags.md` y viaja incrustado en el binario; una
prueba comprueba que las dos copias y la del proyecto son el mismo texto.

La copia de la extensión resuelve su dependencia `typebox` por un enlace a
`extensions/pi/node_modules`, que `make install` también deja puesto; sin él la extensión
no arranca. El paquete se registra en el archivo de ajustes de Pi del proyecto, versionado
(`.pi/settings.json`), con los comandos del host:

```bash
pi install -l .tasks-ai/extensions/pi      # registra la extensión instalada
pi remove -l --approve .tasks-ai/extensions/pi  # lo retira y deja la configuración como estaba
```

Con el paquete registrado se arranca Pi desde la raíz del proyecto —una sesión nueva, que
las que ya están abiertas siguen con la extensión que cargaron— y la extensión encuentra
el tablero subiendo desde el directorio de trabajo hasta `.tasks-ai/`, igual que el CLI,
ejecutando `.tasks-ai/bin/tasks-ai`. Sus herramientas y lo que llega al modelo están en
`contracts.md`; qué se gana con ellas, en el `README.md`; la forma, en `ADR-012` y
`ADR-014`.

OpenCode carga el plugin local que `init` deja en `.tasks-ai/extensions/opencode` con una
entrada en el `opencode.jsonc` del proyecto. Las listas de `plugins` de los archivos de
configuración se suman, así que añadir la del proyecto no retira las globales:

```jsonc
{
  "plugins": ["./.tasks-ai/extensions/opencode"]
}
```

El `.opencode/` del proyecto no guarda nada del adaptador: es el estado de la sesión del
arnés, y el repositorio lo ignora. La forma del plugin está en `ADR-015`.

## Los ADR

Una decisión durable vive en el tablero, como las tareas, y se escribe con el producto
(`ADR-016`). `docs/decisions/` es su **exportación generada**: un archivo por ADR y el
índice del directorio, versionados para que Git guarde la revisión y para que la puerta
de `adr-software` los revise. Editar un archivo exportado a mano no cambia la decisión:
la siguiente exportación lo revierte, y la marca del principio del archivo lo dice.

```bash
./.tasks-ai/bin/tasks-ai adr create --title "..." --deciders "..."   # ADR-001, en proposed
./.tasks-ai/bin/tasks-ai adr edit 1 --status accepted               # congela el cuerpo
./.tasks-ai/bin/tasks-ai adr export --dir docs/decisions            # escribe la exportación
./.tasks-ai/bin/tasks-ai adr export --dir docs/decisions --check    # sale con 1 si hay deriva
```

El cuerpo se congela al salir de `proposed`: después solo se admiten añadidos fechados en
«Más información», y un cambio de decisión se publica como un ADR nuevo que sustituye al
anterior. El sustituto nombra al anterior en «Más información» con el nombre de su archivo
—`--note "Sustituye a ADR-001 (001-la-primera-decision.md)."`, con el `slug` que da `adr
show`—: el producto guarda el enlace del sustituido y no reescribe el texto del sustituto, y
la puerta de `adr-software` es quien comprueba que el nombre está. Su resultado se nombra en
la tarea que cambia una decisión.

## Fronteras del código

- El núcleo contiene las reglas y el acceso a datos; no conoce comandos ni rutas.
- Cada superficie traduce su idioma al del núcleo: el CLI con sus comandos y su JSON,
  el servidor HTTP con sus rutas y sus códigos.
- La interfaz web es la de partida, ajustada a la API del núcleo. Vive en `web/`, la
  construye su propia herramienta de desarrollo y viaja incrustada en el binario:
  `serve` la reparte en la raíz y la API conserva sus rutas (`ADR-010`).
- La base solo la toca el núcleo.

## Convenciones

- Un módulo de Go en la raíz del repositorio.
- Capas sencillas con cableado manual por constructor, árbol de comandos con Cobra y
  sin Viper, y acceso directo a SQLite con migraciones aditivas propias (`ADR-007`,
  `ADR-009`).
- Sin ORM y sin capas de indirección sin justificación; una capa nueva es una decisión,
  no una comodidad.
- Los identificadores del código van en inglés; la documentación, los comentarios y el
  texto de la interfaz web, en español; la ayuda, los errores y los avisos del
  producto, en inglés (`ADR-011`).
- Los mensajes de commit van en inglés, con el formato del contrato de flujo.
- El fin de línea lo fija el repositorio en `.gitattributes`, no la configuración de Git
  de cada máquina: los archivos de texto se guardan y se escriben con LF.
- Los identificadores de los registros son de solo añadir: requisitos en `PRD.md`,
  riesgos en `known-risks.md` y decisiones en sus documentos.

## Cómo se trabaja

1. La tarea en el tablero antes de tocar archivos, y ninguna tarea se crea sin que la
   persona la haya confirmado.
2. Empezar el trabajo pone la tarea en curso (`update --status in_progress`), y dejarla
   la devuelve a `open`. Los avances, las decisiones y los bloqueos se comentan en el
   momento en que surgen, no al final.
3. Un cambio pequeño se hace y se verifica; un cambio grande va por fases, con la
   documentación al día en cada una.
4. Antes del commit: se lee la tarea entera —descripción, notas y comentarios— y se repasa
   su criterio punto por punto sobre lo entregado; se revisan el diff y la documentación
   viva —`README.md`, `docs/PRD.md`, `docs/architecture.md`, `docs/contracts.md`,
   `docs/known-risks.md` y `docs/web/`—, cada una en lo que el cambio le toque;
   y se ejecutan las comprobaciones:
   - `go test ./...`, que incluye la prueba de enlaces y títulos de los documentos y la
     de la guía de uso;
   - `make lint`;
   - `npm test` en `extensions/pi` y en `extensions/opencode`, si cambió una extensión;
   - `make check-web`, si cambió la interfaz;
   - `adr export --dir docs/decisions --check` y la puerta de `adr-software`, si el
     cambio toca una decisión o su exportación.
5. Se presenta el informe, se espera la confirmación y se crea el commit con
   `Refs: tk-<n>`. Se comprueba que el commit fue bien y se cierra la tarea con su motivo:
   el cierre es lo último.
6. Un cambio de diseño se documenta antes de implementarlo, y su ADR explica qué se
   descartó.

Los comandos del día a día son objetivos de `make`, con su equivalente de `go`:

```bash
make install    # compila el binario donde el tablero lo ejecuta y refresca extensión y contrato
make test       # go test ./...
make test-race  # go test -race ./...
make lint       # golangci-lint run ./...
make fmt        # golangci-lint fmt ./...
```

La interfaz se construye con su propia herramienta; el artefacto viaja incrustado en el
binario:

```bash
cd web && npm ci         # instala su herramienta de construcción
cd web && npm run build  # deja el artefacto en web/public/task-manager/
cd web && npm run watch  # la reconstruye al cambiar
```

## Ejecutar

| Comando | Qué hace |
|---|---|
| `./tasks-ai init` | Instala el producto en el proyecto: crea `.tasks-ai/` con la base, el settings y las instrucciones, y devuelve el JSON del resultado. Con `--prefix` fija el prefijo de las referencias en la propia instalación, en vez de `tk`, y escribe las instrucciones con él. No es repetible: un proyecto ya instalado lo rechaza |
| `./tasks-ai serve` | Arranca la API y la interfaz web en la dirección y el puerto del settings: la interfaz responde en la raíz del servidor y la API en sus rutas `/api/...` |
| `./tasks-ai <comando>` | Ejecuta cualquier operación del producto con salida JSON |
| `./serve-web.sh` | Deja en segundo plano un servidor de ficheros para leer `docs/web/` en el navegador, con su pid y su registro en `.run/`; `./serve-web.sh status` dice si está en marcha y dónde escucha, y `./serve-web.sh stop` lo para. La raíz es el repositorio y no `docs/web` porque las páginas enlazan `../PRD.md`, `../contracts.md` y `../../web/public/...`; el puerto es 8765 y `PORT` lo cambia en las tres órdenes. Reconoce el servidor por el puerto que escucha, así que también lo ve y lo para si se arrancó a mano con `python3 -m http.server`; un proceso ajeno en ese puerto no se toca: se nombra con su pid y su línea de órdenes, y el script sale con error |
| `./serve.sh` | Deja el servidor en segundo plano: compila el binario, lo arranca con `serve`, escribe su registro y su pid en `.run/` e informa de la dirección. `./serve.sh status` dice si está en marcha y `./serve.sh stop` lo para. Reconoce el servidor por el puerto que fija el settings, así que también lo ve y lo para si se arrancó a mano con `./tasks-ai serve`; un proceso ajeno en ese puerto no se toca: se nombra con su pid y su línea de órdenes, y el script sale con error |

El tablero del trabajo se opera con el mismo binario: sus comandos y sus reglas están en
[`.tasks-ai/AGENTS.md`](../.tasks-ai/AGENTS.md). El mapa de las fuentes del repositorio está
en [`AGENTS.md`](../AGENTS.md).

## Lo que no se hace

- No se edita la base de tareas con SQL.
- No se modifican los proyectos de referencia.
- No se añade una capa, una dependencia o una abstracción «por si acaso».
