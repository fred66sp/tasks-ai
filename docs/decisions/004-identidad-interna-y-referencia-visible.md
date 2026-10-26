---
status: accepted
date: 2026-10-16
decision-makers: "Alfredo González Portero"
---

# ADR-004: Identidad interna y referencia visible

## Contexto

La referencia funcional identifica cada tarea con una sola cadena: un prefijo más los
primeros dígitos hexadecimales de un valor aleatorio, con el prefijo fijado al crear
la base. Es un espacio corto y propio de cada base, y no existe nada parecido a una
identidad interna.

El Product Owner acordó separar la identidad interna de la referencia visible, y
cerró el mecanismo: prefijo más número, con el número interno presente en la salida
JSON. El prefijo no se cambiará en la práctica, y si se cambia el producto avisa de la
consecuencia.

## Decisión

- Cada tarea tiene un número interno, único dentro del proyecto, que es su identidad
  y no se reutiliza.
- La referencia visible se forma con el prefijo del proyecto y ese número.
- El número interno aparece en la salida JSON, para que un agente tenga una clave que
  no cambia.
- El prefijo es un ajuste del proyecto y vive en su settings.
- Al cambiar el prefijo, el producto advierte de que las referencias citadas fuera del
  producto dejan de resolver.

## Consecuencias

- La identidad de una tarea no cambia al cambiar el prefijo; lo que cambia es cómo se
  escribe.
- Nada escrito fuera del producto puede dar por garantizada una referencia: hay aviso
  cuando el prefijo cambia.
- El número interno es lo que viaja en la salida JSON y lo que se guarda en las
  relaciones.
- Requisitos afectados: `RN-01`, `RN-02`, `RN-03`, `RF-07`, `CA-03`.

## Alternativas consideradas

- **Una sola clave, como la referencia funcional.** Rechazada: cambiar el prefijo
  reescribiría la identidad y rompería las relaciones.
- **Guardar la referencia compuesta en cada tarea.** Rechazada: cambiar el prefijo
  obligaría a reescribir todas las filas y aun así rompería las citas externas.
- **Recordar los prefijos antiguos para que sigan resolviendo.** Rechazada: añade
  reglas permanentes para evitar un cambio que no se va a hacer.
