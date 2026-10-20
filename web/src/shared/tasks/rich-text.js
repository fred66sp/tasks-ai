/**
 * El texto de una tarea —descripción, notas y comentarios— se escribe libremente
 * y se dibuja como HTML para que traiga su formato. Esta es la lista blanca de
 * lo que sobrevive: el texto entra por el CLI, por la API y pegado en la propia
 * pantalla, así que no puede ejecutar nada ni llevarse por delante la pantalla
 * (tk-59 cerró ese mismo agujero en los avisos).
 */
const ALLOWED_TAGS = new Set([
  'A', 'B', 'BLOCKQUOTE', 'BR', 'CODE', 'DD', 'DEL', 'DIV', 'DL', 'DT', 'EM',
  'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'HR', 'I', 'LI', 'MARK', 'OL', 'P', 'PRE',
  'S', 'SMALL', 'SPAN', 'STRIKE', 'STRONG', 'SUB', 'SUP', 'TABLE', 'TBODY', 'TD',
  'TH', 'THEAD', 'TR', 'U', 'UL'
]);

// Estas se van con lo que llevan dentro: su contenido no es texto que alguien
// quiera leer, y una etiqueta que carga o ejecuta algo no tiene lista blanca.
const DROPPED_TAGS = new Set([
  'APPLET', 'BASE', 'EMBED', 'FORM', 'IFRAME', 'INPUT', 'LINK', 'MATH', 'META',
  'NOSCRIPT', 'OBJECT', 'SCRIPT', 'STYLE', 'SVG', 'TEMPLATE', 'TEXTAREA'
]);

const ALLOWED_ATTRIBUTES = new Set(['href', 'title']);

// Una dirección sólo pasa si empieza por un esquema que no ejecuta nada.
const SAFE_URL = /^(https?:|mailto:|#|\/)/i;

// El analizador se crea al primer uso: fuera del navegador DOMParser no existe,
// y el módulo se puede importar sin que cargarlo reviente.
let parser = null;

function clean(node) {
  for (const child of Array.from(node.childNodes)) {
    if (child.nodeType === Node.TEXT_NODE) continue;
    if (child.nodeType !== Node.ELEMENT_NODE) {
      child.remove();
      continue;
    }
    const tag = child.tagName.toUpperCase();
    if (DROPPED_TAGS.has(tag)) {
      child.remove();
      continue;
    }
    if (!ALLOWED_TAGS.has(tag)) {
      // Una etiqueta que no se conoce no se tira con su texto: se suelta la
      // etiqueta y el texto se queda, limpio, en el sitio donde estaba.
      clean(child);
      while (child.firstChild) child.parentNode.insertBefore(child.firstChild, child);
      child.remove();
      continue;
    }
    for (const attribute of Array.from(child.attributes)) {
      const name = attribute.name.toLowerCase();
      const safe = ALLOWED_ATTRIBUTES.has(name) && (name !== 'href' || SAFE_URL.test(attribute.value.trim()));
      if (!safe) child.removeAttribute(attribute.name);
    }
    clean(child);
  }
}

export function richTextHtml(value) {
  const source = String(value ?? '');
  if (!source) return '';
  parser = parser || new DOMParser();
  // DOMParser no ejecuta guiones ni carga recursos: el documento no tiene
  // ventana. Lo que salga de aquí es lo que Vue inserta con v-html.
  const doc = parser.parseFromString(source, 'text/html');
  clean(doc.body);
  return doc.body.innerHTML;
}
