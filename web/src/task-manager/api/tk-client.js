let clientId = '';

export function setClientId(value) {
  clientId = String(value || '');
}

/** Error normalizado que permite distinguir un fallo de transporte. */
export class ApiError extends Error {
  /**
   * @param {string} message Mensaje seguro para presentar al usuario.
   * @param {boolean} unavailable El servidor no responde en este anfitrion.
   */
  constructor(message, unavailable = false) {
    super(message);
    this.name = 'ApiError';
    this.unavailable = unavailable;
  }
}

/**
 * Frontera comun de transporte sobre la API del producto (docs/contracts.md,
 * secciones 2 y 3). Tambien trata `{ error }` con HTTP 200 como fallo.
 *
 * @param {string} path
 * @param {RequestInit} [options]
 * @returns {Promise<*>} JSON ya validado por transporte, no por dominio.
 * @throws {ApiError}
 */
async function apiRequest(path, options = {}) {
  let response;
  const requestOptions = { ...options };
  if (clientId) {
    requestOptions.headers = { ...(options.headers || {}), 'x-client-id': clientId };
  }
  try {
    response = await fetch(path, requestOptions);
  } catch (error) {
    throw new ApiError(error?.message || 'No se pudo conectar con el servidor.', true);
  }

  let data;
  try {
    data = await response.json();
  } catch {
    throw new ApiError('El servidor devolvió una respuesta no válida.');
  }

  if (!response.ok || data?.error) {
    throw new ApiError(data?.error || `La operación falló (${response.status}).`);
  }
  return data;
}

function jsonRequest(path, method, body) {
  return apiRequest(path, {
    method,
    headers: body === undefined ? undefined : { 'content-type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body)
  });
}

// La identidad de una tarea en la API es su referencia visible: tk-18.
export const listTasks = () => apiRequest('/api/tasks');
export const listTags = () => apiRequest('/api/tags');
export const showTask = (ref) => apiRequest(`/api/tasks/${encodeURIComponent(ref)}`);
export const createTask = (task) => jsonRequest('/api/tasks', 'POST', task);
export const updateTask = (ref, changes) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}`, 'PATCH', changes);
// La API pide el motivo en su propia ruta: la interfaz cierra con done, como
// venia haciendo.
export const closeTask = (ref) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/close`, 'POST', { reason: 'done' });
export const addTaskBlocker = (ref, blocker) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/blockers`, 'POST', { blocker });
export const removeTaskBlocker = (ref, blocker) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/blockers/${encodeURIComponent(blocker)}`, 'DELETE');
export const addTaskComment = (ref, body) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/comments`, 'POST', { body });
export const createTag = (tag) => jsonRequest('/api/tags', 'POST', tag);
export const updateTag = (name, tag) => jsonRequest(`/api/tags/${encodeURIComponent(name)}`, 'PATCH', tag);
export const deleteTag = (name) => jsonRequest(`/api/tags/${encodeURIComponent(name)}`, 'DELETE');
