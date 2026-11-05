import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import test from 'node:test';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');
const iconDir = new URL('../../../web/public/icons/heroicons/24/outline/', import.meta.url);
const iconFiles = () => readdirSync(iconDir)
  .filter((name) => name.endsWith('.svg'))
  .map((name) => name.slice(0, -4))
  .sort();
const docPages = ['landing.html', 'index.html', 'design-system.html'];
const docSource = () => docPages
  .map((name) => read(`../../../docs/web/${name}`))
  .join('\n');
const productSource = () => readdirSync(new URL('../../../web/src/', import.meta.url), { recursive: true })
  .filter((name) => (name.endsWith('.js') || name.endsWith('.css')) && !name.endsWith('.test.js'))
  .map((name) => read(`../../../web/src/${name}`))
  .join('\n');

test('la guía visual se puede abrir desde la portada y la guía de uso', () => {
  const page = read('../../../docs/web/design-system.html');
  const landing = read('../../../docs/web/landing.html');
  const guide = read('../../../docs/web/index.html');

  assert.match(landing, /href="design-system\.html"/, 'falta el enlace desde la portada');
  assert.match(guide, /href="design-system\.html"/, 'falta el enlace desde la guía de uso');
  for (const id of ['fundamentos', 'marca', 'color', 'tipografia', 'espacio', 'iconografia', 'acciones', 'insignias', 'etiquetas', 'espacio-trabajo', 'modales', 'estados', 'accesibilidad', 'evolucion']) {
    assert.match(page, new RegExp(`id="${id}"`), `falta la sección ${id}`);
    assert.match(page, new RegExp(`href="#${id}"`), `falta el enlace a ${id}`);
  }
  assert.doesNotMatch(page, /Tacks UI System|\.vue\b|Exportar tokens/, 'la referencia histórica no es el contrato');
  assert.doesNotMatch(read('../../../web/src/task-manager/components/tag-manager-drawer.js'), /Tacks/,
    'la interfaz sigue nombrando al producto anterior');
});

test('la guía consume el CSS del tablero y no mantiene una copia de los componentes', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /href="assets\/vendor\/app-vue\.css"/,
    'la guía no carga la copia vendor del CSS real del tablero');
  assert.doesNotMatch(page, /\bds-[a-z]/, 'la guía conserva una capa de componentes propia');
  for (const specimen of ['tk-button', 'tk-input', 'tk-select', 'tk-badge', 'tk-tab', 'task-row', 'task-tree', 'tk-panel', 'tk-modal', 'tk-dropdown', 'tag-manager-drawer', 'tag-manager-color']) {
    assert.match(page, new RegExp(`class="[^"]*\\b${specimen}\\b`), `falta la muestra ${specimen}`);
  }
  assert.match(page, /assets\/vendor\/toast-anchor\/toast-anchor-wrapper\.js/, 'el aviso y la confirmación no usan el módulo del producto');
  assert.match(page, /id="demo-toast"/);
  assert.match(page, /id="demo-confirm"/);
  assert.doesNotMatch(page, /<iframe/i, 'un iframe no es un componente del sistema de diseño');
});

test('la guía cataloga la selección de iconos y cada archivo existe', () => {
  const page = read('../../../docs/web/design-system.html');
  const files = iconFiles();

  const refs = [...page.matchAll(/<span class="icon-item__name">([a-z0-9-]+)<\/span>/g)]
    .map((match) => match[1]);

  assert.ok(refs.length > 0, 'la guía no muestra el conjunto de iconos');
  assert.deepEqual([...refs].sort(), files, 'el catálogo no cubre exactamente los iconos del conjunto');
  assert.equal(new Set(refs).size, refs.length, 'hay iconos repetidos en el catálogo');
  for (const name of refs) {
    assert.match(page, new RegExp(`tk-icon--${name}" aria-hidden="true"></span>\\s*<span class="icon-item__name">${name}</span>`),
      `${name} no se dibuja con la clase del conjunto`);
  }
  assert.ok(existsSync(new URL('../../../web/public/icons/heroicons/LICENSE', import.meta.url)),
    'falta la licencia del conjunto');
  assert.match(page, /web\/public\/icons\/heroicons\/LICENSE/, 'la guía no declara la licencia');
});

test('el producto dibuja solo iconos del conjunto', () => {
  const files = iconFiles();
  const source = productSource();
  const used = [...source.matchAll(/<tk-icon[^>]*\bname="([a-z0-9-]+)"/g)]
    .map((match) => match[1]);

  assert.ok(used.length > 0, 'la interfaz no usa el componente de iconos');
  for (const name of new Set(used)) {
    assert.ok(files.includes(name), `la interfaz usa ${name}, que no está en el conjunto`);
  }
  assert.doesNotMatch(source, /<svg/i, 'la interfaz vuelve a dibujar un trazo propio');
});

