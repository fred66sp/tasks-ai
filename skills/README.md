# SKILLS-1 Las skills del proyecto

Este proyecto lleva dentro las skills que necesita, copiadas de su origen: el taller
`workshop-golang`, para las de Go y la de TypeScript de la extensión, y el repositorio
propio `SKILLS-AGP`, para la de ADR. Viven en `skills/` y se leen por su ruta relativa al
proyecto, así que ningún trabajo depende de que el origen esté al lado. Git y la
documentación no viven aquí: su procedimiento lo ponen las skills que instala el arnés.

Cada skill es un directorio con su `SKILL.md`, y algunas traen `references/`.

## SKILLS-2 Cuáles hay y de dónde vienen

| Skill | Origen | Licencia |
|---|---|---|
| `golang-how-to` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-project-layout` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-cli` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-database` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-testing` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-error-handling` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-code-style` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-lint` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `golang-naming` | [`samber/cc-skills-golang`](https://github.com/samber/cc-skills-golang) | MIT |
| `typescript` | [`terminal-skills`](https://github.com/terminal-skills) | Apache-2.0 |
| `prd-writing` | [`terminal-skills`](https://github.com/terminal-skills) | Apache-2.0 |
| `prd-to-plan` | [`terminal-skills`](https://github.com/terminal-skills) | Apache-2.0 |
| `prd-to-issue` | [`terminal-skills`](https://github.com/terminal-skills) | Apache-2.0 |
| `adr-software` | Repositorio propio `SKILLS-AGP` | No declarada todavía |

Las skills de terceros conservan en su frontmatter el autor, la página de metadatos y la
licencia con los que vinieron; las de desarrollo propio declaran su versión. Las skills no
se traducen: se usan tal y como están en su origen.

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
         golang-naming typescript prd-writing prd-to-plan prd-to-issue; do
  cp -r "skills/$s" "projects-out/tasks-ai/skills/"
done
```

Y la de ADR, desde la raíz de `SKILLS-AGP`, con la ruta del proyecto a mano:

```bash
cp -r src/adr-software <proyecto>/skills/
```

Si una tarea pide una skill que no está aquí ni la instala el arnés, se añade a esta
lista, a la tabla del `AGENTS.md` y se copia: el proyecto no lee el taller por la ruta de
fuera.
