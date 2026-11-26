# SKILLS-1 Las skills del proyecto

Este proyecto lleva dentro las skills que necesita, copiadas de su origen: el taller
`workshop-golang`, para las de Go, y el repositorio propio `SKILLS-AGP`, para las de ADR,
PRD y TypeScript. Viven en `skills/` y se leen por su ruta relativa al proyecto, así que
ningún trabajo depende de que el origen esté al lado. Git y la documentación no viven aquí:
su procedimiento lo ponen las skills que instala el arnés.

Cada skill es un directorio con su `SKILL.md`, y algunas traen `references/`, `assets/`,
`scripts/` o el agente de su interfaz.

## SKILLS-2 Cuáles hay y de dónde vienen

| Skill | Origen | Licencia | Modificada |
|---|---|---|---|
| `golang-how-to` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | Sí — `references/by-category.md` recorta el catálogo a las skills de Go |
| `golang-project-layout` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `golang-cli` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | Sí — sus doce ejemplos de `assets/examples/` llevan `//go:build ignore` |
| `golang-database` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `golang-testing` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | Sí — `SKILL.md` añade `routing_terms` |
| `golang-error-handling` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `golang-code-style` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `golang-lint` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `golang-naming` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | [MIT](LICENSES/mit-samber-cc-skills-golang.txt) | No — verbatim de `bac46b0` |
| `adr-software` | Repositorio propio `SKILLS-AGP` | [MIT](LICENSES/mit-skills-agp.txt) | No — verbatim de `da9693e` |
| `prd-cartographer` | Repositorio propio `SKILLS-AGP` | [MIT](LICENSES/mit-skills-agp.txt) | No — verbatim de `3932812` |
| `typescript-idiomatic-style` | Repositorio propio `SKILLS-AGP` | [MIT](LICENSES/mit-skills-agp.txt) | No — verbatim de `3932812` |
| `typescript-testing-and-verification` | Repositorio propio `SKILLS-AGP` | [MIT](LICENSES/mit-skills-agp.txt) | No — verbatim de `e6e2729` |

La revisión citada es la de cada origen que coincide byte a byte con la copia, y las tres
modificadas vienen así del taller `workshop-golang`, que es de donde se copiaron: el origen
de samber ha seguido avanzando desde entonces. Las cuatro de `SKILLS-AGP` salen de
`src/<skill>/` —en `.skills/` solo hay enrutadores de un solo fichero— y su única
divergencia respecto a esa revisión es la línea `license:` de su frontmatter, que la
declaración MIT de este proyecto añade. El texto completo de las dos licencias MIT, con su
línea de copyright, vive en `skills/LICENSES/`; el repositorio no lleva `LICENSE` propio y
licenciar el producto queda como decisión aparte del PO.

Las trece declaran su licencia en el frontmatter: las de terceros conservan además el autor
y la página de metadatos con los que vinieron, y las de desarrollo propio declaran su
versión. Las skills no se traducen: se usan tal y como están en su origen.

## SKILLS-3 Cómo se usan

El `AGENTS.md` del proyecto enruta a ellas: la tabla de intención dice cuál leer para
cada tarea y con qué combinarlas. Para Go se empieza por `skills/golang-how-to/SKILL.md`,
que es el orquestador y decide el resto.

Git y la documentación se gobiernan con las skills que instala el arnés,
`git-flow-management` y `document-management-software`. No tienen copia en `skills/`.

## SKILLS-4 Cómo se actualizan

Se vuelven a copiar de su origen, sin editar aquí: lo que se corrige en una skill se
corrige en su origen y se trae después. Para rehacer la copia, desde la raíz del taller:

```bash
for s in golang-how-to golang-project-layout golang-cli golang-database \
         golang-testing golang-error-handling golang-code-style golang-lint \
         golang-naming; do
  cp -r "skills/$s" "projects-out/tasks-ai/skills/"
done
```

Y las de `SKILLS-AGP`, desde la raíz de ese repositorio, con la ruta del proyecto a mano:

```bash
for s in adr-software prd-cartographer typescript-idiomatic-style \
         typescript-testing-and-verification; do
  cp -r "src/$s" <proyecto>/skills/
done
```

Al recopiar, la constancia se recalcula comparando cada skill con su origen (`diff -r`
contra un clon de samber o la revisión de `SKILLS-AGP`, y `git hash-object` del `SKILL.md`),
y se actualiza en la tabla la revisión de la que sale la copia y si dejó de coincidir.

Si una tarea pide una skill que no está aquí ni la instala el arnés, se añade a esta
lista, a la tabla del `AGENTS.md` y se copia: el proyecto no lee el taller por la ruta de
fuera.