test('la documentación dibuja solo iconos del conjunto', () => {
  const files = iconFiles();
  const source = docSource();
  const used = [...source.matchAll(/tk-icon--([a-z0-9-]+)/g)]
    .map((match) => match[1])
    .filter((name) => name !== 'mask');

  assert.ok(used.length > 0, 'la documentación no usa el conjunto de iconos');
  for (const name of new Set(used)) {
    assert.ok(files.includes(name), `la documentación usa ${name}, que no está en el conjunto`);
  }
  assert.doesNotMatch(source, /<svg/i, 'la documentación vuelve a dibujar un trazo propio');
});

test('el producto o la documentación usan cada archivo del conjunto', () => {
  const files = iconFiles();
  const product = [...productSource().matchAll(/<tk-icon[^>]*\bname="([a-z0-9-]+)"/g)]
    .map((match) => match[1]);
  const docs = [...docSource().matchAll(/tk-icon--([a-z0-9-]+)/g)]
    .map((match) => match[1])
    .filter((name) => name !== 'mask');

  assert.deepEqual([...new Set([...product, ...docs])].sort(), files,
    'hay un archivo en el conjunto que ninguna superficie usa');
});

test('la hoja de iconos asigna una clase a cada archivo', () => {
  const files = iconFiles();
  const css = read('../../../web/public/icons/icons.css');
  const mapped = [...css.matchAll(
    /\.tk-icon--([a-z0-9-]+)\s*\{\s*--icon:\s*url\('heroicons\/24\/outline\/([a-z0-9-]+)\.svg'\)/g
  )].map((match) => {
    assert.equal(match[1], match[2], `${match[1]} apunta a otro archivo`);
    return match[1];
  });

  assert.deepEqual(mapped.sort(), files, 'la hoja de iconos no cubre exactamente el conjunto');
});

test('el producto y la documentación cargan la hoja de iconos', () => {
  for (const name of docPages) {
    assert.match(read(`../../../docs/web/${name}`), /href="assets\/vendor\/icons\/icons\.css"/,
      `${name} no carga la hoja de iconos vendorizada`);
  }
  assert.match(read('../../../web/public/task-manager/app-vue.html'), /href="\.\.\/icons\/icons\.css"/,
    'el tablero no carga la hoja compartida de iconos');
});

test('las muestras de la guía usan solo clases del conjunto', () => {
  const page = read('../../../docs/web/design-system.html');
  const files = iconFiles();
  const used = [...page.matchAll(/tk-icon--([a-z0-9-]+)/g)]
    .map((match) => match[1])
    .filter((name) => name !== 'mask');

  for (const name of new Set(used)) {
    assert.ok(files.includes(name), `la guía dibuja ${name}, que no está en el conjunto`);
  }
});

test('la guía está bien anidada y no rompe la columna de contenido', () => {
  const page = read('../../../docs/web/design-system.html');
  const count = (pattern) => (page.match(pattern) || []).length;

  for (const tag of ['div', 'section', 'main', 'nav', 'header', 'footer', 'article', 'aside']) {
    assert.equal(
      count(new RegExp(`<${tag}\\b`, 'g')),
      count(new RegExp(`</${tag}>`, 'g')),
      `${tag}: etiquetas desbalanceadas`
    );
  }
  assert.ok(page.indexOf('</main>') > page.indexOf('id="evolucion"'), 'el cierre de main corta la página');
});

test('la anatomía de la fila vive en el CSS del tablero', () => {  const page = read('../../../docs/web/design-system.html');
  const components = read('../../../web/src/task-manager/styles/components.css');

  assert.match(components, /\.task-row\s*\{[\s\S]*?grid-template-columns: 34px 34px minmax\(170px, 1fr\) 120px 130px 38px/,
    'la rejilla de la fila cambió');
  assert.doesNotMatch(page, /grid-template-columns: 34px/, 'la guía vuelve a dibujar la fila');
});

test('el icono compacto y el título del inspector tienen su regla', () => {
  const components = read('../../../web/src/task-manager/styles/components.css');

  assert.match(components, /\.tk-icon-button--compact\s*\{[^}]*width:\s*32px/, 'el icono compacto no tiene tamaño propio');
  assert.match(components, /\.task-inspector__title\s*\{[^}]*font-weight:\s*750/, 'el título del inspector no aplica el peso del contrato');
});

test('el desplegable de la tarea gira una sola vez', () => {
  const components = read('../../../web/src/task-manager/styles/components.css');

  assert.match(components, /\.tree-expander\[aria-expanded="true"\] \.tk-icon\s*\{[^}]*rotate\(90deg\)/,
    'el icono del desplegable no gira al abrirse');
  assert.doesNotMatch(components, /\.tree-expander\[aria-expanded="true"\]\s*\{[^}]*transform/,
    'el desplegable gira el botón y el icono: el chevron queda al revés');
});

test('la guía asigna a cada etiqueta el color de su catálogo', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /--tag-color: #f2bf59;?"><\/span>improvement/, 'improvement no lleva el amarillo del catálogo');
  assert.match(page, /--tag-color: #b58bff;?"><\/span>ODD/, 'ODD no lleva el violeta del catálogo');
  assert.match(page, /tag-manager-dot" style="--tag-color: #62a7ff;"[\s\S]*?tag-manager-dot" style="--tag-color: #f2bf59;"[\s\S]*?tag-manager-dot" style="--tag-color: #b58bff;"/,
    'el cajón no sigue el orden de colores del catálogo');
  assert.doesNotMatch(page, /--tag-color: #b58bff;?"><\/span>improvement/, 'improvement conserva el color de ODD');
  assert.doesNotMatch(page, /--tag-color: #58d392;?"><\/span>ODD/, 'ODD conserva el color de otra etiqueta');
});

test('la cabecera comparte geometría con la portada y la guía de uso', () => {
  const chrome = read('../../../docs/web/assets/css/chrome.css');

  assert.match(chrome, /padding: 12px clamp\(16px, 4vw, 40px\)/, 'la cabecera no comparte el margen');
  assert.match(chrome, /background: color-mix\(in srgb, var\(--bg\) 82%, transparent\)/, 'la cabecera no comparte el velo');
  assert.match(chrome, /\.site-nav a \{[\s\S]*?line-height: 1\.2/, 'los enlaces de sección cambian el alto de la barra');
  for (const name of docPages) {
    assert.match(read(`../../../docs/web/${name}`), /href="assets\/css\/chrome\.css"/,
      `${name} no carga la mobilia común (ADR-018)`);
  }
});

test('la guía tiene vuelta arriba como la guía de uso', () => {
  const page = read('../../../docs/web/design-system.html');
  const docs = read('../../../docs/web/assets/js/docs.js');

  assert.match(page, /<a class="to-top" id="to-top" href="#top" aria-label="Volver arriba">/);
  assert.match(page, /<div id="top" class="page-top" aria-hidden="true"><\/div>/,
    'falta el ancla real del principio del documento');
  assert.ok(page.indexOf('<div id="top"') < page.indexOf('<header class="topbar" id="docs-topbar">'),
    'el ancla superior debe ir antes de la cabecera sticky');
  assert.doesNotMatch(page, /<header class="topbar" id="top">/,
    'una cabecera sticky no sirve como destino de #top');
  assert.match(docs, /toTop\.classList\.toggle\("is-visible", window\.scrollY > 480\)/,
    'el volver arriba ya no vive en el guion común (ADR-018)');
});

test('cada sección lleva el resplandor del 09', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /section \{[^}]*radial-gradient\(circle at 88% 0, rgb\(98 167 255 \/ \.08\), transparent 34%\)/);
  assert.match(page, /<section id="espacio-trabajo" aria-labelledby="espacio-trabajo-t">/);
  assert.doesNotMatch(page, /board-band/);
});

test('el logo cambia con el tema', () => {
  const docs = read('../../../docs/web/assets/js/docs.js');
  const chrome = read('../../../docs/web/assets/css/chrome.css');

  assert.match(docs, /class="brand-icon-light"[^>]*assets\/img\/app-icons\/tasks-ai-light-64\.png/);
  assert.match(docs, /class="brand-icon-dark"[^>]*assets\/img\/app-icons\/tasks-ai-dark-64\.png/);
  assert.match(chrome, /:root\[data-theme="dark"\] \.brand-mark \.brand-icon-light \{ display: none; \}/);
  assert.match(chrome, /:root\[data-theme="dark"\] \.brand-mark \.brand-icon-dark \{ display: block; \}/);
});

test('la guía declara la paleta de etiquetas y avisos de DESIGN.md', () => {
  const page = read('../../../docs/web/design-system.html');
  const design = read('../../../DESIGN.md');
  const data = read('../../../web/src/task-manager/state/workspace-data.js');
  const anchor = read('../../../web/src/task-manager/notifications/toast-anchor.js');

  for (const hex of ['#62a7ff', '#b58bff', '#58d392', '#f2bf59', '#ff9a55', '#ff7373']) {
    assert.match(data, new RegExp(hex), `el catálogo de etiquetas ya no usa ${hex}`);
    assert.match(design, new RegExp(hex), `DESIGN.md no declara ${hex}`);
    assert.match(page, new RegExp(hex), `la guía no muestra ${hex}`);
  }
  for (const hex of ['#121c27', '#182431', '#344657', '#bac4ce']) {
    assert.match(anchor, new RegExp(hex), `el aviso ya no usa ${hex}`);
    assert.match(design, new RegExp(hex), `DESIGN.md no declara ${hex}`);
    assert.doesNotMatch(page, new RegExp(hex), `la guía duplica el aviso con ${hex}`);
  }
});
