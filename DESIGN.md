---
name: tasks-ai
description: Un tablero local dentro de una carpeta, leído por un agente y por una persona.
colors:
  canvas-night: "#0a0f1c"
  sidebar-night: "#0e1523"
  surface-night: "#111827"
  surface-raised: "#1f2937"
  surface-hover: "#252f3f"
  surface-selected: "#2a3750"
  surface-alt: "#141c2c"
  surface-inset: "#070b14"
  canvas-day: "#f8fafc"
  surface-day: "#ffffff"
  surface-soft-day: "#f1f5f9"
  ink-night: "#e5e7eb"
  ink-secondary-night: "#c3cad5"
  ink-muted-night: "#94a3b8"
  ink-day: "#0f172a"
  ink-muted-day: "#5f6b80"
  ink-disabled: "#6b7688"
  border-subtle-night: "#1e293b"
  border-default-night: "#334155"
  border-strong-night: "#475569"
  border-day: "#e2e8f0"
  border-strong-day: "#cbd5e1"
  violet-ink: "#a78bfa"
  violet-ink-hover: "#c4b5fd"
  violet-ink-day: "#6d28d9"
  violet-brand: "#8b5cf6"
  violet-brand-deep: "#7c3aed"
  azure-brand: "#3882f6"
  azure-ink-day: "#2563eb"
  cyan-brand: "#06b6d4"
  success: "#10b981"
  success-night: "#34d399"
  warning: "#f59e0b"
  warning-night: "#fbbf24"
  danger: "#f87171"
  danger-soft: "rgba(239, 68, 68, 0.14)"
  success-day: "#065f46"
  warning-day: "#92400e"
  danger-day: "#b91c1c"
  priority-high: "#fb923c"
  priority-medium: "#fbbf24"
  priority-low: "#60a5fa"
  code-canvas: "#070b16"
  tag-blue: "#62a7ff"
  tag-purple: "#b58bff"
  tag-green: "#58d392"
  tag-yellow: "#f2bf59"
  tag-orange: "#ff9a55"
  tag-red: "#ff7373"
  notice-surface: "#121c27"
  notice-panel: "#182431"
  notice-line: "#344657"
  notice-ink: "#f3f6f8"
  notice-ink-muted: "#bac4ce"
  overlay: "rgba(10, 16, 23, 0.78)"
  overlay-modal: "rgba(3, 7, 12, 0.78)"
  badge-border-warning: "rgba(242, 191, 89, 0.25)"
  badge-border-success: "rgba(88, 211, 146, 0.25)"
  badge-border-danger: "rgba(255, 115, 115, 0.25)"
  badge-border-info: "rgba(98, 167, 255, 0.25)"
typography:
  display:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "clamp(2.7rem, 5.4vw, 4.9rem)"
    fontWeight: 800
    lineHeight: 0.99
    letterSpacing: "-0.038em"
  headline:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "clamp(34px, 6vw, 62px)"
    fontWeight: 800
    lineHeight: 1.04
    letterSpacing: "-0.035em"
  title:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "clamp(22px, 3vw, 28px)"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.02em"
  body:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.55
  label:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "13px"
    fontWeight: 750
    lineHeight: 1.2
    letterSpacing: "0em"
  mono:
    fontFamily: "JetBrains Mono, ui-monospace, SF Mono, Menlo, Consolas, monospace"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
  doc-section:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "clamp(28px, 3.2vw, 40px)"
    fontWeight: 750
    lineHeight: 1.1
    letterSpacing: "-0.03em"
  doc-body:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.65
  doc-label:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "11px"
    fontWeight: 700
    letterSpacing: "0.1em"
  doc-table-label:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "11.5px"
    fontWeight: 700
    letterSpacing: "0.06em"
  doc-compact:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "12px"
    fontWeight: 700
    letterSpacing: "0.05em"
  doc-button:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "14.5px"
    fontWeight: 650
  task-title:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "26px"
    fontWeight: 750
    lineHeight: 1.24
    letterSpacing: "-0.02em"
  panel-h3:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "18px"
    fontWeight: 700
  modal-h2:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "21px"
    fontWeight: 700
rounded:
  xs: "4px"
  sm: "6px"
  md: "10px"
  notice: "8px"
  lg: "16px"
  pill: "999px"
