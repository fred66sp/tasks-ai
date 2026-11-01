import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');

test('la guía visual se puede abrir desde la portada y la guía de uso', () => {
  const page = read('../../../docs/web/design-system.html');
  const landing = read('../../../docs/web/landing.html');
  const guide = read('../../../docs/web/index.html');

  assert.match(landing, /href="design-system\.html"/, 'falta el enlace desde la portada');
  assert.match(guide, /href="design-system\.html"/, 'falta el enlace desde la guía de uso');
  for (const id of ['fundamentos', 'marca', 'color', 'tipografia', 'espacio', 'acciones', 'insignias', 'etiquetas', 'espacio-trabajo', 'modales', 'estados', 'accesibilidad', 'evolucion']) {
    assert.match(page, new RegExp(`id="${id}"`), `falta la sección ${id}`);
    assert.match(page, new RegExp(`href="#${id}"`), `falta el enlace a ${id}`);
  }
  assert.doesNotMatch(page, /Tacks UI System|\.vue\b|Exportar tokens/, 'la referencia histórica no es el contrato');
});

test('las muestras son componentes propios, no la interfaz incrustada', () => {
  const page = read('../../../docs/web/design-system.html');

  assert.doesNotMatch(page, /<iframe/i, 'un iframe no es un componente del sistema de diseño');
  for (const specimen of ['ds-button', 'ds-input', 'ds-badge', 'ds-tab', 'ds-row', 'ds-inspector', 'ds-panel', 'ds-menu', 'ds-tag', 'ds-drawer', 'ds-modal', 'ds-toast', 'ds-swatch', 'ds-select--status']) {
    assert.match(page, new RegExp(`class="[^"]*\\b${specimen}\\b`), `falta la muestra ${specimen}`);
  }
  assert.match(page, /\.ds-button\s*\{[^}]*align-items: center/, 'el botón de muestra no centra su contenido');
});

test('las muestras conservan los valores del tablero', () => {
  const page = read('../../../docs/web/design-system.html');
  const tokens = read('../../../web/src/task-manager/styles/tokens.css');
  const components = read('../../../web/src/task-manager/styles/components.css');

  for (const [token, value] of [
    ['--tk-color-accent', '#a78bfa'],
    ['--tk-color-surface-selected', '#2a3750'],
    ['--tk-control-height', '42px'],
    ['--tk-tree-row-height', '68px'],
    ['--tk-radius-sm', '6px']
  ]) {
    assert.match(tokens, new RegExp(`${token}:\\s*${value}`), `el token ${token} cambió`);
    assert.match(page, new RegExp(value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `la guía no refleja ${token}`);
  }
  assert.match(components, /\.task-row\s*\{[\s\S]*?grid-template-columns: 34px 34px minmax\(170px, 1fr\) 120px 130px 38px/,
    'la rejilla de la fila cambió');
  assert.match(page, /grid-template-columns: 34px 34px minmax\(170px, 1fr\) 120px 130px 38px/,
    'la muestra no reproduce la anatomía de la fila');
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
  }
});
