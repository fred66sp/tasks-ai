# ADR-007: SQLite directo, sin ORM

- **Estado:** Aceptada
- **Fecha:** 2026-10-16

## Contexto

El producto guarda todo su estado en SQLite, dentro de su carpeta en el proyecto. El
producto anterior ya accedía a la base directamente, sin capa intermedia, y el
Product Owner fijó el criterio de evitar un ORM y las capas innecesarias salvo
justificación.

## Decisión

- El acceso a datos se escribe directamente sobre SQLite.
- No se incorpora un ORM.
- Cualquier capa de indirección sobre el acceso a datos deberá justificarse.

## Consecuencias

- Las consultas y las migraciones se escriben a mano.
- El esquema no depende de las convenciones de una herramienta ajena.
- Añadir una capa es una decisión, no una comodidad por defecto.
- Requisitos afectados: `RNF-01`.

## Alternativas consideradas

- **Un ORM.** Rechazada: añade una dependencia y una traducción del esquema que este
  producto no necesita.
- **Una capa de repositorio sobre el acceso directo.** No se descarta de plano, pero
  no entra por defecto: tendría que justificarse con una necesidad concreta.
