import { dialogFocus } from './dialog-focus.js';
import { richTextHtml } from '../../shared/tasks/rich-text.js';
import { markdownHtml } from '../../shared/tasks/markdown.js';
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
// Las insignias del resto de la interfaz: el estado de un ADR se lee como el de
// una tarea, no con una pieza propia.
const STATUS_BADGES = {
  proposed: 'tk-badge--warning',
  accepted: 'tk-badge--success',
  rejected: 'tk-badge--danger',
  deprecated: 'tk-badge--neutral',
  superseded: 'tk-badge--info'
};
// Sustituir exige nombrar el sustituto y es un cambio posterior, no parte del
// nacimiento: una decisión nueva no puede sustituir a nada.
const CREATE_STATUSES = ['proposed', 'accepted', 'rejected', 'deprecated'];
const BODY_FIELDS = ['title', 'context', 'decision', 'consequences', 'alternatives'];
const PAGE_SIZES = [10, 20, 40, 60];

// Lo que cada sección del cuerpo se llama en pantalla, en el orden del registro.
const SECTIONS = [
  { heading: 'Contexto', field: 'context' },
  { heading: 'Decisión', field: 'decision' },
  { heading: 'Consecuencias', field: 'consequences' },
  { heading: 'Alternativas consideradas', field: 'alternatives' },
  { heading: 'Más información', field: 'more_info' }
];

// El buscador pregunta al servidor por todos los campos; se espera a que la
// persona deje de teclear para no lanzar una petición por pulsación.
const SEARCH_DELAY_MS = 220;

