# tasks-ai · Design System v3

## Identidad visual
Zorro aprobado con guiño pícaro, degradados cian/azul/violeta/magenta y hocico blanco. No reinterpretar, redibujar, vectorizar, deformar ni recolorear. El arte fuente es `master/fox-original-black-1254.png` (imagen original, fondo negro); `master/fox-transparent-native.png` es una extracción bitmap con transparencia aproximada, a validar en fondos distintos.

## Colores y temas
Usar `tokens.css` / `tokens.json` como base. Marca: violeta `#8B5CF6`, azul `#3882F6`, cian `#06B6D4`. Estados: éxito `#10B981`, aviso `#F59E0B`, error `#EF4444`. Tema oscuro: fondo `#0A0F1C`, superficie `#111827`, elevado `#1F2937`, borde `#334155`, texto `#E5E7EB`. Claro: fondo `#F8FAFC`, superficie `#FFFFFF`, elevado `#F1F5F9`, borde `#E2E8F0`, texto `#0F172A`.

## Tipografía
Inter para UI y documentación; JetBrains Mono para CLI, código e IDs. Las fuentes no se redistribuyen en este paquete.

## Componentes
Botones primario, secundario, fantasma y peligro; entradas, selects, tabs, badges, tarjetas y panel inspector. Priorizar densidad funcional del producto existente. Espaciado 4/8/12/16/24/32/48 px; radios 6/10/16 px. Estado no solo por color: acompañar con texto/icono.

## Accesibilidad
Contraste WCAG AA, foco visible, teclado, labels, `prefers-reduced-motion`, soporte 200 % zoom. No introducir cambios funcionales durante el cambio de identidad.

## Fidelidad
El archivo maestro aprobado es un bitmap de 1254 × 1254 píxeles sobre negro. La transparencia se ha extraído por segmentación y puede perder halo exterior. Los iconos de 1024 px no aportan detalle por encima del original. Comprobar a 16/32/180/512 px.