spacing:
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
components:
  button-primary:
    backgroundColor: "{colors.violet-ink}"
    textColor: "{colors.canvas-night}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "42px"
  button-primary-hover:
    backgroundColor: "{colors.violet-ink-hover}"
  button-secondary:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.ink-night}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "42px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink-secondary-night}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "42px"
  button-danger:
    backgroundColor: "{colors.danger-soft}"
    textColor: "{colors.danger}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "42px"
  button-compact:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.ink-night}"
    rounded: "{rounded.sm}"
    padding: "0 8px"
    height: "24px"
  input-default:
    backgroundColor: "{colors.surface-inset}"
    textColor: "{colors.ink-night}"
    rounded: "{rounded.sm}"
    padding: "0 13px"
    height: "42px"
  badge-status:
    rounded: "{rounded.pill}"
    padding: "2px 9px"
    height: "26px"
  panel:
    backgroundColor: "{colors.surface-night}"
    rounded: "{rounded.md}"
  property-card:
    backgroundColor: "{colors.surface-inset}"
    rounded: "{rounded.sm}"
    padding: "12px"
  task-row:
    backgroundColor: "{colors.sidebar-night}"
    textColor: "{colors.ink-night}"
    rounded: "{rounded.sm}"
    padding: "7px"
    height: "68px"
---

# Design System: tasks-ai

## Overview

**Creative North Star: "El archivador del proyecto"**

Este documento explica cómo aplicar la identidad aprobada al tablero, la guía de uso y la
portada. La [guía navegable](docs/web/design-system.html) dibuja las muestras con los valores
de esas fuentes; este contrato fija las reglas sin abrir una segunda paleta ni describir Tacks
como producto actual.
Cada dato se modifica en su fuente responsable:

| Dato | Fuente responsable | Papel de este documento |
|---|---|---|
| Zorro, colores y familias de marca | [`docs/tasks-ai-brand/`](docs/tasks-ai-brand/README.md) | Uso y restricciones; el paquete aprobado no se modifica. |
| Color, densidad, radios y tipografía del tablero | [`web/src/task-manager/styles/tokens.css`](web/src/task-manager/styles/tokens.css) y sus [componentes](web/src/task-manager/styles/components.css) | Criterios de aplicación y correspondencia con la interfaz efectiva. |
| Temas y composición de la documentación | [`docs/web/index.html`](docs/web/index.html) y [`docs/web/landing.html`](docs/web/landing.html) | Reglas compartidas y excepciones explícitas de cada superficie. |
| Iconos de la interfaz | [`web/public/icons/heroicons/24/outline/`](web/public/icons/heroicons/24/outline/) y su [`LICENSE`](web/public/icons/heroicons/LICENSE) | Origen único, tamaño, color y nombre accesible; el archivo no se redibuja. |
| Capacidades, estados y contratos de tarea | [`docs/PRD.md`](docs/PRD.md) y [`docs/contracts.md`](docs/contracts.md) | Solo representación visual; no redefine el negocio. |

El frontmatter conserva los valores necesarios para las herramientas que leen `DESIGN.md`;
es una proyección de esas fuentes, no el lugar donde se decide una paleta nueva. Los ejemplos
de [`.impeccable/design.json`](.impeccable/design.json) son muestras para el panel visual:
se contrastan con el CSS al cambiarlo, nunca se usan para construir el producto. El HTML
de [Tacks](docs/02-gestor-tareas-system-desing_v2.html) es un antecedente histórico de patrones,
no un contrato ni una fuente de tokens de tasks-ai.

El contrato [`design-system.json`](design-system.json) proyecta estas fuentes en el esquema
`design-system.v1`: identidad, tokens del tablero, componentes efectivos y gobernanza. No
decide por su cuenta ni abre una segunda paleta; una prueba
(`web/src/task-manager/design-tokens.test.js`) lo contrasta con `tokens.css`, y su puerta es
`check` de la skill `design-system-architect`. El catálogo sigue siendo la
[guía navegable](docs/web/design-system.html); el contrato no genera otro.

El mundo es un archivador, no una aplicación: todo el trabajo de un proyecto vive en un cajón
—`.tasks-ai/`—, y el tablero es el expediente que se consulta. Por eso la pantalla no
presenta nada ni se presenta a sí misma; ordena. Dos columnas y una cabecera, una fila por
tarea, un panel donde se lee y se corrige lo que hay. La persona y el agente miran el mismo
expediente, así que la jerarquía sirve tanto a un vistazo rápido como a una lectura larga.

