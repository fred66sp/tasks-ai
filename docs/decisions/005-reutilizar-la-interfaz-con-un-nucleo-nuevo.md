# ADR-005: Reutilizar la interfaz y sustituir el servidor

- **Estado:** Aceptada
- **Fecha:** 2026-10-16

## Contexto

`project-task-manager` tiene una interfaz completa y en uso, con su servidor Node. El
análisis de esa interfaz mostró qué parte no depende del servidor —el árbol, los
filtros, la búsqueda, las preferencias y las hojas de estilo— y qué parte sí: la capa
de peticiones, el cálculo del estado bloqueado, las reglas de cierre y la cascada del
catálogo de etiquetas.

El Product Owner indicó que esa interfaz es la base del producto y precisó que no se
construye una interfaz nueva: solo se ajusta lo que exijan el núcleo nuevo y las
capacidades nuevas.

## Decisión

- Se reutiliza la interfaz existente; no se construye una nueva.
- El servidor Node se sustituye por el núcleo en Go.
- La capa de peticiones de la interfaz se ajusta a la API del núcleo.
- Lo que hoy calcula el servidor pasa al núcleo: el estado bloqueado, las reglas de
  cierre y la cascada de etiquetas.
- Las capacidades nuevas, el progreso de épicas y el recuento, se añaden a esa misma
  interfaz.

## Consecuencias

- El trabajo de interfaz se limita a lo que cambia de forma o de origen.
- La interfaz conserva lo que ya traía: un componente de aviso sin uso y endpoints
  que no consume. Se decidirá qué se hace con ellos al implementar.
- La forma de los datos que la interfaz pide deja de ser obligación del producto: la
  fija el núcleo, y la interfaz se ajusta.
- Requisitos afectados: `O-02`, `RF-02`, `NO-04`, `CA-08`, `CA-09`, `CA-10`.

## Alternativas consideradas

- **Construir una interfaz nueva.** Rechazada por el Product Owner: la existente ya
  resuelve lo que el producto necesita.
- **Conservar el servidor Node y envolverlo desde Go.** Rechazada: mantendría dos
  núcleos y las reglas de negocio acabarían duplicadas.
