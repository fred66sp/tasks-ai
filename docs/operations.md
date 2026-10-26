# Operación

Cómo se construye, se prueba y se ejecuta el producto. Las formas de datos y de API
están en `contracts.md`.

## Ejecutar

| Comando | Qué hace |
|---|---|
| `./tasks-ai init` | Instala el producto en el proyecto: crea `.tasks-ai/` con la base, el settings y las instrucciones, y devuelve el JSON del resultado. Con `--prefix` fija el prefijo de las referencias en la propia instalación, en vez de `tk`, y escribe las instrucciones con él. No es repetible: un proyecto ya instalado lo rechaza |
| `./tasks-ai serve` | Arranca la API y la interfaz web en la dirección y el puerto del settings: la interfaz responde en la raíz del servidor y la API en sus rutas `/api/...` |
| `./tasks-ai <comando>` | Ejecuta cualquier operación del producto con salida JSON |
| `./serve.sh` | Deja el servidor en segundo plano: compila el binario, lo arranca con `serve`, escribe su registro y su pid en `.run/` e informa de la dirección. `./serve.sh status` dice si está en marcha y `./serve.sh stop` lo para. Reconoce el servidor por el puerto que fija el settings, así que también lo ve y lo para si se arrancó a mano con `./tasks-ai serve`; un proceso ajeno en ese puerto no se toca: se nombra con su pid y su línea de órdenes, y el script sale con error |

Construir el binario, probarlo y construir la interfaz están en
[`development.md`](development.md); la decisión de incrustarla, en `ADR-010`.

## El tablero del trabajo

El trabajo del proyecto vive en `.tasks-ai/`, y se opera con el mismo binario:

```bash
./.tasks-ai/bin/tasks-ai list        # todas las tareas, con su estado y si están bloqueadas
./.tasks-ai/bin/tasks-ai show tk-1   # la tarea con sus hijas, sus bloqueos y sus comentarios
./.tasks-ai/bin/tasks-ai serve       # la interfaz web y su API
```

La herramienta de desarrollo anterior —un servidor en Node sobre `tk`— se retiró, y su
histórico se importó en este tablero (`ADR-005`).
