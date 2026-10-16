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

/**
 * La API presenta cada tarea con su identidad interna y su referencia visible.
 * La interfaz sigue usando una sola identidad, y para ella vale la referencia:
 * es lo que las rutas de la API piden y lo que una persona ve.
 */
export function mapTask(task = {}) {
  const chips = Array.isArray(task.tags) ? task.tags : (Array.isArray(task.chips) ? task.chips : []);
  const parentId = task.parent_id || task.parentId || null;
  // El estado bloqueado no se guarda: la API lo deriva en un booleano. La
  // pantalla lo presenta como un estado, y de eso se encarga esta traducción.
  const blocked = Boolean(task.blocked) && task.status !== 'done';
  const epic = chips.some((tag) => tag.toLocaleLowerCase('es') === 'epic');
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
    kind: epic ? 'epic' : (parentId ? 'subtask' : 'task'),
    type: epic ? 'Épica' : (parentId ? 'Subtarea' : 'Tarea'),
    icon: epic ? 'E' : (parentId ? 'S' : 'T'),
    blockers: blockers.map((blocker) => mapRelation(blocker)),
    dependents: dependents.map((dependent) => mapRelation(dependent)),
    comments: comments.map(mapComment)
  };
}

// Las relaciones del detalle llegan como tareas completas: vale su referencia.
function mapRelation(relation) {
  if (typeof relation === 'string') return { id: relation, title: '' };
  return { id: relation?.ref || relation?.id || '', title: relation?.title || '' };
}

function mapComment(comment) {
  if (typeof comment === 'string') return { body: comment, author: '', createdAt: '' };
  return {
    body: comment?.body || comment?.text || '',
    author: comment?.author || '',
    createdAt: comment?.created_at || comment?.createdAt || ''
  };
}
