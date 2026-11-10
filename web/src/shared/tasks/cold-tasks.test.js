import assert from 'node:assert/strict';
import test from 'node:test';
import { COLD_TASK_DAYS, coldTasks } from './task-model.js';

const DAY = 24 * 60 * 60 * 1000;
const NOW = Date.parse('2026-11-10T12:00:00Z');
const ago = (days, extraMs = 0) => new Date(NOW - days * DAY - extraMs).toISOString();
const task = (id, overrides = {}) => ({ id, title: `Tarea ${id}`, status: 'open', updated_at: ago(1), ...overrides });

test('el umbral es fijo en siete días', () => {
  assert.equal(COLD_TASK_DAYS, 7);
});

test('una tarea sin movimiento desde hace ocho días está fría', () => {
  const cold = coldTasks([task('tk-1', { updated_at: ago(8) })], [], NOW);

  assert.deepEqual(cold, [{ id: 'tk-1', title: 'Tarea tk-1', days: 8 }]);
});

test('una tarea con movimiento hace seis días no está fría', () => {
  assert.deepEqual(coldTasks([task('tk-1', { updated_at: ago(6) })], [], NOW), []);
});

test('el umbral exacto cuenta como fría y un milisegundo antes no', () => {
  assert.deepEqual(coldTasks([task('tk-1', { updated_at: ago(7) })], [], NOW), [
    { id: 'tk-1', title: 'Tarea tk-1', days: 7 }
  ]);
  assert.deepEqual(coldTasks([task('tk-1', { updated_at: ago(7, -1) })], [], NOW), []);
});

test('un comentario reciente resucita la tarea aunque el cambio sea antiguo', () => {
  const tasks = [task('tk-1', { updated_at: ago(30) })];
  const comments = [{ ref: 'tk-1', created_at: ago(2) }];

  assert.deepEqual(coldTasks(tasks, comments, NOW), []);
});

test('un cambio reciente no se vuelve frío por comentarios antiguos', () => {
  const tasks = [task('tk-1', { updated_at: ago(1) })];
  const comments = [{ ref: 'tk-1', created_at: ago(40) }];

  assert.deepEqual(coldTasks(tasks, comments, NOW), []);
});

test('entre varios comentarios manda el más nuevo, y solo cuenta el de su tarea', () => {
  const tasks = [task('tk-1', { updated_at: ago(30) }), task('tk-2', { updated_at: ago(30) })];
  const comments = [
    { ref: 'tk-1', created_at: ago(40) },
    { ref: 'tk-1', created_at: ago(3) },
    { ref: 'tk-2', created_at: ago(20) }
  ];

  assert.deepEqual(coldTasks(tasks, comments, NOW), [
    { id: 'tk-2', title: 'Tarea tk-2', days: 20 }
  ]);
});

test('las tareas hechas o bloqueadas nunca están frías', () => {
  const tasks = [
    task('tk-1', { status: 'done', updated_at: ago(30) }),
    task('tk-2', { status: 'blocked', updated_at: ago(30) }),
    task('tk-3', { status: 'in_progress', updated_at: ago(30) })
  ];

  assert.deepEqual(coldTasks(tasks, [], NOW), [
    { id: 'tk-3', title: 'Tarea tk-3', days: 30 }
  ]);
});

test('se ordenan de la más fría a la menos fría', () => {
  const tasks = [
    task('tk-1', { updated_at: ago(9) }),
    task('tk-2', { updated_at: ago(30) }),
    task('tk-3', { updated_at: ago(15) })
  ];

  assert.deepEqual(coldTasks(tasks, [], NOW).map((entry) => entry.id), ['tk-2', 'tk-3', 'tk-1']);
});

test('una tarea sin fecha válida no se declara fría', () => {
  assert.deepEqual(coldTasks([task('tk-1', { updated_at: '' })], [], NOW), []);
});
