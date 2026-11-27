# Producto tasks-ai

<!-- impeccable:product-schema 1 -->

Registro durable del producto para el trabajo de diseño. Los requisitos, su alcance y sus
criterios de aceptación son de [`docs/PRD.md`](docs/PRD.md); este documento no los repite:
enlaza.

## Platform

web

## Users

**Los agentes son el consumidor primario.** Operan el tablero por el CLI y consumen su salida
JSON; no leen texto preparado para personas.

**La persona** gobierna el trabajo de su proyecto: decide qué entra, revisa el estado y
comprueba lo entregado. Lo hace sobre el mismo tablero desde la interfaz web, y desde el CLI
cuando le conviene. No hay un tercer usuario: un solo proyecto, una sola persona.

## Product Purpose

Gestor de tareas local-first para un proyecto, escrito en Go y con las tareas en una base
SQLite. Un solo núcleo de negocio sostiene dos superficies: el CLI con JSON y la interfaz web
con su API HTTP. El tablero guarda el trabajo —tareas anidadas, su especificación, sus
comentarios, sus bloqueos— y también las decisiones durables del proyecto.

El éxito tiene tres caras, y las tres cuentan:

- el ciclo de una tarea se cierra sin fricción: crearla, trabajarla, comentarla y cerrarla
  sin preparar nada antes, y el agente puede recorrerlo entero con JSON;
- el tablero es la única fuente de verdad, sin listas paralelas: lo que no está en él no
  entra en el trabajo;
- instalar en un proyecto nuevo deja dentro solo `.tasks-ai/` y no ensucia nada más.

## Positioning

El registro del trabajo vive dentro del proyecto, en una carpeta, y el agente lo lee con
JSON en lugar de interpretar prosa. Un producto con servicio remoto no puede copiar eso sin
dejar de ser lo que es: aquí no hay servicio remoto, ni cuenta, ni sincronización, ni ámbito
que no sea el proyecto donde está la carpeta.

## Operating Context

La persona trabaja en su máquina, dentro de un solo proyecto, con una terminal y un agente
al lado. Instala el producto una vez —`init`— y desde entonces el tablero es
el registro: el agente propone, la persona confirma y cierra, y el commit es lo que guarda
la historia. La interfaz web se levanta desde el mismo binario con `serve`, y se usa para
mirar y corregir el tablero, no para administrar un servicio.

## Capabilities and Constraints

Dueño de los requisitos: [`docs/PRD.md`](docs/PRD.md). Dueño de las formas de datos y de la
API: [`docs/contracts.md`](docs/contracts.md).

- Un proyecto y un ámbito de configuración; nada global, nada multiusuario.
- Un solo binario: CLI, servidor y API. La interfaz va incrustada y se compila antes.
- Acceso a datos directo sobre SQLite, sin ORM.
- Varias sesiones y agentes escriben sobre la misma base; la interfaz recarga cuando los
  datos cambian desde fuera.
- El texto de una tarea no guarda el estado del trabajo: se rechaza.
- La ayuda del CLI y los mensajes de error del producto van en inglés; la documentación, el
  código, los comentarios y el texto de la interfaz, en español.
- La interfaz de partida se reutiliza y se ajusta; no se construye una nueva.

## Brand Commitments

Paquete aprobado y cerrado: [`docs/tasks-ai-brand/`](docs/tasks-ai-brand/). Se usa tal cual y
su cabecera se conserva.

- Símbolo: zorro con guiño pícaro, arte bitmap aprobado. No se reinterpreta, redibuja,
  vectoriza, deforma ni recolorea; el paquete no incluye SVG.
- Paleta de marca: violeta, azul y cian, con degradado; estados con éxito, aviso y error, y
  tema claro y oscuro desde `tokens.css` / `tokens.json`.
- Tipografía: Inter para interfaz y documentación; JetBrains Mono para CLI, código e
  identificadores. Las fuentes no se redistribuyen.
- Densidad funcional por delante del adorno; el estado nunca se comunica solo por color.

## Evidence on Hand

- `docs/tasks-ai-brand/`: arte maestro, símbolo y composiciones, iconos de aplicación,
  favicons, previsualizaciones y tokens en CSS y JSON. Los iconos existen en los tamaños que
  el paquete enumera; no se generan nuevos.
- `docs/web/index.html`: la guía de uso del producto, con ejemplos reales de comandos y
  respuestas, en español y en inglés.
- `docs/web/landing.html`: la presentación del producto y su entrada a la guía.
- `docs/decisions/`: las decisiones durables exportadas desde el tablero.

No hay capturas de referencia ni pruebas visuales comprometidas, ni testimonios, clientes,
métricas de uso o banco de pruebas. El trabajo de diseño no puede fabricarlos.

## Product Principles

1. **El tablero es la única fuente de verdad.** Lo que no está registrado no entra en el
   trabajo, y nada mantiene una lista paralela.
2. **JSON es la interfaz de primera clase.** Una capacidad que el agente no puede completar
   por el CLI no está terminada.
3. **Una carpeta, un proyecto.** Todo lo del producto vive dentro de `.tasks-ai/`; instalar
   no deja rastro fuera de ella.
4. **El texto describe, las herramientas marcan el estado.** Ni el título ni las notas
   escriben en qué punto está el trabajo.
5. **Se ajusta lo que existe antes de añadir.** Cada capacidad nueva pesa contra el núcleo
   compartido y las dos superficies que ya hay.

## Accessibility & Inclusion

Contraste WCAG AA, foco visible, recorrido completo por teclado, etiquetas en los campos,
respeto de `prefers-reduced-motion` y uso correcto a 200 % de zoom. El estado no se comunica
solo por color.
