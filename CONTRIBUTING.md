# Flujo de trabajo Git

Política de este repositorio para ramas, commits, verificación e integración. Prevalece
sobre cualquier modelo genérico. La ceremonia de cada operación —clasificarla, preparar el
stage, presentar el informe y esperar confirmación— la gobierna la skill
`git-flow-management`, instalada en el arnés.

## Modelo

- `main` es la única rama de larga vida; contiene lo que está listo para usarse.
- No hay `develop`, `release/*` ni `hotfix/*`: introducirlos exige cambiar esta política.
- El commit se hace antes de cerrar la tarea en el tablero. El orden completo del cierre
  está en [`docs/development.md`](docs/development.md).

## Ramas

Se crean desde `main` y viven lo mínimo: nombres en minúsculas, palabras separadas con
guiones.

| Patrón | Uso |
|---|---|
| `feature/<slug>` | Funcionalidad nueva |
| `fix/<slug>` | Corrección de un defecto |
| `docs/<slug>` | Documentación |
| `refactor/<slug>` | Reorganización sin cambio de comportamiento |
| `chore/<slug>` | Mantenimiento, herramientas, configuración |

Una tarea pequeña puede ir directamente a `main`. Es pequeña cuando el diff se entiende
leyéndolo, se deshace revirtiendo el commit y no toca datos almacenados, contratos
públicos, seguridad, concurrencia ni código ejecutable en varios módulos, ni mezcla
cambios ajenos. Ante la duda, rama.

Crear una rama local no integra ni publica nada.

## Commits

- Conventional Commits, en inglés: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
  `style`, `perf`, `ci`, `build`, `revert`.
- Descripción breve, en imperativo y sin punto final.
- Un commit representa una unidad coherente. Varios solo cuando separan cambios
  independientes o hacen la revisión más clara.
- El cuerpo nombra la tarea que sostiene el commit con `Refs: tk-<n>`, una por tarea. Es
  la única atadura entre tarea y commit; el tablero no guarda hashes:

```bash
git log --format='%h %s' --grep='Refs: tk-1'
```

- Ningún commit con cabos sueltos: lo que se descubre al prepararlo se arregla o se
  pregunta antes, no en un commit de corrección inmediata.
- Sin firmas de IA ni líneas `Co-Authored-By` salvo petición expresa.
- Ningún commit se crea sin confirmación explícita: cubre lo presentado, y un cambio de
  alcance pide otra. El orden del cierre y las comprobaciones están en
  [`docs/development.md`](docs/development.md).

## Verificación

La verificación es proporcional: revisar el diff y una comprobación localizada en un
cambio pequeño; la validación principal del proyecto en uno normal; y las comprobaciones
de los puntos críticos en uno complejo. Toda verificación no ejecutada se declara.

## Integración

- Se integra en `main` con merge no fast-forward: `git merge --no-ff <rama>`.
- Sin `squash` ni `rebase` sobre ramas ya publicadas.
- Antes: la tarea terminada, las pruebas correctas, ningún cambio ajeno que se pierda y la
  rama correcta.
- Después: se comprueba estado, rama y que el commit está en `main`.
- Una rama con trabajo no integrado no se borra sin autorización.

## CHANGELOG

- Se anota lo que tiene valor para quien usa el proyecto, al publicar una versión, bajo
  `## [Sin publicar]`, con las secciones `Añadido`, `Modificado` y `Corregido`.
- No se actualiza por cada commit ni por cambios internos.

## Versiones y etiquetas

- Versionado semántico cuando haya versión publicada o se pida una: `PATCH` para una
  corrección compatible, `MINOR` para funcionalidad compatible y `MAJOR` para un cambio
  incompatible.
- La etiqueta es anotada, con prefijo `v`, y marca el commit integrado.
- Crear una etiqueta, preparar una release o publicar exige autorización explícita.

## Operaciones que exigen autorización

- Fusionar en `main`.
- `push`, `fetch` para integrar y cualquier otra operación remota.
- Crear una etiqueta o publicar una versión.
- `reset`, `clean`, `rebase`, `filter-repo` u otra operación que elimine o reescriba
  trabajo.
- Sobrescribir, descartar u ocultar cambios de la persona.
- Borrar una rama con trabajo no integrado.

Una autorización cubre lo que nombra y no se hereda a otra operación.

## Principio

Git aporta seguridad y trazabilidad sin convertirse en una carga. El orden de prioridad:
no perder trabajo, entregar algo verificable, mantener un historial comprensible, pedir
autorización para lo persistente y no añadir ramas, commits ni comprobaciones que no
aporten.