// El botón de vuelta arriba aparece cuando la cabecera del registro ya no se
// ve, como en la documentación de SKILLS-AGP.
const TO_TOP_AFTER_PX = 480;

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
      // Cada tarjeta abierta y su cuerpo, que se pide al desplegarla. El
      // plegado vive de la lista: su contexto ya viene en el resumen.
      expanded: [],
      details: {},
      editing: 0,
      creating: false,
      draft: null,
      formError: '',
      search: '',
      statusFilter: '',
      listError: '',
      busy: false,
      statuses: STATUSES,
      createStatuses: CREATE_STATUSES,
      page: 1,
      pageSize: PAGE_SIZES[0],
      pageSizes: PAGE_SIZES,
      showToTop: false,
      // La última búsqueda pedida manda: una respuesta lenta no pisa a la que
      // llegó después.
      loadSeq: 0,
      searchTimer: 0
    };
  },
  computed: {
    visible() {
      if (!this.statusFilter) return this.adrs;
      return this.adrs.filter((adr) => adr.status === this.statusFilter);
    },
    pageCount() {
      return Math.max(1, Math.ceil(this.visible.length / this.pageSize));
    },
    paged() {
      const start = (this.page - 1) * this.pageSize;
      return this.visible.slice(start, start + this.pageSize);
    },
    rangeLabel() {
      const total = this.visible.length;
      if (!total) return '0 decisiones';
      if (this.pageCount === 1) return total === 1 ? '1 decisión' : `${total} decisiones`;
      const last = Math.min(this.page * this.pageSize, total);
      return `${(this.page - 1) * this.pageSize + 1}–${last} de ${total}`;
    },
    // El cuerpo se congela al salir de proposed: la pantalla deja de ofrecerlo.
    frozenBody() {
      const detail = this.editing ? this.details[this.editing] : null;
      return Boolean(detail) && detail.status !== 'proposed';
    }
  },
  mounted() {
    // Escape llega antes por la ventana que al cierre global: mientras se
    // escribe, cancela la edición y no deja que la capa se cierre con el
    // borrador dentro, venga el foco de donde venga.
    window.addEventListener('keydown', this.handleEscape, true);
  },
  beforeUnmount() {
    window.removeEventListener('keydown', this.handleEscape, true);
  },
  watch: {
    search() {
      this.page = 1;
      clearTimeout(this.searchTimer);
      this.searchTimer = setTimeout(() => this.load(), SEARCH_DELAY_MS);
    },
    statusFilter() {
      this.page = 1;
    },
    open(value) {
      if (value) {
        this.page = 1;
        this.showToTop = false;
        this.load();
        this.$nextTick(() => this.scrollTo(0));
        return;
      }
      clearTimeout(this.searchTimer);
      this.cancel();
    },
    refreshToken() {
      if (this.open) this.load();
    }
  },
  methods: {
    statusLabel(status) {
      return STATUS_LABELS[status] || status;
    },
    statusBadge(status) {
      return STATUS_BADGES[status] || 'tk-badge--neutral';
    },
    // El cuerpo se escribe en markdown; lo que sale se pasa por la lista blanca
    // de siempre, igual que el texto de una tarea.
    html(value) {
      return richTextHtml(markdownHtml(value));
    },
    // El extracto del plegado es el texto del contexto, sin sus etiquetas: el
    // recorte a tres líneas lo hace el estilo, no este método.
    excerpt(value) {
      const box = document.createElement('div');
      // Los bloques del markdown se separan al leerlos: sin esto, el final de
      // un párrafo y el principio del siguiente quedan pegados.
      box.innerHTML = this.html(value).replace(/<\/(p|li|ul|ol|h[1-6]|blockquote|pre|div)>/gi, ' ');
      return String(box.textContent || '').replace(/\s+/g, ' ').trim();
    },
    sectionsOf(detail) {
      return SECTIONS.filter((section) => String(detail[section.field] || '').trim());
    },
    isOpen(number) {
      return this.expanded.includes(number);
    },
    isEditing(number) {
      return this.editing === number;
    },
    async load() {
      const seq = ++this.loadSeq;
      this.listError = '';
      try {
        const adrs = await listADRs(this.search.trim());
        if (seq !== this.loadSeq) return;
        this.adrs = adrs;
        this.page = Math.min(this.page, this.pageCount);
      } catch (error) {
        if (seq !== this.loadSeq) return;
        this.listError = error?.message || 'No se pudieron leer las decisiones.';
      }
    },
    setPageSize(value) {
      this.pageSize = Number(value);
      this.page = 1;
    },
    goToPage(page) {
      const next = Math.min(Math.max(1, page), this.pageCount);
      if (next === this.page) return;
      this.page = next;
      this.showToTop = false;
      this.scrollTo(0);
    },
    scrollTo(top) {
      const scroller = this.$refs.scroller;
      if (!scroller) return;
      const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
      scroller.scrollTo({ top, behavior: reduce ? 'auto' : 'smooth' });
    },
    onScroll(event) {
      this.showToTop = event.target.scrollTop > TO_TOP_AFTER_PX;
    },
    scrollToTop() {
      this.scrollTo(0);
      this.showToTop = false;
    },
    async toggle(adr) {
      if (this.isEditing(adr.number)) return;
      if (this.isOpen(adr.number)) this.collapse(adr.number);
      else await this.expand(adr.number);
    },
    async expand(number) {
      if (!this.isOpen(number)) this.expanded = [...this.expanded, number];
      if (this.details[number]) return;
      try {
        const detail = await showADR(number);
        this.details = { ...this.details, [number]: detail };
      } catch (error) {
        this.listError = error?.message || 'No se pudo leer la decisión.';
        this.collapse(number);
      }
    },
    collapse(number) {
      if (this.isEditing(number)) return;
      this.expanded = this.expanded.filter((open) => open !== number);
      if (!this.details[number]) return;
      const details = { ...this.details };
      delete details[number];
      this.details = details;
    },
    async expandAll() {
      // Todo el registro filtrado, no solo la página que se ve: es lo que el
      // botón promete, y un ADR expandido no se pierde al cambiar de página.
      await Promise.all(this.visible.map((adr) => this.expand(adr.number)));
    },
    // Plegar todas respeta la tarjeta en edición: un clic de más no tira el
    // borrador.
    collapseAll() {
      this.expanded = this.expanded.filter((number) => number === this.editing);
      this.details = this.editing && this.details[this.editing] ? { [this.editing]: this.details[this.editing] } : {};
    },
    emptyForm() {
      return {
        title: '',
        deciders: '',
        status: 'proposed',
        context: '',
        decision: '',
        consequences: '',
        alternatives: '',
        note: ''
      };
    },
    startNew() {
      this.creating = true;
      this.editing = 0;
      this.draft = this.emptyForm();
      this.formError = '';
    },
    startEdit(number) {
      const detail = this.details[number];
      if (!detail) return;
      this.creating = false;
      this.editing = number;
      this.draft = {
        title: detail.title,
        deciders: detail.deciders,
        status: detail.status,
        supersededBy: detail.superseded_by || '',
        context: detail.context || '',
        decision: detail.decision || '',
        consequences: detail.consequences || '',
        alternatives: detail.alternatives || '',
        note: ''
      };
      this.formError = '';
    },
    cancel() {
      this.editing = 0;
      this.creating = false;
      this.draft = null;
      this.formError = '';
    },
    // Escape cancela lo que se está escribiendo y deja el segundo Escape para
    // cerrar la capa, que escucha en la ventana.
    handleEscape(event) {
      if (event.key !== 'Escape' || !this.open) return;
      if (!this.editing && !this.creating) return;
      event.stopImmediatePropagation();
      event.preventDefault();
      this.cancel();
    },
    // La página en la que cae un ADR del registro ya cargado: crear o saltar a
    // un sustituto no puede dejarlo fuera de la página que se está viendo.
    revealPage(number) {
      const index = this.visible.findIndex((adr) => adr.number === number);
      if (index >= 0) this.page = Math.floor(index / this.pageSize) + 1;
    },
    // El sustituto viaja como referencia: ADR-014. Se limpia el filtro y la
    // búsqueda para que su tarjeta exista, y se desplaza hasta ella.
    async openReference(reference) {
      const number = Number(String(reference || '').replace(/\D/g, ''));
      if (!number) return;
      this.search = '';
      this.statusFilter = '';
      this.page = 1;
      await this.load();
      this.revealPage(number);
      await this.expand(number);
      await this.$nextTick();
      document.getElementById(`adr-card-${number}`)?.scrollIntoView({ block: 'start' });
    },
    async save() {
      if (!this.draft || this.busy) return;
      this.busy = true;
      this.formError = '';
      try {
        if (this.creating) await this.create();
        else await this.edit();
      } catch (error) {
        this.formError = error?.message || 'No se pudo guardar la decisión.';
      } finally {
        this.busy = false;
      }
    },
    async create() {
      const created = await createADR({
        title: this.draft.title,
        deciders: this.draft.deciders,
        status: this.draft.status,
        context: this.draft.context,
        decision: this.draft.decision,
        consequences: this.draft.consequences,
        alternatives: this.draft.alternatives
      });
      this.creating = false;
      this.draft = null;
      // Quien acaba de crear quiere ver lo creado, aunque la búsqueda o el
      // filtro de antes lo escondieran.
      this.search = '';
      this.statusFilter = '';
      this.page = 1;
      await this.load();
      this.revealPage(created.number);
      await this.expand(created.number);
      await this.$nextTick();
      document.getElementById(`adr-card-${created.number}`)?.scrollIntoView({ block: 'start' });
    },
    async edit() {
      const detail = this.details[this.editing];
      if (!detail) {
        this.cancel();
        return;
      }
      const changes = {};
      if (!this.frozenBody) {
        for (const field of BODY_FIELDS) {
          if ((this.draft[field] || '') !== (detail[field] || '')) changes[field] = this.draft[field];
        }
      }
      if (this.draft.status !== detail.status) changes.status = this.draft.status;
      if (this.draft.status === 'superseded' && this.draft.supersededBy !== (detail.superseded_by || '')) {
        changes.superseded_by = this.draft.supersededBy;
      }
      if (this.draft.note.trim()) changes.note = this.draft.note;
      if (!Object.keys(changes).length) {
        this.cancel();
        return;
      }
      const updated = await updateADR(detail.number, changes);
      this.details = { ...this.details, [updated.number]: updated };
      this.editing = 0;
      this.draft = null;
      await this.load();
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
      <div class="decisions-header__actions"><button class="tk-button tk-button--secondary tk-button--compact"
          type="button" title="Nueva decisión" @click="startNew()"><span class="tk-icon" aria-hidden="true"><svg
            viewBox="0 0 24 24">
              <path d="M12 5v14M5 12h14" />
            </svg></span>Nueva decisión</button><button class="tk-icon-button" type="button"
          aria-label="Cerrar decisiones" title="Cerrar decisiones" @click="$emit('close')">×</button></div>
    </header>
    <div class="decisions-body" ref="scroller" @scroll="onScroll">
      <div class="decisions-stack">
        <div class="decisions-toolbar"><input class="tk-input" type="search" aria-label="Buscar decisiones"
            placeholder="Buscar en todas las decisiones" :value="search"
            @input="search = $event.target.value" /><select class="tk-select" aria-label="Filtrar por estado"
            :value="statusFilter" @change="statusFilter = $event.target.value">
            <option value="">Todos los estados</option>
            <option v-for="status in statuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
          </select><select class="tk-select tk-select--compact" aria-label="Decisiones por página" :value="pageSize"
            @change="setPageSize($event.target.value)">
            <option v-for="size in pageSizes" :key="size" :value="size">{{ size }} por página</option>
          </select></div>
        <div class="decisions-summary"><span>{{ rangeLabel }}</span>
          <div class="decisions-actions"><button class="tk-button tk-button--ghost" type="button"
              @click="expandAll()">Expandir todas</button><button class="tk-button tk-button--ghost" type="button"
              @click="collapseAll()">Plegar todas</button></div>
        </div>
        <p v-show="listError" class="decisions-error">{{ listError }}</p>

        <form v-if="creating" class="decisions-item is-open decisions-create" @submit.prevent="save()">
          <div class="decisions-form__bar"><span class="decisions-item__ref">Nueva decisión</span>
            <div class="decisions-actions"><button class="tk-button tk-button--ghost tk-button--compact" type="button"
                title="Descartar cambios" @click="cancel()"><span class="tk-icon" aria-hidden="true"><svg
                    viewBox="0 0 24 24">
                    <path d="M6 6l12 12M18 6 6 18" />
                  </svg></span>Cancelar</button><button class="tk-button tk-button--primary tk-button--compact"
                type="submit" title="Guardar decisión" :disabled="busy"><span class="tk-icon" aria-hidden="true"><svg
                    viewBox="0 0 24 24">
                    <path d="m5 12 4 4L19 6" />
                  </svg></span>Guardar</button></div>
          </div>
          <div class="form-grid"><label class="tk-field span-2"><span class="tk-field__label">Título</span><input
                class="tk-input" required :value="draft.title"
                @input="draft.title = $event.target.value" /></label><label class="tk-field"><span
                class="tk-field__label">Decisores</span><input class="tk-input" required :value="draft.deciders"
                @input="draft.deciders = $event.target.value" /></label><label class="tk-field"><span
                class="tk-field__label">Estado</span><select class="tk-select" :value="draft.status"
                @change="draft.status = $event.target.value">
                <option v-for="status in createStatuses" :key="status" :value="status">{{ statusLabel(status) }}
                </option>
              </select></label><label class="tk-field span-2"><span class="tk-field__label">Contexto</span><textarea
                class="tk-textarea" rows="4" :value="draft.context"
                @input="draft.context = $event.target.value"></textarea></label><label
              class="tk-field span-2"><span class="tk-field__label">Decisión</span><textarea class="tk-textarea"
                rows="4" :value="draft.decision" @input="draft.decision = $event.target.value"></textarea></label><label
              class="tk-field span-2"><span class="tk-field__label">Consecuencias</span><textarea class="tk-textarea"
                rows="4" :value="draft.consequences"
                @input="draft.consequences = $event.target.value"></textarea></label><label
              class="tk-field span-2"><span class="tk-field__label">Alternativas consideradas</span><textarea
                class="tk-textarea" rows="4" :value="draft.alternatives"
                @input="draft.alternatives = $event.target.value"></textarea></label></div>
          <p v-show="formError" class="decisions-error">{{ formError }}</p>
        </form>

        <article v-for="adr in paged" :id="'adr-card-' + adr.number" :key="adr.number" class="decisions-item"
          :class="{ 'is-open': isOpen(adr.number) }">
          <div class="decisions-item__head" role="button" :tabindex="isEditing(adr.number) ? -1 : 0"
            :aria-expanded="isOpen(adr.number)" :aria-controls="'adr-' + adr.number"
            :aria-disabled="isEditing(adr.number) ? 'true' : undefined"
            :aria-label="adr.ref + ': ' + adr.title" @click="toggle(adr)" @keydown.enter.prevent="toggle(adr)"
            @keydown.space.prevent="toggle(adr)">
            <div class="decisions-item__topline"><span class="decisions-item__meta"><span
                  class="decisions-item__ref">{{ adr.ref }}</span><span class="tk-badge"
                  :class="statusBadge(adr.status)">{{ statusLabel(adr.status) }}</span></span><span
                class="decisions-item__meta"><time class="decisions-item__date" :datetime="adr.date">{{ adr.date
                }}</time><span class="decisions-item__chev" :class="{ 'is-open': isOpen(adr.number) }"
                aria-hidden="true"><span class="tk-icon"><svg viewBox="0 0 24 24">
                    <path d="m6 9 6 6 6-6"></path>
                  </svg></span></span></span></div>
            <h3 class="decisions-item__title">{{ adr.title }}</h3>
            <p v-if="!isOpen(adr.number)" class="decisions-item__excerpt">{{ excerpt(adr.context) }}</p>
          </div>

          <div v-if="isOpen(adr.number)" :id="'adr-' + adr.number" class="decisions-content">
            <div class="decisions-separator"></div>
            <p v-if="!details[adr.number] && !isEditing(adr.number)" class="decisions-empty">Leyendo la decisión…</p>
            <form v-if="isEditing(adr.number) && details[adr.number]" class="decisions-form" @submit.prevent="save()">
              <div class="decisions-form__bar"><span class="decisions-item__ref">Editando {{ adr.ref }}</span>
                <div class="decisions-actions"><button class="tk-button tk-button--ghost tk-button--compact"
                    type="button" title="Descartar cambios" @click="cancel()"><span class="tk-icon"
                      aria-hidden="true"><svg viewBox="0 0 24 24">
                        <path d="M6 6l12 12M18 6 6 18" />
                      </svg></span>Cancelar</button><button class="tk-button tk-button--primary tk-button--compact"
                    type="submit" title="Guardar decisión" :disabled="busy"><span class="tk-icon"
                      aria-hidden="true"><svg viewBox="0 0 24 24">
                        <path d="m5 12 4 4L19 6" />
                      </svg></span>Guardar</button></div>
              </div>
              <div class="form-grid"><label class="tk-field span-2"><span class="tk-field__label">Título</span><input
                    class="tk-input" required :value="draft.title" :disabled="frozenBody"
                    @input="draft.title = $event.target.value" /></label><label class="tk-field"><span
                    class="tk-field__label">Decisores</span><input class="tk-input" disabled
                    :value="draft.deciders" /></label><label class="tk-field"><span
                    class="tk-field__label">Estado</span><select class="tk-select" :value="draft.status"
                    @change="draft.status = $event.target.value">
                    <option v-for="status in statuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
                  </select></label><label v-show="draft.status === 'superseded'" class="tk-field span-2"><span
                    class="tk-field__label">Sustituto</span><input class="tk-input" placeholder="ADR-017"
                    :value="draft.supersededBy" @input="draft.supersededBy = $event.target.value" /></label><label
                  class="tk-field span-2"><span class="tk-field__label">Contexto</span><textarea class="tk-textarea"
                    rows="4" :value="draft.context" :disabled="frozenBody"
                    @input="draft.context = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Decisión</span><textarea class="tk-textarea"
                    rows="4" :value="draft.decision" :disabled="frozenBody"
                    @input="draft.decision = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Consecuencias</span><textarea
                    class="tk-textarea" rows="4" :value="draft.consequences" :disabled="frozenBody"
                    @input="draft.consequences = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Alternativas consideradas</span><textarea
                    class="tk-textarea" rows="4" :value="draft.alternatives" :disabled="frozenBody"
                    @input="draft.alternatives = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Añadir a más información</span><textarea
                    class="tk-textarea" rows="3" :value="draft.note"
                    @input="draft.note = $event.target.value"></textarea></label></div>
              <p v-show="frozenBody" class="decisions-hint">El cuerpo está congelado: una decisión que ya no está
                propuesta solo admite añadidos en «Más información».</p>
              <p v-show="formError" class="decisions-error">{{ formError }}</p>
            </form>
            <div v-if="!isEditing(adr.number) && details[adr.number]" class="decisions-document">
              <div class="decisions-editbar">
                <p class="decisions-document__meta"><span>Decisores: {{ details[adr.number].deciders }}</span><span
                    v-if="details[adr.number].superseded_by">Sustituido por <button class="decisions-link"
                      type="button"
                      @click="openReference(details[adr.number].superseded_by)">{{ details[adr.number].superseded_by
                      }}</button></span></p><button class="tk-button tk-button--secondary tk-button--compact"
                  type="button" title="Editar decisión" @click="startEdit(adr.number)"><span class="tk-icon"
                    aria-hidden="true"><svg viewBox="0 0 24 24">
                      <path d="m4 20 4.5-1 11-11a2 2 0 0 0-3-3l-11 11L4 20Z" />
                    </svg></span>Editar</button>
              </div>
              <section v-for="section in sectionsOf(details[adr.number])" :key="section.field" class="decisions-section"
                :class="'decisions-section--' + section.field">
                <h4>{{ section.heading }}</h4>
                <div class="tk-rich-text" v-html="html(details[adr.number][section.field])"></div>
              </section>
            </div>
          </div>
        </article>

        <p v-if="!visible.length" class="decisions-empty">No hay decisiones que coincidan.</p>

        <nav v-if="pageCount > 1" class="decisions-pager" aria-label="Páginas de decisiones"><button
            class="tk-button tk-button--ghost" type="button" :disabled="page === 1"
            @click="goToPage(page - 1)">Anterior</button><span>Página {{ page }} de {{ pageCount }}</span><button
            class="tk-button tk-button--ghost" type="button" :disabled="page === pageCount"
            @click="goToPage(page + 1)">Siguiente</button></nav>
      </div>
    </div>
  </div>
  <button class="decisions-to-top" :class="{ 'is-visible': showToTop }" type="button" aria-label="Volver arriba"
    title="Volver arriba" @click="scrollToTop()"><span class="tk-icon"><svg viewBox="0 0 24 24">
        <path d="M6 15l6-6 6 6"></path>
      </svg></span></button>
</section>`
};
