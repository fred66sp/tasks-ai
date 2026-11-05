// Un icono del conjunto de trabajo. La hoja de estilos resuelve el archivo por
// el nombre y lo pinta con una máscara, así que el trazo del SVG se conserva y
// el color lo hereda del control. El icono es decorativo: el nombre accesible
// vive en el botón que lo contiene.
export const TkIcon = {
  name: 'TkIcon',
  props: {
    name: { type: String, required: true },
    spinning: { type: Boolean, default: false }
  },
  template: `
    <span :class="['tk-icon tk-icon--mask', 'tk-icon--' + name, { 'is-spinning': spinning }]"
      aria-hidden="true"></span>`
};
