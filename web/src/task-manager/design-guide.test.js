import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import test from 'node:test';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');

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

  assert.match(page, /href="\.\.\/\.\.\/web\/public\/task-manager\/app-vue\.css"/,
    'la guía no carga el CSS real del tablero');
  assert.doesNotMatch(page, /\bds-[a-z]/, 'la guía conserva una capa de componentes propia');
  for (const specimen of ['tk-button', 'tk-input', 'tk-select', 'tk-badge', 'tk-tab', 'task-row', 'task-tree', 'tk-panel', 'tk-modal', 'tk-dropdown', 'tag-manager-drawer', 'tag-manager-color']) {
    assert.match(page, new RegExp(`class="[^"]*\\b${specimen}\\b`), `falta la muestra ${specimen}`);
  }
  assert.match(page, /notifications\/toast-anchor\.js/, 'el aviso y la confirmación no usan el módulo del producto');
  assert.match(page, /id="demo-toast"/);
  assert.match(page, /id="demo-confirm"/);
  assert.doesNotMatch(page, /<iframe/i, 'un iframe no es un componente del sistema de diseño');
});

test('la guía cataloga el conjunto de iconos y cada archivo existe', () => {
  const page = read('../../../docs/web/design-system.html');
  const dir = new URL('../../../web/public/icons/heroicons/24/outline/', import.meta.url);

  const refs = [...page.matchAll(/--icon:\s*url\('\.\.\/\.\.\/web\/public\/icons\/heroicons\/24\/outline\/([a-z0-9-]+)\.svg'\)/g)]
    .map((match) => match[1]);
  const files = readdirSync(dir)
    .filter((name) => name.endsWith('.svg'))
    .map((name) => name.slice(0, -4))
    .sort();

  assert.ok(refs.length > 0, 'la guía no muestra el conjunto de iconos');
  assert.deepEqual([...refs].sort(), files, 'el catálogo no cubre exactamente los iconos del conjunto');
  assert.equal(new Set(refs).size, refs.length, 'hay iconos repetidos en el catálogo');
  for (const name of refs) {
    assert.ok(existsSync(new URL(`${name}.svg`, dir)), `falta ${name}.svg en el conjunto`);
  }
  assert.ok(existsSync(new URL('../../../web/public/icons/heroicons/LICENSE', import.meta.url)),
    'falta la licencia del conjunto');
  assert.match(page, /web\/public\/icons\/heroicons\/LICENSE/, 'la guía no declara la licencia');
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
  const landing = read('../../../docs/web/landing.html');
  const guide = read('../../../docs/web/index.html');
  const page = read('../../../docs/web/design-system.html');

  for (const doc of [page, landing, guide]) {
    assert.match(doc, /padding: 12px clamp\(16px, 4vw, 40px\)/, 'la cabecera no comparte el margen');
    assert.match(doc, /background: color-mix\(in srgb, var\(--(?:page|bg)\) 82%, transparent\)/, 'la cabecera no comparte el velo');
  }
  for (const doc of [page, landing]) {
    assert.match(doc, /\.site-nav a \{[\s\S]*?line-height: 1\.2/, 'los enlaces de sección cambian el alto de la barra');
  }
});

test('la guía tiene vuelta arriba como la guía de uso', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /<a class="to-top" id="to-top" href="#top" aria-label="Volver arriba">/);
  assert.match(page, /<div id="top" class="page-top" aria-hidden="true"><\/div>/,
    'falta el ancla real del principio del documento');
  assert.ok(page.indexOf('<div id="top"') < page.indexOf('<header class="topbar">'),
    'el ancla superior debe ir antes de la cabecera sticky');
  assert.doesNotMatch(page, /<header class="topbar" id="top">/,
    'una cabecera sticky no sirve como destino de #top');
  assert.match(page, /toTop\.classList\.toggle\('is-visible', window\.scrollY > 480\)/);
});

test('cada sección lleva el resplandor del 09', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /section \{[^}]*radial-gradient\(circle at 88% 0, rgb\(98 167 255 \/ \.08\), transparent 34%\)/);
  assert.match(page, /<section id="espacio-trabajo" aria-labelledby="espacio-trabajo-t">/);
  assert.doesNotMatch(page, /board-band/);
});

test('el logo cambia con el tema', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /class="brand-icon-light"[^>]*tasks-ai-light-64\.png/);
  assert.match(page, /class="brand-icon-dark"[^>]*tasks-ai-dark-64\.png/);
  assert.match(page, /:root\[data-theme="dark"\] \.brand-mark \.brand-icon-light \{ display: none; \}/);
  assert.match(page, /:root\[data-theme="dark"\] \.brand-mark \.brand-icon-dark \{ display: block; \}/);
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