La voz es **densa y precisa**. El adorno tiene un sitio reservado y pequeño: el zorro de la
marca, una línea de degradado, el violeta de la selección. Todo lo demás es instrumento:
medidas repetidas, bordes de un píxel, estados nombrados con texto y no solo con color. La
paleta es nocturna en la interfaz y dual en la documentación; el código es oscuro en los dos
temas, porque un bloque de comandos no cambia de significado según la hora.

**Key Characteristics:**

- Archivador antes que escenario: dos columnas, filas de 68 px y un panel de lectura, sin
  rejillas de tarjetas como estructura de página.
- Densidad funcional: los controles son de 42 px, los compactos de 36 y 24, y las filas del
  árbol de 68; el espacio se gasta en separar grupos, no en aire decorativo.
- Neutros por capas: el fondo, la barra lateral, la superficie, el realce y la fila
  seleccionada son pasos del mismo azul nocturno.
- El violeta pertenece a la marca y al foco; cuando escribe, usa su variante legible.
- Un degradado de marca solo en la portada, y el primario de violeta a azul en la guía y en
  la portada; ninguna otra decoración de color.

## Colors

Paleta nocturna de marca sobre neutros azulados, con un violeta que selecciona y un trío de
estados que siempre se acompaña de texto.

### Primary

- **Violeta que escribe** (`#a78bfa`): el violeta de la interfaz. Pinta el texto de enlace y
  el icono de acento, la pestaña activa, la barra de la fila seleccionada, el botón primario y
  el anillo de foco. Es un paso aclarado del violeta de marca porque el de marca no llega al
  contraste mínimo sobre la superficie oscura (4,19:1).
- **Violeta de marca** (`#8b5cf6`): pintura, no tinta. Vive en el símbolo, en la cabecera de
  marca y en el degradado de la portada. No lleva texto.
- **Violeta hondo** (`#7c3aed`): el violeta de marca un paso oscurecido; rellena el botón
  primario de la documentación porque lleva texto blanco y el de marca no alcanza el
  contraste. Tampoco es tinta.
- **Violeta gastado** (`#6d28d9`): el mismo acento en la guía y la portada en tema claro, un
  paso oscurecido del violeta de marca para que el texto cumpla AA sobre fondo casi blanco.

### Secondary

- **Azul de apoyo** (`#2563eb` en tema claro; `#3882f6` como parada de marca): segunda voz de
  la documentación. Aparece en el degradado del botón primario y en la malla de la portada;
  en la interfaz solo existe como parte de la rampa de prioridad.

### Tertiary

- **Cian de cierre** (`#06b6d4`): la última parada del degradado de marca, en la regla de dos
  píxeles bajo la portada y en su malla. Nunca comunica un estado.

### Neutral

- **Noche** (`#0a0f1c`): el lienzo de la interfaz.
- **Barra lateral** (`#0e1523`): el explorador de tareas y la cabecera.
- **Superficie** (`#111827`): paneles, modales, tarjetas de documentación en tema oscuro.
- **Realce** (`#1f2937`): botones secundarios, insignias neutras, superficies elevadas.
- **Selección** (`#2a3750`): la fila elegida en el árbol y el botón en hover.
- **Hueco** (`#070b14` en el tablero, `#070b16` en los bloques de código de la guía):
  superficies hundidas propias de cada soporte.
- **Tinta** (`#e5e7eb` / `#c3cad5` / `#94a3b8` / `#6b7688`): texto principal, secundario,
  apagado y deshabilitado, en pasos que se distinguen sin mirar el tamaño.
- **Bordes** (`#1e293b` / `#334155` / `#475569`): separación, control y énfasis; también el
  color de la barra de desplazamiento, que se hereda en toda la interfaz.
- **Día** (`#f8fafc`, `#ffffff`, `#f1f5f9`, `#0f172a`, `#5f6b80`, `#e2e8f0`, `#cbd5e1`): el
  tema claro de la guía y la portada, los mismos papeles con los valores del paquete.

### Named Rules

