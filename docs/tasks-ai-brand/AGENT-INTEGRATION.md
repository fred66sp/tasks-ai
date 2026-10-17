# tasks-ai · Instrucciones para agente de implementación (v3)

Objetivo: aplicar la identidad aprobada sin alterar la lógica, rutas, filtros, estados, jerarquías, etiquetas, dependencias ni persistencia.

1. Inventariar UI, CSS, favicons, rutas de estáticos, empaquetado y capturas base. Crear rama de trabajo según contrato Git vigente.
2. Usar el maestro `master/fox-original-black-1254.png` como referencia visual. **No usar SVG ni recreaciones del zorro**. Para fondos transparentes usar `master/fox-transparent-native.png` y revisar bordes/halos en temas claro y oscuro.
3. Instalar iconos desde `favicons/` según framework. Configurar `site.webmanifest` y rutas absolutas/relativas reales. Para Windows usar ICO; para otras plataformas adaptar PNG al formato requerido por el empaquetador, sin cambiar el arte.
4. Aplicar `design/tokens.css` progresivamente, mapeando variables actuales; no hacer una reescritura de la UI. Validar contraste, focus, tabs, selects, chips y paneles.
5. Usar `logos/tasks-ai-horizontal-dark.png` en fondos oscuros y variante light en claros. Las variantes `-transparent` permiten composición, pero comprobar contornos.
6. Pruebas visuales: tamaños 16/32/180/512; fondos oscuros/claros; navegador/escritorio; zoom 200 %; navegación por teclado; tests actuales y build.
7. Documentar archivos cambiados, capturas antes/después, pruebas y limitaciones. No hacer commit sin autorización si el flujo vigente así lo exige.
