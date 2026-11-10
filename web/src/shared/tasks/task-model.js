export const STATUS_LABELS = {
  open: 'Pendiente',
  in_progress: 'En curso',
  done: 'Hecha',
  blocked: 'Bloqueada'
};

export const PRIORITY_LABELS = ['Crítica', 'Alta', 'Media', 'Baja'];

export const statusLabel = (status) => STATUS_LABELS[status] || status || 'Sin estado';
export const priorityLabel = (priority) => PRIORITY_LABELS[Number(priority)] || 'Sin prioridad';
export const statusBadgeClass = (status) => ({
  open: 'tk-badge--warning',
  in_progress: 'tk-badge--info',
  done: 'tk-badge--success',
  blocked: 'tk-badge--danger'
}[status] || 'tk-badge--neutral');

// Lo que la fila y el inspector dicen de una tarea, y el icono que lo acompaña:
// los dos salen del mismo sitio para que cuenten lo mismo.
const TASK_KINDS = {
  epic: { type: 'Épica', icon: 'E' },
  subtask: { type: 'Subtarea', icon: 'S' },
  task: { type: 'Tarea', icon: 'T' }
};

// El detalle trae sus subtareas dentro; el listado, un aviso de que las tiene.
const hasSubtasks = (task) => Array.isArray(task.children)
  ? task.children.length > 0
  : Boolean(task.children);

/**
 * La API presenta cada tarea con su identidad interna y su referencia visible.
 * La interfaz sigue usando una sola identidad, y para ella vale la referencia:
 * es lo que las rutas de la API piden y lo que una persona ve.
 *
 * El tipo sale de la forma de la tarea: la etiqueta `epic` o sus subtareas la
 * hacen épica, y el padre distingue la subtarea de la tarea suelta. El listado
 * no trae las hijas, así que quien las conoce —el árbol— se lo dice.
 */
export function mapTask(task = {}, subtasks = hasSubtasks(task)) {
  const chips = Array.isArray(task.tags) ? task.tags : (Array.isArray(task.chips) ? task.chips : []);
  const parentId = task.parent_id || task.parentId || null;
  // El estado bloqueado no se guarda: la API lo deriva en un booleano. La
  // pantalla lo presenta como un estado, y de eso se encarga esta traducción.
  const blocked = Boolean(task.blocked) && task.status !== 'done';
  const epic = subtasks || chips.some((tag) => tag.toLocaleLowerCase('es') === 'epic');
  const kind = epic ? 'epic' : (parentId ? 'subtask' : 'task');
  const { parent_id: _parentId, tags: _tags, blockers = [], dependents = [], comments = [], ...rest } = task;

  return {
    ...rest,
    id: task.ref || task.id || '',
    title: task.title || '',
    description: task.description || '',
    priority: String(task.priority ?? 2),
    status: blocked ? 'blocked' : (task.status || 'open'),
    parentId,
    chips,
    kind,
    ...TASK_KINDS[kind],
    blockers: blockers.map((blocker) => mapRelation(blocker)),
    dependents: dependents.map((dependent) => mapRelation(dependent)),
    comments: comments.map(mapComment)
  };
}

// Una tarea abierta o en curso que lleva días sin actividad de tablero está
// fría. El umbral es fijo: es una señal para el equipo, no una preferencia.
export const COLD_TASK_DAYS = 7;
const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Tareas abiertas o en curso sin actividad de tablero reciente.
 *
 * La actividad es la fecha más nueva entre el cambio de la tarea y su último
 * comentario: comentar no toca `updated_at`, así que sin el máximo un
 * comentario de hoy no despertaría a una tarea de hace un mes.
 *
 * @param {Array<object>} tasks Tareas ya mapeadas, con `status` y `updated_at`.
 * @param {Array<{ref?: string, created_at?: string}>} comments Comentarios del proyecto.
 * @param {number} [now] Reloj inyectable, en milisegundos.
 * @param {number} [thresholdDays] Umbral en días.
 * @returns {Array<{id: string, title: string, days: number}>} De más fría a menos.
 */
export function coldTasks(tasks = [], comments = [], now = Date.now(), thresholdDays = COLD_TASK_DAYS) {
  const lastCommentAt = new Map();
  for (const comment of comments) {
    const timestamp = Date.parse(comment?.created_at);
    if (!comment?.ref || !Number.isFinite(timestamp)) continue;
    if (timestamp > (lastCommentAt.get(comment.ref) ?? -Infinity)) lastCommentAt.set(comment.ref, timestamp);
  }
  const threshold = thresholdDays * DAY_MS;
  return tasks
    .filter((task) => task.status === 'open' || task.status === 'in_progress')
    .map((task) => {
      const updatedAt = Date.parse(task.updated_at);
      const activity = Math.max(
        Number.isFinite(updatedAt) ? updatedAt : -Infinity,
        lastCommentAt.get(task.id) ?? -Infinity
      );
      if (!Number.isFinite(activity) || now - activity < threshold) return null;
      return { id: task.id, title: task.title, days: Math.floor((now - activity) / DAY_MS) };
    })
    .filter(Boolean)
    .sort((left, right) => right.days - left.days);
}

// Las relaciones del detalle llegan como tareas completas: vale su referencia.
function mapRelation(relation) {
  if (typeof relation === 'string') return { id: relation, title: '' };
  return { id: relation?.ref || relation?.id || '', title: relation?.title || '' };
}

function mapComment(comment) {
  if (typeof comment === 'string') return { id: null, body: comment, createdAt: '' };
  return {
    id: comment?.id ?? null,
    body: comment?.body || comment?.text || '',
    createdAt: comment?.created_at || comment?.createdAt || ''
  };
}
