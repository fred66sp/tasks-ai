# PRD del producto tasks-ai

| Campo | Valor |
|---|---|
| Producto | tasks-ai |
| Autor | Alfredo González Portero |
| Misión del documento | Lo que el producto debe cumplir: propósito, usuarios, alcance, requisitos y criterios de aceptación |
| Fuentes | Decisiones de producto acordadas con el Product Owner y los análisis registrados en el tablero ODD del proyecto |

Este documento describe lo que el producto debe ser, no cómo se construye. El diseño
vive en `docs/architecture.md` y las formas de datos en `docs/contracts.md`; el
contexto de cada decisión, en `docs/decisions/`.

---

## 1. Propósito

tasks-ai es un gestor de tareas local-first escrito en Go: los datos y el servicio
viven en la máquina de la persona, sin servicio remoto. Atiende a un único proyecto.

El proyecto conserva sus tareas en una base SQLite. Sobre el mismo núcleo de negocio
hay dos superficies: un CLI con salida JSON, pensado para agentes, y una interfaz web
con su API HTTP, pensada para la persona. Todo lo del producto vive en una sola
carpeta dentro del proyecto, de modo que el proyecto no se ensucia.

Tacks es la referencia funcional del producto. Inspira el modelo de tareas; no obliga
a conservar su compatibilidad ni a traducir su implementación.

## 2. Usuarios y casos de uso

**La persona** gobierna el trabajo de su proyecto sin salir de la máquina.

**Los agentes** operan por CLI y consumen JSON; no leen texto preparado para personas.

| ID | Caso de uso |
|---|---|
| CU-01 | Operar las tareas del proyecto desde el CLI, con salida JSON |
| CU-02 | Operar las mismas tareas desde la interfaz web |
| CU-03 | Ajustar la configuración del proyecto |
| CU-04 | Definir el prefijo de las referencias visibles |
| CU-05 | Instalar el producto en un proyecto nuevo, dejando dentro solo la carpeta del producto |

## 3. Alcance aprobado

### Objetivos

| ID | Objetivo |
|---|---|
| O-01 | Producto local-first escrito en Go, con las tareas en una base SQLite |
| O-02 | Un solo núcleo de negocio compartido por el CLI y por la API HTTP |
| O-03 | CLI y JSON como interfaces de primera clase, aptas para agentes |

### Interfaz

El producto reutiliza la interfaz de `project-task-manager`. No se construye una
interfaz nueva: solo se ajusta lo que exijan el núcleo nuevo y las capacidades que se
hayan aprobado.

### Modelo de tarea

Título, descripción, estado, prioridad numérica, responsable, tarea padre, etiquetas,
marcas de tiempo, motivo de cierre y notas. Los estados guardados son pendiente
(`open`), en curso (`in_progress`) y hecha (`done`): abierta y cerrada no son estados,
abierta agrupa pendiente y en curso, cerrada es hecha, y reabrir es volver a poner la
tarea en pendiente. Bloqueada se calcula desde las dependencias y no se guarda.

### Capacidades

| Capacidad | Qué incluye |
|---|---|
| Alta | Crear tarea y subtarea con título, descripción, prioridad, etiquetas y tarea padre |
| Detalle | Consultar y editar título, descripción, notas y responsable |
| Estado y prioridad | Cambiarlos desde el listado y desde el detalle |
| Cierre | Cerrar con motivo y comentario de cierre |
| Dependencias | Alta y baja de bloqueos, y ver qué bloquea a qué |
| Comentarios | Historial de solo añadir en cada tarea |
| Etiquetas | Catálogo con color y descripción: alta, renombrado y borrado, con aviso del uso antes de borrar |
| Listado | Árbol con jerarquía visible, filtros por estado, prioridad y etiqueta, búsqueda por texto y vista por periodo |
| Épicas | Marcar frentes de nivel superior y mostrar su progreso |
| Recuento | Trabajo abierto del proyecto |
| Preferencias | Recordar en el navegador lo último que miró la persona |
| Avisos | Avisar cuando los datos cambian desde fuera de la interfaz |
| Doble superficie | Ofrecer cada capacidad por CLI con JSON y por la interfaz |

### No objetivos

| ID | No objetivo |
|---|---|
| NO-01 | Conservar la compatibilidad con Tacks: es referencia funcional, no una obligación de compatibilidad ni una traducción de Rust |
| NO-02 | Sustituir o modificar el gestor de tareas con el que se desarrolla: son herramientas distintas |
| NO-03 | Atender varios proyectos a la vez y mantener una configuración global: el producto tiene un único proyecto y un único ámbito de configuración |
| NO-04 | Construir una interfaz nueva: se reutiliza la existente y solo se ajusta lo necesario |

