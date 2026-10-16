// Package web trae consigo la interfaz ya construida: la carpeta public/ con
// sus páginas, sus guiones y sus estilos viaja incrustada en el binario, de
// modo que un solo comando serve la reparte junto con la API. El artefacto
// construido —public/task-manager/app-vue.js y su CSS— se versiona: lo
// construye la herramienta de desarrollo de la interfaz, no el compilador de
// Go (ADR-010).
package web

import "embed"

// Files es la interfaz construida, tal como la deja npm run build en web/.
//
//go:embed public
var Files embed.FS
