// Package embedded carries the files the product installs beside the binary.
package embedded

import "embed"

// Pi es la fuente de la extensión del agente, que el install copia dentro de la
// carpeta del producto para que el proyecto no dependa del repositorio de origen.
//
//go:embed extensions/pi/index.ts extensions/pi/board.ts extensions/pi/cli.ts extensions/pi/package.json
var Pi embed.FS

// OpenCode es la fuente del plugin del agente OpenCode, que el install copia con
// la extensión: reutiliza el CLI y el tablero de la de Pi, y no trae dependencias
// propias porque el host le aporta su paquete de plugins.
//
//go:embed extensions/opencode/index.ts extensions/opencode/tasks.ts extensions/opencode/package.json
var OpenCode embed.FS
