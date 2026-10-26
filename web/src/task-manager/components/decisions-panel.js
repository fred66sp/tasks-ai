import { dialogFocus } from './dialog-focus.js';
import { richTextHtml, isPlainText } from '../../shared/tasks/rich-text.js';
import { createADR, listADRs, showADR, updateADR } from '../api/tk-client.js';

// Los estados que un ADR puede tener y el nombre que la pantalla les da.
const STATUSES = ['proposed', 'accepted', 'rejected', 'deprecated', 'superseded'];
const STATUS_LABELS = {
  proposed: 'Propuesta',
  accepted: 'Aceptada',
  rejected: 'Rechazada',
  deprecated: 'Obsoleta',
  superseded: 'Sustituida'
};
const STATUS_CLASSES = {
  proposed: 'is-proposed',
  accepted: 'is-accepted',
  rejected: 'is-rejected',
  deprecated: 'is-deprecated',
  superseded: 'is-superseded'
};
const BODY_FIELDS = ['title', 'context', 'decision', 'consequences', 'alternatives'];

export const DecisionsPanel = {
  name: 'DecisionsPanel',
  mixins: [dialogFocus],
  props: {
    open: { type: Boolean, required: true },
    // Aviso de un cambio hecho desde fuera de esta ventana: el tablero lo
    // incrementa con cada aviso del canal, y aquí relee el registro.
    refreshToken: { type: Number, default: 0 }
  },
  emits: ['close'],
  data() {
    return {
      adrs: [],
      selected: 0,
      detail: null,
      search: '',
      statusFilter: '',
      formMode: '',
      form: null,
      formError: '',
      listError: '',
      busy: false,
      statuses: STATUSES
    };
  },
  computed: {
    visible() {
      const query = this.normalize(this.search);
      return this.adrs.filter((adr) => {
        if (this.statusFilter && adr.status !== this.statusFilter) return false;
        if (!query) return true;
        return this.normalize(adr.ref).includes(query) || this.normalize(adr.title).includes(query);
      });
    },
    // El cuerpo se congela al salir de proposed: la pantalla deja de ofrecerlo.
    frozenBody() {
      return Boolean(this.detail) && this.detail.status !== 'proposed';
    },
    sections() {
      if (!this.detail) return [];
      return [
        { heading: 'Contexto', field: 'context' },
        { heading: 'Decisión', field: 'decision' },
        { heading: 'Consecuencias', field: 'consequences' },
        { heading: 'Alternativas consideradas', field: 'alternatives' },
        { heading: 'Más información', field: 'more_info' }
      ].filter((section) => String(this.detail[section.field] || '').trim());
    }
  },
  watch: {
    open(value) {
      if (value) this.load();
    },
    refreshToken() {
      if (this.open) this.load();
    }
  },
  methods: {
    normalize(value) {
      return String(value || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().trim();
    },
    statusLabel(status) {
      return STATUS_LABELS[status] || status;
    },
    statusClass(status) {
      return STATUS_CLASSES[status] || '';
    },
    html(value) {
      return richTextHtml(value);
    },
    plainText(value) {
      return isPlainText(value);
    },
    async load() {
      this.listError = '';
      try {
        this.adrs = await listADRs();
      } catch (error) {
        this.listError = error?.message || 'No se pudieron leer las decisiones.';
      }
    },
    emptyForm() {
      return {
        title: '',
        deciders: '',
        status: 'proposed',
        supersededBy: '',
        context: '',
        decision: '',
        consequences: '',
        alternatives: '',
        note: ''
      };
    },
    startNew() {
      this.formMode = 'new';
      this.form = this.emptyForm();
      this.formError = '';
      this.selected = 0;
      this.detail = null;
    },
    startEdit() {
      if (!this.detail) return;
      this.formMode = 'edit';
      this.form = {
        title: this.detail.title,
        deciders: this.detail.deciders,
        status: this.detail.status,
        supersededBy: this.detail.superseded_by || '',
        context: this.detail.context || '',
        decision: this.detail.decision || '',
        consequences: this.detail.consequences || '',
        alternatives: this.detail.alternatives || '',
        note: ''
      };
      this.formError = '';
    },
    cancelForm() {
      this.formMode = '';
      this.form = null;
      this.formError = '';
    },
    async openADR(number) {
      if (!number) return;
      this.formMode = '';
      this.form = null;
      this.selected = number;
      try {
        this.detail = await showADR(number);
      } catch (error) {
        this.listError = error?.message || 'No se pudo leer la decisión.';
      }
    },
    closeDetail() {
      this.detail = null;
      this.selected = 0;
      this.formMode = '';
      this.form = null;
    },
    // El sustituto viaja como referencia: ADR-014.
    referenceNumber(reference) {
      return Number(String(reference || '').replace(/\D/g, ''));
    },
    async save() {
      if (!this.form || this.busy) return;
      this.busy = true;
      this.formError = '';
      try {
        const number = this.formMode === 'edit' ? await this.edit() : await this.create();
        if (number) {
          await this.load();
          await this.openADR(number);
        }
      } catch (error) {
        this.formError = error?.message || 'No se pudo guardar la decisión.';
      } finally {
        this.busy = false;
      }
    },
    async create() {
      const created = await createADR({
        title: this.form.title,
        deciders: this.form.deciders,
        context: this.form.context,
        decision: this.form.decision,
        consequences: this.form.consequences,
        alternatives: this.form.alternatives
      });
      return created.number;
    },
    async edit() {
      const changes = {};
      if (!this.frozenBody) {
        for (const field of BODY_FIELDS) {
          if (this.form[field] !== (this.detail[field] || '')) changes[field] = this.form[field];
        }
      }
      if (this.form.status !== this.detail.status) changes.status = this.form.status;
      if (this.form.status === 'superseded' && this.form.supersededBy !== (this.detail.superseded_by || '')) {
        changes.superseded_by = this.form.supersededBy;
      }
      if (this.form.note.trim()) changes.note = this.form.note;
      if (!Object.keys(changes).length) return this.detail.number;
      const updated = await updateADR(this.detail.number, changes);
      return updated.number;
    }
  },
  template: `
    <section v-show="open" class="decisions-layer is-open">
  <div class="decisions-panel" ref="dialogRoot" tabindex="-1" role="dialog" aria-modal="true"
    aria-labelledby="decisions-title" @keydown.tab="trapDialogTab">
    <header class="decisions-header">
      <div>
        <h2 id="decisions-title">Decisiones</h2>
        <p>Los ADR del proyecto</p>
      </div>
      <div class="decisions-header__actions"><button class="tk-button tk-button--secondary" type="button"
          @click="startNew()">Nueva decisión</button><button class="tk-icon-button" type="button"
          aria-label="Cerrar decisiones" title="Cerrar decisiones" @click="$emit('close')">×</button></div>
    </header>
    <div class="decisions-body" :class="{ 'is-detail': Boolean(detail) || Boolean(formMode) }">
      <aside class="decisions-list">
        <div class="decisions-toolbar"><input class="tk-input" type="search" placeholder="Buscar por número o título"
            :value="search" @input="search = $event.target.value" /><select class="tk-input" :value="statusFilter"
            @change="statusFilter = $event.target.value">
            <option value="">Todos los estados</option>
            <option v-for="status in statuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
          </select></div>
        <p v-show="listError" class="decisions-error">{{ listError }}</p>
        <ul class="decisions-items">
          <li v-for="adr in visible" :key="adr.number"><button class="decisions-item" type="button"
              :class="{ 'is-active': adr.number === selected }" @click="openADR(adr.number)">
              <span class="decisions-item__ref">{{ adr.ref }}</span><span
                class="decisions-item__title">{{ adr.title }}</span><span class="decisions-item__meta"><span
                  class="decisions-status" :class="statusClass(adr.status)">{{ statusLabel(adr.status) }}</span><time
                  :datetime="adr.date">{{ adr.date }}</time></span></button></li>
          <li v-show="!visible.length" class="decisions-empty">No hay decisiones que coincidan.</li>
        </ul>
      </aside>
      <article class="decisions-detail"><button class="decisions-back" type="button" @click="closeDetail()">←
          Volver</button>
        <p v-if="!detail && !formMode" class="decisions-empty">Elige una decisión para leerla.</p>
        <form v-if="formMode" class="decisions-form" @submit.prevent="save()">
          <p class="decisions-form__title">{{ formMode === 'new' ? 'Nueva decisión' : 'Editar ' + detail.ref }}</p>
          <div class="decisions-form__grid"><label class="decisions-field"><span class="decisions-label">Título</span><input
                class="tk-input" required :value="form.title" :disabled="frozenBody"
                @input="form.title = $event.target.value" /></label><label class="decisions-field"><span
                class="decisions-label">Decisores</span><input class="tk-input" required :value="form.deciders"
                :disabled="formMode === 'edit'" @input="form.deciders = $event.target.value" /></label><label
              class="decisions-field"><span class="decisions-label">Estado</span><select class="tk-input"
                :value="form.status" @change="form.status = $event.target.value">
                <option v-for="status in statuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
              </select></label><label v-show="form.status === 'superseded'" class="decisions-field"><span
                class="decisions-label">Sustituto</span><input class="tk-input" placeholder="ADR-017"
                :value="form.supersededBy" @input="form.supersededBy = $event.target.value" /></label><label
              class="decisions-field decisions-field--wide"><span class="decisions-label">Contexto</span><textarea
                class="tk-input" rows="4" :value="form.context" :disabled="frozenBody"
                @input="form.context = $event.target.value"></textarea></label><label
              class="decisions-field decisions-field--wide"><span class="decisions-label">Decisión</span><textarea
                class="tk-input" rows="4" :value="form.decision" :disabled="frozenBody"
                @input="form.decision = $event.target.value"></textarea></label><label
              class="decisions-field decisions-field--wide"><span class="decisions-label">Consecuencias</span><textarea
                class="tk-input" rows="4" :value="form.consequences" :disabled="frozenBody"
                @input="form.consequences = $event.target.value"></textarea></label><label
              class="decisions-field decisions-field--wide"><span class="decisions-label">Alternativas consideradas</span><textarea
                class="tk-input" rows="4" :value="form.alternatives" :disabled="frozenBody"
                @input="form.alternatives = $event.target.value"></textarea></label><label v-show="formMode === 'edit'"
              class="decisions-field decisions-field--wide"><span class="decisions-label">Añadir a más información</span><textarea
                class="tk-input" rows="3" :value="form.note"
                @input="form.note = $event.target.value"></textarea></label></div>
          <p v-show="frozenBody" class="decisions-hint">El cuerpo está congelado: una decisión aceptada solo admite
            añadidos en «Más información».</p>
          <p v-show="formError" class="decisions-error">{{ formError }}</p>
          <div class="decisions-form__actions"><button class="tk-button tk-button--ghost" type="button"
              @click="cancelForm()">Cancelar</button><button class="tk-button tk-button--primary" type="submit"
              :disabled="busy">Guardar</button></div>
        </form>
        <div v-else-if="detail" class="decisions-document">
          <header class="decisions-document__header">
            <div class="decisions-document__ref">{{ detail.ref }}</div>
            <h3>{{ detail.title }}</h3>
            <div class="decisions-document__meta"><span class="decisions-status"
                :class="statusClass(detail.status)">{{ statusLabel(detail.status) }}</span><time
                :datetime="detail.date">{{ detail.date }}</time><span>{{ detail.deciders }}</span><button
                class="tk-button tk-button--secondary" type="button" @click="startEdit()">Editar</button></div>
          </header>
          <p v-show="detail.superseded_by" class="decisions-substitute">Sustituido por <button class="decisions-link"
              type="button"
              @click="openADR(referenceNumber(detail.superseded_by))">{{ detail.superseded_by }}</button></p>
          <section v-for="section in sections" :key="section.field" class="decisions-section">
            <h4>{{ section.heading }}</h4>
            <div :class="['tk-rich-text', plainText(detail[section.field]) && 'tk-rich-text--plain']"
              v-html="html(detail[section.field])"></div>
          </section>
        </div>
      </article>
    </div>
  </div>
</section>`
};
