#!/usr/bin/env bash
#
# Sirve este repositorio por HTTP para revisar la documentación y el sistema de diseño
# con sus enlaces relativos resueltos: por file:// no cargan ni el CSS ni los enlaces.
#
#   ./serve-web.sh          sirve en http://127.0.0.1:8765 hasta Ctrl+C
#   PORT=9000 ./serve-web.sh   cambia el puerto
#
# La raíz es el repositorio y no docs/web: las páginas enlazan ../PRD.md,
# ../contracts.md y ../../web/public/..., que solo resuelven desde ahí.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$root"

port="${PORT:-8765}"
address="http://127.0.0.1:$port"

if ! command -v python3 >/dev/null 2>&1; then
	echo "Falta python3: se usa su servidor HTTP para servir los archivos." >&2
	exit 1
fi

echo "Sirviendo $root en $address (Ctrl+C para parar)"
echo "  Guía de uso:    $address/docs/web/index.html"
echo "  Presentación:   $address/docs/web/landing.html"
echo "  Sistema visual: $address/docs/web/design-system.html"

exec python3 -m http.server --bind 127.0.0.1 "$port"