**El violeta que escribe.** El violeta de marca (`#8b5cf6`) no lleva texto nunca. Si el
violeta nombra algo, es `#a78bfa` sobre oscuro o `#6d28d9` sobre claro.

**La gradiente tiene una sola casa.** El degradado de marca —violeta `#8b5cf6`, azul
`#3882f6`, cian `#06b6d4`— vive en la portada: la palabra marcada del titular, la malla y la
regla de dos píxeles que la cierra. En la guía solo el botón primario lleva un degradado, y es
de violeta a azul. En el tablero no aparece ninguno.

**El estado no va solo en color.** Éxito, aviso y peligro llevan siempre palabra, icono o
forma además del tono; cada uno tiene su relleno suave al 14–16 % para el fondo de insignias.
Sobre papel claro su texto se oscurece a `#065f46`, `#92400e` y `#b91c1c` para mantener AA.

## Typography

**Display Font:** Inter (con `ui-sans-serif` y la pila del sistema como respaldo)
**Body Font:** Inter (la misma pila)
**Label/Mono Font:** JetBrains Mono (con `ui-monospace`, SF Mono, Menlo, Consolas)

**Character:** Una sola familia para todo lo que se lee, con el peso haciendo el trabajo y la
monoespaciada reservada a lo que se escribe o se mide: comandos, JSON, identificadores
`tk-14` y los códigos de salida. Las fuentes no se redistribuyen: si no están instaladas, la
pila cae en la del sistema sin descargar nada.

### Hierarchy

- **Display** (800, `clamp(2.7rem, 5.4vw, 4.9rem)`, 0.99, −0.038em): el titular de la
  portada, una frase de 11 caracteres de ancho como máximo, equilibrada con `text-wrap:
  balance`. No se usa en la interfaz.
- **Headline** (800, `clamp(34px, 6vw, 62px)`, 1.04, −0.035em): el titular de la guía.
- **Title** (700, `clamp(22px, 3vw, 28px)`, −0.02em): los títulos de sección de la guía; en la
  interfaz equivalen el `h3` del panel (18 px) y el `h2` del modal (21 px).
- **Section** (750, `clamp(28px, 3.2vw, 40px)`, 1.1, −0.03em): los títulos de sección de la
  portada; su titular de cierre usa Headline.
- **Body** (400, 15 px, 1.55): el texto de la interfaz; la guía lee a 16 px y 1.65, con las
  líneas de apoyo limitadas a `74ch`, la última línea de cada párrafo repartida con
  `text-wrap: pretty` y el texto corrido a la anchura de la columna —acotarlo corta los
  párrafos contra tablas, tarjetas y código, que la ocupan entera—; el párrafo de portada
  va a 46.
- **Label** (750, 13 px, mayúsculas): etiquetas de propiedad y de campo. En las tablas de la
  guía bajan a 11,5 px con 0,06em de tracking; el índice lateral usa 11 px y 0,1em.
- **Mono** (400, 13 px, 1.5): CLI, código, JSON, rutas y referencias de tarea.

### Named Rules

**La jerarquía es de peso.** Dos textos del mismo tamaño se separan por peso antes que por
color: los títulos de fila van a 720, las etiquetas a 750, las insignias a 780 y los titulares
a 800. Nada que nombre algo baja de 500.

**El titular no lleva ceja.** Ni la guía ni la portada ponen una etiqueta pequeña encima de un
título; el título aguanta solo.

## Layout

La interfaz ocupa la ventana entera (`100dvh`): una cabecera de 68 px y un cuerpo en dos
columnas, el explorador con `minmax(600px, 38%)` y el inspector con el resto. A 1500 px el
explorador se fija en 560 px; por debajo de 1180 la aplicación conserva 1100 px de ancho
mínimo y la ventana desplaza; a 720 px el explorador toma la pantalla y el inspector pasa a
hoja fija de pantalla completa con un control de vuelta. Las propiedades de la tarea van en
una rejilla de cuatro columnas que baja a dos a 1500 px y a dos estrechas a 720.

La densidad es parte del sistema: filas del árbol de 68 px con 24 px de sangría por nivel,
controles de 42 px, compactos de 36 y 24, cabecera de panel de 52 px. El ritmo de espacio es
8/12/16/20/24: apretado dentro de un grupo, generoso entre grupos, y siempre más aire encima
de un título que debajo.

