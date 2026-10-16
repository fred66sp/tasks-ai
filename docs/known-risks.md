# Riesgos y deuda

Qué riesgo existe, con qué se cubre y qué deuda se acepta. Este documento es un
registro: puede citar hechos con su fecha.

## Riesgos vivos

### RX-1 El diseño va por delante del código

Los requisitos, la arquitectura y los contratos están escritos antes de que exista
código que los cumpla. Lo que se documentó puede separarse de lo que acabe
implementándose.

**Cobertura:** el plan por fases contrasta el diseño con el código en cada fase y
corrige el documento en la misma fase, no después.

### RX-2 Nada comprueba lo escrito

No hay compilación ni pruebas que cubran los documentos ni, todavía, el producto. La
coherencia del conjunto documental se apoya en revisiones manuales.

**Cobertura:** las comprobaciones del contrato de documentación, hechas a mano, con su
entrada en el registro de verificaciones de `operations.md`. Cuando exista suite, esas
comprobaciones pasan a ejecutarse solas.

### RX-3 La auditoría de la interfaz de partida es de segunda mano

El análisis de la interfaz se hizo leyendo el código de otro repositorio, y se verificó
por muestreo. Puede haber capacidades o acoplamientos que no se vieran.

**Cobertura:** al integrar la interfaz, contrastar contra su código y sus pruebas antes
de dar por buena cualquier conclusión del análisis.

### RX-4 Los contratos adoptados apuntan fuera del repositorio

Los contratos de documentación y de flujo se adoptaron tal cual. Citan una suite
documental, un perfil de confirmación y rutas que en este repositorio no existen.

**Cobertura:** deuda aceptada. Se revisan al primer uso y, si estorban, se decide si se
adaptan o se retiran, con su documento de decisión.

### RX-5 El catálogo de etiquetas se reimplementa sin pruebas heredadas

El catálogo con color, descripción y aviso de uso vivía en el servidor anterior, y su
cascada al renombrar o borrar no tiene pruebas nuestras.

**Cobertura:** pruebas del núcleo en la fase que implemente el catálogo.

## Deuda aceptada

- La interfaz de partida trae piezas sin uso, como un componente de aviso que nadie
  importa, y endpoints que no consume. Se decide qué se hace con ellas al adaptarla.
- El repositorio no está versionado en ningún remoto: el trabajo vive solo en local.
