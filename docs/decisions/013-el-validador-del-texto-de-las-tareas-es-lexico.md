---
status: accepted
date: 2026-10-21
decision-makers: "Alfredo González Portero"
---

# ADR-013: El validador del texto de las tareas es léxico

## Contexto

El tablero volvió a escribir lo que ya tiene dueño: hashes de commit y frases que
declaran el estado de una tarea, en títulos, descripciones, notas y comentarios. Había
pasado antes, se limpió a mano, y volvió a pasar —también en la sesión que escribía la
regla—, así que la causa no era el descuido de una sesión.

Ya había una decisión en contra. Escribir el hash en el tablero se midió, el ruido se
consideró aceptable y se resolvió por disciplina, con la limpieza manual como remedio. El
remedio se aplicó dos veces y el problema reapareció: una regla que solo depende de que
quien escribe se acuerde no es una regla del producto.

El estado tiene dueño y no es prosa: la tarea y sus bloqueos lo dicen en el tablero, Git
dice qué se entregó y el remitente `Refs: tk-<n>` ata cada commit a su tarea. Una copia
en prosa envejece sin que nadie la corrija y hace creer que el trabajo está donde no está.

## Decisión

El núcleo comprueba el texto de las tareas antes de escribir, con listas cerradas —un hash
de commit, el nombre de una rama, una referencia a un remoto, una declaración de estado y
una declaración del estado de la entrega—, y rechaza lo que cae en ellas como entrada
inválida: no escribe nada, y el mensaje dice qué campo, qué fragmento y qué escribir en su
lugar. El alfabeto y el mensaje son el apartado 8 de [`docs/contracts.md`](../contracts.md).

Lo entrecomillado no exime. La regla se cita nombrando lo que se rechaza, y los ejemplos
viven en el contrato.

La comprobación vive en el núcleo, junto a las demás reglas de negocio, para que las tres
superficies que escriben —CLI, API HTTP y extensión del agente— la reciban sin repetirla.

## Alternativas descartadas

- **Seguir con disciplina.** Era la decisión tomada y falló: el texto que se había
  limpiado volvió a escribirse, por quien conocía la regla.
- **Un aviso en vez de un rechazo.** Un aviso es lo que la disciplina ya hacía: llega
  después de escribir y depende de que alguien lo lea.
- **Rechazar el vocabulario del estado.** Las mismas palabras describen lo que el
  producto hace —una tarea cerrada, una tarea bloqueada, lo que se comprueba al cerrar—,
  así que una regla ancha rechaza especificaciones legítimas del propio producto.
- **Un modelo de lenguaje dentro del producto.** El producto no habla con la red ni trae
  modelo, y un juicio que cambia de una ejecución a otra no es un contrato.

## Consecuencias

- Lo que se rechaza deja de escribirse: lo aprendido se enuncia como hecho, no como
  estado. Algunas fórmulas cómodas dejan de caber en el tablero.
- Las listas son cerradas y finitas: se leen, se enmiendan y se prueban como el resto del
  contrato, y su enmienda es un cambio visible, no un criterio que se mueva solo. Una
  forma entra cuando demuestra el mismo daño y no aparece en ningún texto legítimo; la
  primera enmienda fue la tolerancia al plural del estado de la entrega, y las formas que
  se midieron como legítimas se quedaron fuera, con su motivo escrito en el contrato.
- Queda fuera lo que no se parece ni a un hash ni a un marcador: un hash escrito solo con
  dígitos, un nombre de rama con un prefijo que también es una carpeta del repositorio y
  una declaración de estado sin marcador, escrita en prosa. Lo que las listas no alcanzan
  lo cubre la revisión, y está en [`docs/known-risks.md`](../known-risks.md).
- El texto ya escrito se limpia después de que la guarda exista, no antes: primero la
  regla, después el código, y con el código se comprueba la limpieza.
- Escribir la base sin pasar por el producto no pasa por esta comprobación.
