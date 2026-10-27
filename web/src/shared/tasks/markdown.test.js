import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { markdownHtml } from './markdown.js';

// El conversor compone el HTML del cuerpo de un ADR y no decide qué esquemas de
// enlace valen: eso lo quita `richTextHtml`, que pasa después. Estas pruebas
// fijan las dos mitades de ese reparto.
describe('markdownHtml', () => {
  it('escapa el HTML del texto antes de componer el suyo', () => {
    assert.equal(
      markdownHtml('<script>alert(1)</script>'),
      '<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>'
    );
  });

  it('deja el código en línea como texto, sin que sus etiquetas se dibujen', () => {
    assert.equal(markdownHtml('`<b>x</b>`'), '<p><code>&lt;b&gt;x&lt;/b&gt;</code></p>');
  });

  it('compone párrafos, listas, encabezados, negrita y enlaces', () => {
    assert.equal(
      markdownHtml('## Decisión\n\n- uno\n- dos\n\nEl **cuerpo** y [el índice](README.md).'),
      '<h2>Decisión</h2><ul><li>uno</li><li>dos</li></ul>' +
        '<p>El <strong>cuerpo</strong> y <a href="README.md">el índice</a>.</p>'
    );
  });

  it('junta en el mismo punto la línea sangrada que lo continúa', () => {
    assert.equal(markdownHtml('- uno\n  sigue aquí'), '<ul><li>uno sigue aquí</li></ul>');
  });

  it('entrega el enlace a la lista blanca: el esquema peligroso no se filtra aquí', () => {
    assert.equal(markdownHtml('[x](javascript:alert)'), '<p><a href="javascript:alert">x</a></p>');
  });
});
