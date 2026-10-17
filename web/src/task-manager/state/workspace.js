import { inspectorState } from './inspector.js';
import { tagActions } from './tag-actions.js';
import { taskActions } from './task-actions.js';
import { taskTree } from './task-tree.js';
import { workspaceData } from './workspace-data.js';
import { workspaceUi } from './ui.js';

/**
 * Los mixins se fusionan en una unica instancia Options API. Sus dependencias
 * mediante `this` son deliberadas y se documentan en docs/architecture.md.
 */
// Margen en el que la ventana reconoce como propio el aviso de la última
// tarea que escribió: cubre el retardo del canal —vive en la base y se lee por
// sondeo— sin alcanzar cambios posteriores de otra superficie.
// shortcut: dos ventanas que tocan la misma tarea dentro del margen hacen que
// una de ellas pierda un aviso; si algún día importa, que el aviso lleve la
// identidad de la escritura y se compare aquí.
const OWN_WRITE_MARGIN_MS = 2000;

export const workspaceOptions = {
  name: 'AppShell',
  mixins: [
    workspaceData,
    workspaceUi,
    taskTree,
    taskActions,
    tagActions,
    inspectorState
  ],
  watch: {
    activeInspectorTab() { this.scheduleUiPreferencesPersistence(); },
    activeTaskFilter() { this.scheduleUiPreferencesPersistence(); },
    currentPeriod() { this.scheduleUiPreferencesPersistence(); },
    selectedTaskId() { this.scheduleUiPreferencesPersistence(); },
    advancedStatuses() { this.scheduleUiPreferencesPersistence(); },
    advancedPriorities() { this.scheduleUiPreferencesPersistence(); },
    advancedTags() { this.scheduleUiPreferencesPersistence(); },
    quickTagFilter() { this.scheduleUiPreferencesPersistence(); },
    taskSearchQuery() { this.scheduleUiPreferencesPersistence(); },
    expandedIds() { this.scheduleUiPreferencesPersistence(); },
    collapsedIds() { this.scheduleUiPreferencesPersistence(); },
    areTaskChipsVisible() { this.scheduleUiPreferencesPersistence(); }
  },
  mounted() {
    window.addEventListener('keydown', this.handleEscape);
    this.restoreUiPreferences();
    this.connectWorkspaceEvents();
    this.reloadWorkspace();
  },
  beforeUnmount() {
    window.removeEventListener('keydown', this.handleEscape);
    this.cancelUiPreferencesPersistence();
    this.disconnectWorkspaceEvents();
  },
  methods: {
    connectWorkspaceEvents() {
      if (typeof EventSource === 'undefined') return;
      this.disconnectWorkspaceEvents();
      const source = new EventSource('/api/events');
      source.addEventListener('workspace.changed', (event) => {
        let change;
        try {
          change = JSON.parse(event.data);
        } catch {
          return;
        }
        this.noticeReload();
        // El aviso sale para cualquier cambio de tarea, venga de donde venga:
        // solo se lo calla la ventana que pidió el cambio, con su propio
        // estado local, sin identidad viajando al servidor.
        if (change.resource === 'tasks' && !this.suppressOwnWriteNotice(change.changes)) {
          this.showWorkspaceChangeToast?.(change.changes || []);
        }
      });
      this.workspaceEventSource = source;
    },
    disconnectWorkspaceEvents() {
      this.workspaceEventSource?.close();
      this.workspaceEventSource = null;
    },
    /**
     * Decide si un aviso del canal es el eco del cambio que esta ventana acaba
     * de escribir: si nombra la última tarea escrita dentro del margen, se
     * calla una vez y se consume, para que un cambio posterior sí avise.
     *
     * @param {Array<{id?: string}>} [changes]
     * @returns {boolean}
     */
    suppressOwnWriteNotice(changes) {
      if (!this.lastTaskWriteRef) return false;
      if (Date.now() - this.lastTaskWriteAt > OWN_WRITE_MARGIN_MS) return false;
      if (!Array.isArray(changes) || !changes.some((change) => change?.id === this.lastTaskWriteRef)) return false;
      this.lastTaskWriteRef = '';
      this.lastTaskWriteAt = 0;
      return true;
    }
  }
};