La documentación es una columna de 1180 px con márgenes de `clamp(16px, 4vw, 40px)`; la
barra superior es fija, translúcida (`color-mix` al 82 % con `saturate(180%) blur(14px)`) y
deja 88 px de reserva al desplazarse a un ancla. La portada alterna tramos de fondo claro con
una banda nocturna y una rejilla de dos columnas que colapsa a una a 720 px.

### Named Rules

**El modal no edita.** Un modal solo crea o confirma —crear tarea, crear etiqueta, borrar—;
lo que se edita se edita en el inspector, que es la segunda columna y no interrumpe. Si una
tarea necesita foco protegido, se protege con el panel, no abriendo otra ventana.

## Elevation & Depth

El sistema es plano por capas: cinco neutros del mismo azul nocturno —lienzo, barra lateral,
superficie, realce, selección— dan la profundidad sin sombra. La sombra queda para lo que de
verdad flota sobre la pantalla: el modal, el aviso emergente y, en la documentación, las
tarjetas y tablas. El lienzo de la interfaz lleva además un radial azul al 7 % en la esquina
superior derecha, que separa la cabecera del vacío sin dibujar nada.

### Shadow Vocabulary

- **Flotante** (`0 30px 90px rgba(0, 0, 0, 0.52)`): modales y avisos emergentes, siempre sobre
  una capa de velo (`rgba(3, 7, 12, 0.78)`).
- **Papel claro** (`0 1px 2px rgba(15, 23, 42, 0.05), 0 8px 24px -12px rgba(15, 23, 42, 0.18)`):
  tarjetas y tablas de la guía en tema claro; su pareja oscura es
  `0 1px 2px rgba(0, 0, 0, 0.4), 0 12px 32px -18px rgba(0, 0, 0, 0.8)`.
  La escala grande (`0 20px 60px -30px` claro, `0 24px 70px -34px` oscuro) es solo para
  superposición de sección.
- **Artefacto** (`0 1px 1px rgba(2, 6, 23, 0.18), 0 28px 70px -32px rgba(2, 6, 23, 0.6)`):
  el tablero de ejemplo de la portada, que se presenta como una pieza sobre la mesa.
- **Foco** (`0 0 0 3px rgba(139, 92, 246, 0.16)`): el halo del campo activo, siempre junto al
  borde violeta, nunca solo.

### Named Rules

**La sombra solo flota.** Si algo no se superpone a la pantalla, no lleva sombra: la
profundidad de un panel es su capa neutra. Un halo de color sin desplazamiento es decoración y
no entra.

## Shapes

La forma es de herramienta: esquinas pequeñas y bordes de un píxel. Los controles, las filas y
las insignias de campo usan 6 px; los paneles, la marca y las tarjetas de la interfaz, 10 px;
los modales y las tarjetas y tablas de la documentación, 16 px. Las insignias y las etiquetas
del árbol son píldoras de 999 px. El símbolo de la marca vive en un cuadrado de 42 px en la
cabecera del tablero —34 en la guía— con 10 px de esquina y sin fondo propio.

La geometría que distingue al sistema no es el radio sino la línea: 1 px de borde en todo
control, 3 px de barra violeta a la izquierda de la fila seleccionada, 2 px de subrayado en la
pestaña activa, 2 px de degradado bajo la portada, 1 px discontinuo en el estado vacío. Las
líneas de anidado del árbol —1 px al 70 % de opacidad— son la única guía vertical de la
interfaz.

## Components

### Buttons

Herramienta con etiqueta: altura completa, radio corto, peso alto y transición de 120 ms en
fondo, borde y color.

- **Shape:** 6 px de esquina, 42 px de altura, 16 px de relleno lateral, 750 de peso.
- **Primary:** fondo `#a78bfa`, texto `#0a0f1c`; en hover pasa a `#c4b5fd`. El primario de la
  guía y la portada es distinto a propósito: un degradado de violeta a azul —la guía lo toma
  de las tintas del tema, la portada del violeta hondo `#7c3aed` al azul `#2563eb`—, 10 px de
  esquina, 650 de peso y 14,5 px de texto; la portada lo fija en 44 px de alto y la guía en
  11×18 px de relleno, con `translateY(-1px)` al pasar por encima.
