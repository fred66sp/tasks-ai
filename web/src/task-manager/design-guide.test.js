import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import test from 'node:test';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');
const iconDir = new URL('../../../web/public/icons/heroicons/24/outline/', import.meta.url);
const iconFiles = () => readdirSync(iconDir)
  .filter((name) => name.endsWith('.svg'))
  .map((name) => name.slice(0, -4))
  .sort();
const docPages = ['landing.html', 'index.html', 'design-system.html', 'demo.html'];
test('la portada explica el trabajo agéntico y los sistemas de documentación', () => {
  const page = read('../../../docs/web/landing.html');
  assert.match(page, /desarrollo agéntico/i);
  assert.match(page, /agente[\s\S]*CLI[\s\S]*interfaz web/i);
  assert.match(page, /SDD[\s\S]*ODD/);
  assert.match(page, /<svg[^>]*aria-hidden="true"/);
});
test('los tres heroes comparten la cuadrícula central y los colores laterales', () => {
  const chrome = read('../../../docs/web/assets/css/chrome.css');
  assert.match(chrome, /\.hero > \.hero-mesh\s*\{[^}]*linear-gradient\(to right[\s\S]*linear-gradient\(to bottom[\s\S]*56px 56px[\s\S]*mask-image: radial-gradient/);
  for (const page of docPages) {
    assert.match(read(`../../../docs/web/${page}`), /<div class="hero-mesh" aria-hidden="true"><\/div>/, page);
  }
  assert.match(chrome, /--hero-glow:\s*radial-gradient\(55% 55% at 20% 30%[^;]*var\(--accent-3\)/);
  assert.match(chrome, /\.hero::before,\s*\.shell::before\s*\{[^}]*height:\s*620px;[^}]*background:\s*var\(--hero-glow\)/);
  assert.doesNotMatch(read('../../../docs/web/index.html'), /animation:\s*hero-drift/);
});
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
  assert.doesNotMatch(source.replace(/<div class="flow">[\s\S]*?<!-- flow:end -->/g, ''), /<svg/i, 'la documentación vuelve a dibujar un icono propio');
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

test('la guía declara el tamaño real del símbolo de marca', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.match(page, /El símbolo ocupa un cuadro de 42 px en la cabecera del tablero y 34 px en la documentación/,
    'la frase conserva el tamaño base en vez del de la cabecera del tablero');
});

test('la guía presenta el visor de escritorio como una ficha descargable', () => {
  const page = read('../../../docs/web/index.html');
  const keys = [
    'interfaz.23', 'interfaz.24', 'interfaz.25', 'interfaz.26',
    'interfaz.27', 'interfaz.28', 'interfaz.29', 'interfaz.30'
  ];

  assert.match(page, /class="note viewer"/, 'el visor no tiene una ficha propia');
  for (const key of keys) {
    assert.match(page, new RegExp(`data-i18n="${key}"`), `falta el nodo ${key}`);
  }
  assert.match(page, /<a class="btn btn-primary" href="downloads\/tasks-viewer-windows-amd64\.zip"/,
    'la descarga no tiene una acción primaria');

  const dictionary = JSON.parse(
    page.match(/<script type="application\/json" id="tasks-ai-en">([\s\S]*?)<\/script>/)[1]
  );
  for (const key of keys) {
    assert.ok(dictionary[key], `falta la traducción ${key}`);
  }
});

test('los botones de salida del índice no heredan el estilo de los capítulos', () => {
  const page = read('../../../docs/web/index.html');

  assert.match(page, /\.toc > a \{/, 'el índice no limita el estilo de capítulo a sus enlaces');
  assert.doesNotMatch(page, /\.toc a \{/, 'los botones de salida siguen heredando el enlace de capítulo');
  assert.match(page, /\.toc \.toc-extra \{[^}]*display: grid/,
    'los dos botones de salida no comparten rejilla y separación');
});

test('las visualizaciones de código usan MicroLighter', () => {
  for (const name of docPages) {
    const page = read(`../../../docs/web/${name}`);
    for (const [pre] of page.matchAll(/<pre\b[^>]*>/g)) {
      assert.match(pre, /data-language="[a-z]+"/, `${name}: ${pre} no declara lenguaje`);
    }
  }

  const landing = read('../../../docs/web/landing.html');
  assert.match(landing, /assets\/vendor\/microlighter\/microlighter\.min\.js/,
    'la portada no carga MicroLighter');
  assert.doesNotMatch(landing, /<span class="[ksnbcp]">/,
    'la portada conserva el resaltado manual en vez de MicroLighter');
});

test('el sistema de diseño entra con el mismo gesto y los tokens del tablero', () => {
  const page = read('../../../docs/web/design-system.html');
  const chrome = read('../../../docs/web/assets/css/chrome.css');

  assert.match(chrome, /@keyframes rise/, 'falta la animación de entrada compartida');
  assert.match(page, /animation: rise/, 'los capítulos no usan la animación compartida');
  assert.match(page, /prefers-reduced-motion: no-preference/, 'la entrada no respeta el movimiento reducido');
  assert.match(page, /prefers-reduced-motion: reduce/, 'el guion no respeta el movimiento reducido');
  assert.match(page, /\.chapter-head\.is-ready/, 'los capítulos no se revelan al asomarse');
  assert.match(page, /background: var\(--tk-color-canvas\)/, 'la muestra no usa el token del lienzo');
  assert.match(page, /color: var\(--tk-color-text-muted\)/, 'la muestra no usa el token del texto atenuado');
});

test('el reveal muestra lo que el navegador dejó sobre el pliegue al recargar o volver', () => {
  for (const page of ['index.html', 'design-system.html']) {
    const source = read(`../../../docs/web/${page}`);

    assert.match(source, /getBoundingClientRect\(\)\.top < window\.innerHeight/,
      `${page}: el arranque no revela los bloques restaurados por encima del pliegue`);
    assert.match(source, /addEventListener\("pageshow"/,
      `${page}: la vuelta desde otra página no vuelve a revelar lo restaurado`);
  }
});

test('el hero del catálogo comparte el acento y el gesto de las otras superficies', () => {
  const page = read('../../../docs/web/design-system.html');
  const chrome = read('../../../docs/web/assets/css/chrome.css');

  assert.match(page, /<h1>[\s\S]*?<span class="grad">/, 'el título del hero no lleva el acento de la guía');
  assert.match(page, /\.shell::before \{[^}]*grid-column:\s*1 \/ -1[^}]*grid-row:\s*1[^}]*inset:\s*-250px -10% auto/, 'los degradados deben alinearse con todo el ancho de la portada');
  assert.match(page, /\.hero > \.hero-mesh \{[^}]*left:\s*calc\(-272px - var\(--gutter\)\)[^}]*width:\s*100vw/, 'la cuadrícula debe alinearse con el ancho de la ventana');
  assert.match(page, /@media \(max-width: 1020px\) \{[^}]*\}[^}]*\.shell::before \{[^}]*grid-row:\s*2/, 'en móvil deben comenzar bajo el rail');
  assert.doesNotMatch(page, /\.content \{[^}]*radial-gradient/, 'main no debe tener degradados');
  assert.match(page, /\.hero::after \{[^}]*linear-gradient/, 'el hero no cierra con la regla de marca');
  assert.match(page, /\.chapter-num \{[^}]*background: var\(--accent-soft\)/,
    'los números de capítulo no usan el contenedor de color de la guía');
  assert.match(chrome, /\.grad \{[^}]*var\(--accent-2\)/, 'el degradado de acento no vive en la mobilia común');
});

