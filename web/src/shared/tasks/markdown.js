/**
 * El cuerpo de un ADR se escribe en markdown: el registro lo guarda tal cual y
 * la pantalla lo dibuja. Este conversor cubre lo que los ADR usan —párrafos,
 * listas, negrita, cursiva, código y enlaces— y escapa el HTML antes de
 * insertar etiquetas, así que el texto no puede traer las suyas. La lista
 * blanca de `rich-text.js` pasa después, como en el resto de la interfaz.
 */
const ESCAPES = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };

function escapeHtml(text) {
  return text.replace(/[&<>"']/g, (letter) => ESCAPES[letter]);
}

// El código en línea se aparta antes de aplicar el resto del formato: lo que
// va entre acentos graves se dibuja tal cual, sin negritas ni enlaces dentro.
function inline(text) {
  const codes = [];
  let out = text.replace(/`([^`]+)`/g, (_, code) => {
    codes.push(`<code>${code}</code>`);
    return `\u0000${codes.length - 1}\u0000`;
  });

  out = out.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '<a href="$2">$1</a>');
  out = out.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  out = out.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>');
  out = out.replace(/\u0000(\d+)\u0000/g, (_, index) => codes[Number(index)]);

  return out;
}

export function markdownHtml(value) {
  const source = escapeHtml(String(value ?? '').replace(/\r\n?/g, '\n'));
  const blocks = [];
  let paragraph = [];
  let list = [];
  let listOpen = false;

  const flushParagraph = () => {
    if (!paragraph.length) return;
    blocks.push(`<p>${inline(paragraph.join(' '))}</p>`);
    paragraph = [];
  };

  const flushList = () => {
    if (!listOpen) return;
    blocks.push(`<ul>${list.map((item) => `<li>${inline(item)}</li>`).join('')}</ul>`);
    list = [];
    listOpen = false;
  };

  for (const raw of source.split('\n')) {
    const line = raw.trim();
    if (!line) {
      flushParagraph();
      flushList();
      continue;
    }

    // Una línea sangrada continúa el punto anterior: los ADR vienen partidos
    // a mano y el markdown los une en el mismo párrafo del punto.
    if (listOpen && /^\s+/.test(raw)) {
      list[list.length - 1] += ` ${line}`;
      continue;
    }

    if (/^[-*]\s+/.test(line)) {
      flushParagraph();
      listOpen = true;
      list.push(line.replace(/^[-*]\s+/, ''));
      continue;
    }

    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    if (heading) {
      flushParagraph();
      flushList();
      blocks.push(`<h${heading[1].length}>${inline(heading[2])}</h${heading[1].length}>`);
      continue;
    }

    if (/^([-*_])\1{2,}$/.test(line)) {
      flushParagraph();
      flushList();
      blocks.push('<hr>');
      continue;
    }

    flushList();
    paragraph.push(line);
  }

  flushParagraph();
  flushList();

  return blocks.join('');
}
