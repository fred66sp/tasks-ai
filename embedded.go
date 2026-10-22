// Package embedded carries the files the product installs beside the binary.
package embedded

import "embed"

// Pi es la fuente de la extensión del agente, que el install copia dentro de la
// carpeta del producto para que el proyecto no dependa del repositorio de origen.
//
//go:embed extensions/pi/index.ts extensions/pi/board.ts extensions/pi/cli.ts extensions/pi/package.json
var Pi embed.FS