test('los bloques de código comparten marco y cabecera en la guía y la portada', () => {
  const chrome = read('../../../docs/web/assets/css/chrome.css');
  const docs = read('../../../docs/web/assets/js/docs.js');

  assert.match(chrome, /\.code::before \{[^}]*attr\(data-language\)/,
    'la cabecera del bloque no declara el lenguaje');
  assert.match(chrome, /\.code \{[^}]*border-radius: var\(--radius\)/, 'el marco del código no es común');
  assert.match(docs, /block\.dataset\.language = .*data-language/,
    'la cabecera no toma el lenguaje del bloque');
  for (const page of ['index.html', 'landing.html']) {
    assert.doesNotMatch(read(`../../../docs/web/${page}`), /\.code pre \{/,
      `${page}: el bloque de código conserva su propio marco`);
  }
});

test('el degradado de texto mantiene contraste y sobrevive a la impresión sin fondos', () => {
  const chrome = read('../../../docs/web/assets/css/chrome.css');
  const grad = chrome.match(/\.grad \{([^}]*)\}/);

  assert.ok(grad, 'falta la regla .grad');
  assert.doesNotMatch(grad[1], /--accent-3/,
    'el cian de cierre no alcanza 3:1 como texto sobre el papel claro');
  assert.match(chrome, /@media print[\s\S]*?\.grad \{[^}]*(?:color|text-fill-color): #0f172a/,
    'el título queda transparente al imprimir sin fondos');
});

test('el antetítulo del visor conserva su margen frente a la nota', () => {
  const page = read('../../../docs/web/index.html');

  assert.match(page, /\.viewer \.viewer-eyebrow \{[^}]*margin: 0 0 6px/,
    '.note p anula el margen del antetítulo');
});

test('el código de la ficha del visor recibe el chip de la guía', () => {
  const page = read('../../../docs/web/index.html');
  const rule = page.match(/([^{}]*\.viewer-meta code[^{}]*)\{([^}]*)\}/);

  assert.ok(rule, 'falta el estilo del código dentro de viewer-meta');
  assert.match(rule[2], /background: var\(--bg-soft\)/, 'el código no lleva el fondo del chip');
  assert.match(rule[2], /border: 1px solid var\(--border\)/, 'el código no lleva el borde del chip');
});

test('los botones del índice no se subrayan al pasar el ratón', () => {
  const chrome = read('../../../docs/web/assets/css/chrome.css');
  const hover = chrome.match(/([^{}]*\.toc \.icon-btn:hover[^{}]*)\{([^}]*)\}/);

  assert.ok(hover, 'los botones del índice no tienen regla de hover');
  assert.match(hover[2], /text-decoration: none/, 'el subrayado global de enlace alcanza a los botones');
});

test('la portada documenta la limitación de file:// sin depender del resaltado', () => {
  const notice = read('../../../docs/web/assets/vendor/microlighter/NOTICE.txt');
  const landing = read('../../../docs/web/landing.html');

  assert.match(notice, /file:\/\//,
    'la limitación de MicroLighter bajo file:// no está documentada');
  assert.match(landing, /<pre data-language="json"[^>]*><code>\{/,
    'el bloque conserva texto plano legible sin resaltado');
});