- **Secondary:** borde `#334155` sobre `#1f2937`; el hover sube el borde a `#475569` y el fondo
  a `#252f3f`.
- **Ghost:** sin fondo, texto `#c3cad5`; el hover pinta `#252f3f`. Es la acción de cabecera y de
  barra de panel.
- **Danger:** relleno rojo al 14 %, borde rojo al 28 %, texto `#f87171`.
- **Compact:** 24 px de alto, 12 px de texto, 5 px de hueco; solo cabecera y barras de panel.
  Por debajo de 720 px pierde la etiqueta y conserva el icono.

### Chips

- **Style:** píldora de 26 px, 13 px de texto a 780, borde del tono al 25 % y relleno al
  14–16 %. El tono neutro usa `#1f2937` con borde `#1e293b`.
- **State:** el estado de una tarea y la prioridad son insignias, no etiquetas libres; la
  prioridad tiene su propia rampa —crítica `#f87171`, alta `#fb923c`, media `#fbbf24`, baja
  `#60a5fa`— y las etiquetas del catálogo llevan su color con punto.

El estado persistido de una tarea es `open`, `in_progress` o `done`. «Bloqueada» se calcula a
partir de sus dependencias; no es un cuarto estado guardado. El árbol combina estado y
prioridad con palabras y controles, no con un color aislado. El catálogo de etiquetas aporta
los colores de cada etiqueta: su tono no es un estado ni una prioridad. La semántica de
estos datos vive en [`docs/PRD.md`](docs/PRD.md), no en las muestras visuales.

### Cards / Containers

- **Corner Style:** 10 px en la interfaz, 16 px en la documentación.
- **Background:** `#111827` el panel, `#070b16` la tarjeta de propiedad, `#ffffff` o `#111827`
  la tarjeta de la guía.
- **Shadow Strategy:** ninguna en la interfaz; la de papel solo en la documentación (ver
  Elevation & Depth).
- **Border:** 1 px de `#1e293b`; el estado vacío cambia a 1 px discontinuo de `#334155`.
- **Internal Padding:** 16 px de contenido, 12 px en la tarjeta de propiedad, 20 px en la de la
  guía.

### Inputs / Fields

- **Style:** fondo hundido `#070b16`, borde `#334155`, 6 px de esquina, 42 px de alto, 13 px de
  relleno; el área de texto parte de 118 px y crece en vertical.
- **Focus:** borde violeta y halo de 3 px al 16 %, sin `outline` propio. El foco de teclado del
  resto de controles sí usa `outline: 2px solid #a78bfa` con 2 px de separación.
- **Error / Disabled:** el error se escribe debajo del campo en `#f87171`; lo deshabilitado baja
  a 0,58 de opacidad y no se puede pulsar.

### Notices

Las notificaciones y la confirmación no reutilizan el modal del producto: usan su propio juego
(`notice-surface`, `notice-panel`, `notice-line`, `notice-ink`) y un radio de 8 px
(`rounded.notice`), sobre un velo `overlay` con desenfoque. Su dueño es
`web/src/task-manager/notifications/toast-anchor.js`; aquí solo se declaran los valores. La
confirmación nombra lo que se pierde y la acción destructiva va en rojo; el aviso emergente
solo confirma que algo se guardó y no sustituye a un error que exige acción.

### Navigation

La cabecera de 68 px lleva el símbolo de 42 px, el nombre del producto y una segunda línea de
13 px apagada; a la derecha, acciones compactas. En la interfaz no hay más navegación: el
árbol es el índice. En la guía la barra superior fija lleva marca, selector de idioma y de
tema, y el índice lateral marca la sección activa; las pestañas de la interfaz son texto de
15 px a 750 separadas 28 px, con 2 px de violeta bajo la activa.

### Signature Components

**La fila del árbol.** La pieza que define el producto: 68 px de alto, rejilla de
`34 / 34 / 1fr / 120 / 130 / 38` —expansor, tipo, título, prioridad, estado, acciones—, 24 px
de sangría por nivel con una guía vertical de 1 px. Al seleccionarla, fondo `#2a3750`, borde
azul al 38 % y 3 px de violeta a la izquierda. El título va a 16 px y 720; la meta, a 13 px
apagados.