## 4. Requisitos aprobados

### Funcionales

| ID | Requisito |
|---|---|
| RF-01 | Toda operación del producto se puede ejecutar desde el CLI y devuelve JSON |
| RF-02 | El CLI y la API HTTP comparten el mismo núcleo: ninguna regla de negocio vive en una sola de las dos superficies |
| RF-03 | El producto se instala en el proyecto en una sola carpeta, `.task-ai/`, que guarda todo lo suyo |
| RF-04 | El producto localiza el proyecto por su carpeta: desde cualquier punto de su interior encuentra `.task-ai/` |
| RF-05 | La base guarda las tareas, sus dependencias, sus comentarios y el catálogo de etiquetas con color y descripción |
| RF-06 | La carpeta tiene un archivo de settings propio, y el producto lo escribe al instalar con valores por omisión |
| RF-07 | Al cambiar el prefijo, el producto advierte de que las referencias citadas fuera del producto dejan de resolver |
| RF-08 | La carpeta incluye instrucciones para personas y agentes, y el texto que hay que añadir fuera para apuntar a ellas |
| RF-09 | Los errores de la API se comunican con el código de estado que corresponde, no dentro de una respuesta correcta |
| RF-10 | El CLI y el servidor son el mismo programa: un solo binario, que levanta el servidor de la interfaz con un comando |

### Reglas de negocio

| ID | Regla |
|---|---|
| RN-01 | Cada tarea tiene una identidad interna estable, distinta de su referencia visible |
| RN-02 | El prefijo de las referencias visibles es un ajuste del proyecto |
| RN-03 | La referencia visible se forma con el prefijo del proyecto y el número interno de la tarea, y ese número es la identidad que aparece en la salida JSON |
| RN-04 | El estado bloqueado se calcula desde las dependencias y no se almacena |
| RN-05 | Una tarea con subtareas abiertas no se puede cerrar |
| RN-06 | Cerrar una subtarea no cierra su tarea padre |
| RN-07 | La marca de épica la pone quien crea la tarea; el producto no la añade sola |
| RN-08 | La prioridad tiene un rango válido, y el valor que se salga de él se rechaza |

### No funcionales

| ID | Requisito |
|---|---|
| RNF-01 | El acceso a datos se escribe directamente sobre SQLite: sin ORM ni capas de indirección sin justificación |
| RNF-02 | Los datos y el servicio viven en la máquina de la persona: no hay servicio remoto |
| RNF-03 | Varias sesiones o agentes pueden escribir sobre la misma base: la base ordena las escrituras y la interfaz recarga cuando los datos cambian desde fuera |
| RNF-04 | La ayuda del CLI y los mensajes de error de las dos superficies se emiten en inglés; la documentación, los comentarios y el texto de la interfaz web, en español |

### Ajustes del proyecto

Los ajustes que reconoce el producto, sus tipos y sus valores por omisión están en
[`docs/contracts.md`](contracts.md), que es su dueño: este documento no los repite.

## 5. Criterios de aceptación

| ID | Criterio |
|---|---|
| CA-01 | Un agente completa el ciclo de trabajo de una tarea usando solo el CLI y su salida JSON |
| CA-02 | La misma operación pedida por el CLI y por la interfaz deja los datos igual |
| CA-03 | El archivo de settings del proyecto define el prefijo de sus referencias visibles |
| CA-04 | Instalar el producto en un proyecto nuevo deja dentro solo su carpeta y no toca ningún otro archivo |
| CA-05 | Un proyecto recién instalado funciona sin editar el settings: todos sus ajustes tienen valor por omisión |
| CA-06 | Dos escrituras a la vez no se pierden: la segunda entra sobre el resultado de la primera |
| CA-07 | El catálogo de etiquetas conserva color y descripción, y avisa del uso de una etiqueta antes de borrarla |
| CA-08 | La vista de épicas muestra el progreso de sus subtareas |
| CA-09 | La interfaz conserva las preferencias de la persona y avisa cuando los datos cambian desde fuera |
| CA-10 | La interfaz de partida se reutiliza, con los ajustes que exijan el núcleo nuevo y las capacidades nuevas, sin reescribirla |
| CA-11 | El servidor de la interfaz se arranca desde el propio CLI, sin instalar nada aparte |
