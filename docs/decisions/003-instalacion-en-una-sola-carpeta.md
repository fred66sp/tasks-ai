# ADR-003: Instalación en una sola carpeta del proyecto

- **Estado:** Aceptada
- **Fecha:** 2026-10-16

## Contexto

El producto se instala dentro de proyectos que ya existen y no debe ensuciarlos. El
producto anterior repartía su estado entre una base de tareas gestionada por un
binario de terceros y otra base propia para el catálogo de etiquetas, y se
configuraba con variables de entorno.

El Product Owner decidió que la instalación cree una sola carpeta con todo lo
necesario dentro del proyecto, y que cuando algo haya que hacer fuera de esa carpeta
se le entregue ya escrito.

## Decisión

- Todo lo del producto vive en `.task-ai/`, en la raíz del proyecto.
- Dentro están la base, el archivo de settings y las instrucciones.
- La base guarda las tareas, sus dependencias, sus comentarios y el catálogo de
  etiquetas con color y descripción.
- El producto localiza el proyecto por su carpeta: desde cualquier punto de su
  interior encuentra `.task-ai/`.
- Las instrucciones incluyen un `AGENTS.md` y, si hacen falta, skills.
- Fuera de esa carpeta solo va lo que el usuario decida añadir, con el texto
  preparado que el producto le ofrezca, por ejemplo un puntero en su `AGENTS.md`.

## Consecuencias

- Instalar en un proyecto no toca ningún otro archivo, y desinstalar es retirar una
  carpeta.
- La configuración vive en el proyecto y no depende de variables de entorno.
- La carpeta se versiona o se ignora según decida cada proyecto; el producto no lo
  impone.
- Requisitos afectados: `RF-03`, `RF-04`, `RF-05`, `RF-06`, `RF-08`, `CA-04`,
  `CA-05`, `CA-07`.

## Alternativas consideradas

- **Instalar en el directorio de la persona, con una carpeta común.** Rechazada:
  vuelve a mezclar proyectos y obliga a registrar cada uno.
- **Repartir los archivos por el proyecto: configuración en la raíz y base aparte.**
  Rechazada: ensucia el proyecto y complica el desinstalar.
- **Configurar con variables de entorno, como el producto anterior.** Rechazada: no
  sobreviven entre sesiones ni se documentan solas.