**El aviso emergente.** 400 px como máximo, arriba a la derecha, sobre `#1f2937`, con una
insignia circular de éxito, la sombra flotante y una entrada de 180 ms desde −8 px. Es la única
confirmación de que algo se guardó.

**El inspector.** Se abre al seleccionar una tarea y permite leer y corregir el expediente;
en móvil ocupa la pantalla y ofrece una acción para volver al árbol. Los diálogos crean o
confirman; los filtros y paneles no convierten una edición normal en un modal. Si no hay tarea
seleccionada, el inspector explica cómo abrirla. Carga, vacío y error conservan el mismo
marco y una salida comprensible; el indicador de carga y el color de error no sustituyen el
texto. Los controles que solo muestran un icono tienen nombre accesible y foco visible.

## Iconography

**El trazo es instrumento.** El conjunto de trabajo de la interfaz es
[Heroicons 24/outline](web/public/icons/heroicons/24/outline/): caja de 24,
`viewBox="0 0 24 24"`, trazo de 1,5 px, extremos y uniones redondeados y
`stroke="currentColor"`. La licencia MIT viaja con los archivos en
[`LICENSE`](web/public/icons/heroicons/LICENSE).

- **Tamaño:** 16 px en la fila y el control compacto, 18 px en la cabecera y el botón de
  icono, 20 px en el panel. El icono no se escala fuera de su caja.
- **Color:** hereda el color del texto del control; no se recolorea por archivo ni se usa
  como decoración. El violeta de acento solo cuando el control es el acento.
- **Nombre:** un control de solo icono lleva `aria-label` y `title`; el icono decorativo va
  con `aria-hidden="true"`. El icono no sustituye a la palabra en crear, cerrar, guardar,
  eliminar o ejecutar.
- **Origen:** un icono nuevo entra en el conjunto antes de usarse; no se mezcla otro juego
  ni se redibuja el trazo del archivo.
- **Sustitución:** cuando un componente adopta un archivo del conjunto, cambian el componente
  y su muestra en la misma unidad y se conserva el nombre accesible. Mientras un componente
  dibuje su propio trazo, ese trazo es el implementado.

La [guía navegable](docs/web/design-system.html) cataloga los 72 archivos agrupados y enlaza
cada uno desde la ruta pública; la interfaz sirve el conjunto en `/icons/`.

## Do's and Don'ts

### Do:

- **Do** elegir el violeta que escribe según el fondo: `#a78bfa` sobre oscuro, `#6d28d9` sobre
  claro.
- **Do** construir la profundidad con los cinco neutros antes de recurrir a una sombra.
- **Do** acompañar cada estado con palabra o icono además del color, y mantener la insignia en
  su píldora de 26 px.
- **Do** mantener las medidas de densidad: 68 px de fila, 42 px de control, 24 px de sangría,
  8/12/16/20/24 de ritmo.
- **Do** dejar el código oscuro en los dos temas, con el fondo `#070b16` o `#0a0f1c`.
- **Do** usar un icono del conjunto cuando la acción sea convencional y acompañarlo de texto
  en crear, cerrar, guardar, eliminar o ejecutar.
- **Do** respetar `prefers-reduced-motion`: la entrada de la portada y el pulso del filtro se
  apagan, y el resto de transiciones no pasan de 180 ms.

### Don't:

- **Don't** usar `#8b5cf6` como color de texto ni de icono sobre superficie oscura.
- **Don't** poner el degradado de marca en el tablero, en una insignia o en texto corrido.
- **Don't** estructurar una página con rejillas de tarjetas iguales de icono, título y párrafo,
  ni anidar tarjetas.
- **Don't** poner una ceja o etiqueta pequeña encima de un titular.
- **Don't** usar cristal, desenfoque o halos de color como adorno; el único desenfoque
  decorativo es la malla de la portada.
- **Don't** añadir una sombra a un panel que no se superpone, ni un borde lateral de color de
  más de 1 px salvo la barra de 3 px de la fila seleccionada.
- **Don't** introducir una familia tipográfica nueva ni redistribuir Inter o JetBrains Mono:
  el paquete de marca no las incluye.
- **Don't** mezclar iconos de otro conjunto, escalar un icono fuera de su caja de 24 ni
  recolorear un archivo del conjunto.
- **Don't** redibujar, recolorear ni vectorizar el zorro: el paquete trae el arte bitmap en los
  tamaños que enumera.
