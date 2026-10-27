import { workspaceOptions } from './state/workspace.js';
import { TagManagerDrawer } from './components/tag-manager-drawer.js';
import { NewTaskDialog } from './components/new-task-dialog.js';
import { TaskList } from './components/task-list.js';
import { TaskInspector } from './components/task-inspector.js';
import { DecisionsPanel } from './components/decisions-panel.js';

export const AppShell = {
  ...workspaceOptions,
  name: 'AppShell',
  components: { NewTaskDialog, TagManagerDrawer, TaskInspector, TaskList, DecisionsPanel },
  template: `
    <main class="workspace-frame" @click="handleWorkspaceClick">
  <div class="workspace-app">
    <header class="workspace-header">
      <div class="workspace-header__brand">
        <!-- El símbolo aprobado vive en web/public/brand/, copiado del paquete de marca de docs/tasks-ai-brand. -->
        <div class="brand-mark"><img src="../brand/tasks-ai-icon-96.png" alt="" width="38" height="38" /></div>
        <div><strong>Gestor de tareas</strong><span>Local</span></div>
      </div>
      <div class="workspace-header__actions">
        <button class="tk-icon-button" type="button" aria-label="Actualizar datos" title="Actualizar datos"
          :disabled="refreshing || loading" @click.stop="reloadWorkspace()"><span class="tk-icon"
            :class="{ 'is-spinning': refreshing }">↻</span></button>
        <button class="tk-button tk-button--danger" type="button"
          :disabled="!hasSelection || selectedTask.status === 'done' || isBusy('close:' + selectedTaskId)"
          @click.stop="closeTask()">Cerrar tarea</button>
        <button class="tk-button tk-button--secondary" type="button" data-decisions-open=""
          :disabled="unavailable || loading" @click.stop="isDecisionsOpen = true">Decisiones</button>
        <button class="tk-button tk-button--secondary" type="button" data-tag-manager-open=""
          :disabled="unavailable || loading" @click.stop="isTagManagerOpen = true">Etiquetas</button>
        <button class="tk-button tk-button--primary" type="button" :disabled="unavailable || loading"
          @click.stop="newTaskDraft = emptyTaskDraft(); isNewTaskOpen = true">Nueva tarea</button>
      </div>
    </header>
    <section class="workspace-body">
      <task-list :loading="loading" :has-tasks="Boolean(tasks.length)" :tasks="visibleTasks"
        :active-count="activeTaskCount" :selected-id="selectedTaskId" :expanded-ids="expandedIds"
        :chips-visible="areTaskChipsVisible" :open-menu-id="openTaskMenuId" :busy="isBusy" :tag-color="tagColor"
        :filter-props="taskFilterProps" @refresh="reloadWorkspace()" @dismiss-error="errorMessage = ''"
        @update-search="taskSearchQuery = $event" @toggle-filters="isAdvancedFiltersOpen = !isAdvancedFiltersOpen"
        @close-filters="isAdvancedFiltersOpen = false" @clear-filters="clearAdvancedFilters"
        @update-statuses="advancedStatuses = $event" @update-priorities="advancedPriorities = $event"
        @update-tags="advancedTags = $event" @update-quick-filter="quickTagFilter = $event"
        @update-active-filter="activeTaskFilter = $event" @update-current-period="currentPeriod = $event"
        @toggle-chips="areTaskChipsVisible = !areTaskChipsVisible"
        @toggle-list-menu="isListMenuOpen = !isListMenuOpen" @set-expanded="setAllExpanded" @select="loadTaskDetail"
        @toggle-expanded="toggleExpanded" @update-field="updateTaskField" @toggle-menu="toggleTaskMenu"
        @menu-action="runTaskMenuAction"></task-list>
      <task-inspector v-bind="taskInspectorProps" v-model:mobile-open="isMobileInspectorOpen"
        v-model:active-tab="activeInspectorTab" v-model:edit-draft="editDraft"
        v-model:dependency-draft="dependencyDraft" v-model:comment-draft="commentDraft"
        v-model:editing-comment-draft="editingCommentDraft"
        @action="handleInspectorAction"></task-inspector>
    </section>
  </div>
</main>
<new-task-dialog :open="isNewTaskOpen" :draft="newTaskDraft" :tasks="mappedTasks" :tags="tagCatalog" :colors="colors"
  :busy="isBusy('create')" @close="isNewTaskOpen = false" @update-draft="newTaskDraft = $event" @submit="createTask">
</new-task-dialog>
<decisions-panel :open="isDecisionsOpen" :refresh-token="decisionsToken" @close="isDecisionsOpen = false">
</decisions-panel>
<tag-manager-drawer :open="isTagManagerOpen" :tags="tagCatalog" :visible-tags="visibleTags" :colors="colors"
  :search-query="tagSearchQuery" :form-mode="tagFormMode" :draft="tagDraft" :editing-name="editingTagName"
  :form-error="tagFormError" :busy="isBusy" :color-hex="colorHex"
  @close="isTagManagerOpen = false" @update-search="tagSearchQuery = $event"
  @open-create="openCreateTag" @open-edit="openEditTag" @request-delete="requestDeleteTag"
  @update-draft="tagDraft = $event" @close-form="closeTagForm" @save="saveTag"></tag-manager-drawer>
`
};
