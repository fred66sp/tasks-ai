(() => {
  // src/task-manager/state/inspector.js
  var inspectorState = {
    computed: {
      selectedBlockers() {
        return Array.isArray(this.detail?.blockers) ? this.detail.blockers : [];
      },
      selectedComments() {
        return Array.isArray(this.detail?.comments) ? this.detail.comments : [];
      },
      selectedDependents() {
        return Array.isArray(this.detail?.dependents) ? this.detail.dependents : [];
      },
      /**
       * Tareas que el formulario de edicion ofrece como padre. La tarea editada
       * no se ofrece a si misma: un ciclo mas profundo lo rechaza el servidor y su
       * error se presenta tal cual.
       *
       * @returns {Array<{ id: string, title: string }>}
       */
      parentOptions() {
        const ownId = this.selectedTask.id;
        return this.mappedTasks.filter((task) => task.id !== ownId).map((task) => ({ id: task.id, title: task.title }));
      },
      taskInspectorProps() {
        return {
          hasSelection: this.hasSelection,
          task: this.selectedTask,
          parentTask: this.selectedParentTask,
          menuOpen: this.isInspectorMenuOpen,
          busy: this.isBusy,
          statusBadgeClass: this.statusBadgeClass,
          statusLabel: this.statusLabel,
          priorityLabel: this.priorityLabel,
          subtasks: this.selectedSubtasks,
          parentOptions: this.parentOptions,
          blockers: this.selectedBlockers,
          dependents: this.selectedDependents,
          comments: this.selectedComments,
          editingCommentId: this.editingCommentId,
          editingCommentDraft: this.editingCommentDraft,
          tagCatalog: this.tagCatalog,
          colors: this.colors
        };
      }
    },
    methods: {
      /**
       * Traduce el unico canal de eventos de dominio del inspector a metodos del
       * workspace. Los `v-model` del componente no pasan por este despachador.
       *
       * @param {InspectorAction} [action]
       * @returns {*}
       */
      handleInspectorAction(action = {}) {
        const payload = action.payload || {};
        switch (action.type) {
          case "menu.toggle":
            this.isInspectorMenuOpen = !this.isInspectorMenuOpen;
            return;
          case "edit.open":
            return this.openEditTask();
          case "edit.cancel":
            return this.cancelTaskEdit();
          case "edit.save":
            return this.saveSelectedTask();
          case "task.copy":
            return this.copyTaskId();
          case "task.close":
            return this.closeTask();
          case "task.select":
            return this.loadTaskDetail(payload.id);
          case "task.update-field":
            return this.updateTaskField(payload.id, payload.field, payload.value);
          case "subtask.open":
            return this.openSubtaskDialog();
          case "dependency.add":
            return this.addDependency();
          case "dependency.remove-blocker":
            return this.removeDependency(payload.id);
          case "dependency.remove-dependent":
            return this.removeDependent(payload.id);
          case "comment.add":
            return this.addComment();
          case "comment.edit.start":
            return this.startCommentEdit(payload.comment);
          case "comment.edit.cancel":
            return this.cancelCommentEdit();
          case "comment.edit.save":
            return this.saveCommentEdit();
          default:
            console.warn(`[Gestor de tareas] Accion de inspector desconocida: ${action.type}`);
            return;
        }
      }
    }
  };

  // src/task-manager/api/tk-client.js
  var ApiError = class extends Error {
    /**
     * @param {string} message Mensaje seguro para presentar al usuario.
     * @param {boolean} unavailable El servidor no responde en este anfitrion.
     */
    constructor(message, unavailable = false) {
      super(message);
      this.name = "ApiError";
      this.unavailable = unavailable;
    }
  };
  async function apiRequest(path, options = {}) {
    let response;
    try {
      response = await fetch(path, options);
    } catch (error) {
      throw new ApiError(error?.message || "No se pudo conectar con el servidor.", true);
    }
    let data;
    try {
      data = await response.json();
    } catch {
      throw new ApiError("El servidor devolvi\xF3 una respuesta no v\xE1lida.");
    }
    if (!response.ok || data?.error) {
      throw new ApiError(data?.error || `La operaci\xF3n fall\xF3 (${response.status}).`);
    }
    return data;
  }
  function jsonRequest(path, method, body) {
    return apiRequest(path, {
      method,
      headers: body === void 0 ? void 0 : { "content-type": "application/json" },
      body: body === void 0 ? void 0 : JSON.stringify(body)
    });
  }
  async function listTasks() {
    const tasks = await apiRequest("/api/tasks");
    const refOf = new Map(tasks.map((task) => [task.id, task.ref]));
    return tasks.map((task) => task.parent_id == null || !refOf.has(task.parent_id) ? task : { ...task, parent_id: refOf.get(task.parent_id) });
  }
  var listTags = () => apiRequest("/api/tags");
  var listAllComments = () => apiRequest("/api/comments");
  var showTask = (ref) => apiRequest(`/api/tasks/${encodeURIComponent(ref)}`);
  var createTask = (task) => jsonRequest("/api/tasks", "POST", task);
  var updateTask = (ref, changes) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}`, "PATCH", changes);
  var closeTask = (ref) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/close`, "POST", { reason: "done" });
  var addTaskBlocker = (ref, blocker) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/blockers`, "POST", { blocker });
  var removeTaskBlocker = (ref, blocker) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/blockers/${encodeURIComponent(blocker)}`, "DELETE");
  var addTaskComment = (ref, body) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/comments`, "POST", { body });
  var editTaskComment = (ref, id, body) => jsonRequest(`/api/tasks/${encodeURIComponent(ref)}/comments/${encodeURIComponent(id)}`, "PATCH", { body });
  var createTag = (tag) => jsonRequest("/api/tags", "POST", tag);
  var updateTag = (name, tag) => jsonRequest(`/api/tags/${encodeURIComponent(name)}`, "PATCH", tag);
  var deleteTag = (name) => jsonRequest(`/api/tags/${encodeURIComponent(name)}`, "DELETE");
  var listADRs = (search = "") => apiRequest(`/api/adrs${search ? `?q=${encodeURIComponent(search)}` : ""}`);
  var showADR = (number) => apiRequest(`/api/adrs/${encodeURIComponent(number)}`);
  var createADR = (adr) => jsonRequest("/api/adrs", "POST", adr);
  var updateADR = (number, changes) => jsonRequest(`/api/adrs/${encodeURIComponent(number)}`, "PATCH", changes);

  // public/vendor/toast-anchor.mjs
  if (typeof document < "u" && !document.getElementById("__tk_css")) {
    let i = document.createElement("style");
    i.id = "__tk_css", i.textContent = "@keyframes __tk_spin{to{transform:rotate(360deg)}}", document.head.appendChild(i);
  }
  var g = { success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>', error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>', warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>', info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>', loading: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation:__tk_spin 1s linear infinite"><path stroke-linecap="round" stroke-linejoin="round" d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>' };
  var m = { position: "bottom-right", duration: 4e3, sound: false, dismissOnClick: false, showProgress: false, maxToasts: 5, theme: { success: { bg: "#10b981", light: "#d1fae5" }, error: { bg: "#ef4444", light: "#fee2e2" }, warning: { bg: "#f59e0b", light: "#fef3c7" }, info: { bg: "#3b82f6", light: "#dbeafe" } }, toast: { bg: "#ffffff", color: "#1e293b", descColor: "#64748b", borderColor: "#f1f5f9", borderWidth: "1px", borderRadius: "14px", shadow: "0 4px 24px rgba(0,0,0,0.08),0 1px 4px rgba(0,0,0,0.04)", padding: "13px 14px", width: "316px", fontSize: "13.5px", fontWeight: "600", descFontSize: "12px", descFontWeight: "400", showIcon: true, showClose: true, closeColor: "#cbd5e1", closeHoverColor: "#94a3b8", progressH: "3px", progressRadius: "0 0 14px 14px", iconSize: "32px", iconRadius: "50%", actionColor: null, actionFontSize: "12px" }, anchored: { position: "top", bg: "#0f172a", color: "#ffffff", borderColor: "transparent", borderWidth: "0px", borderRadius: "10px", shadow: "0 8px 24px rgba(0,0,0,0.2)", padding: "8px 14px", fontSize: "13px", fontWeight: "600", showIcon: true, arrowSize: 10, showArrow: true, gap: 13 }, anchoredConfirm: { position: "top", bg: "#0f172a", color: "#f8fafc", borderColor: "transparent", borderWidth: "0px", borderRadius: "12px", shadow: "0 8px 28px rgba(0,0,0,0.22)", padding: "12px 14px", fontSize: "13px", minWidth: "160px", showIcon: true, confirmBg: "#ef4444", confirmColor: "#ffffff", confirmHoverBg: "#dc2626", cancelBg: "#1e293b", cancelColor: "#94a3b8", cancelHoverBg: "#334155", cancelBorder: "#334155", btnRadius: "8px", btnFontSize: "12px", btnFontWeight: "700", btnPadding: "6px 0", arrowSize: 10, showArrow: true, gap: 13 }, modal: { position: "center", overlayBg: "rgba(15,23,42,0.55)", overlayBlur: "5px", bg: "#ffffff", borderColor: "transparent", borderWidth: "0px", borderRadius: "20px", shadow: "0 25px 60px rgba(0,0,0,0.22)", maxWidth: "420px", padding: "24px", footerBg: "#f8fafc", footerBorder: "#f1f5f9", footerPaddingV: "14px", footerPaddingVB: "18px", titleColor: "#0f172a", titleSize: "17px", titleWeight: "700", titleMargin: "0 0 8px", messageColor: "#64748b", messageSize: "14px", iconSize: "48px", iconRadius: "50%", iconBg: null, iconColor: null, confirmBg: null, confirmColor: "#ffffff", confirmHoverBg: null, confirmRadius: "10px", confirmSize: "13.5px", confirmWeight: "600", confirmPadding: "9px 22px", cancelColor: "#475569", cancelHoverBg: "#f1f5f9", cancelRadius: "10px", cancelSize: "13.5px", cancelWeight: "500", cancelPadding: "9px 18px", cancelBg: "transparent", cancelBorder: "none" } };
  function Y(i = {}) {
    let n = ["toast", "anchored", "anchoredConfirm", "modal", "theme"];
    for (let [r, o] of Object.entries(i)) if (n.includes(r) && o && typeof o == "object") if (r === "theme") for (let [t, e] of Object.entries(o)) m.theme[t] = { ...m.theme[t], ...e };
    else Object.assign(m[r], o);
    else m[r] = o;
  }
  var k = null;
  var te = { success: { wave: "sine", sf: 600, ef: 900, gs: 0.15, ge: 0.01, dur: 0.15 }, error: { wave: "sawtooth", sf: 400, ef: 200, gs: 0.15, ge: 0.01, dur: 0.15 }, warning: { wave: "triangle", sf: 600, ef: 380, gs: 0.12, ge: 0.01, dur: 0.13 }, info: { wave: "sine", sf: 800, ef: 300, gs: 0.15, ge: 0.01, dur: 0.1 }, pop: { wave: "sine", sf: 700, ef: 350, gs: 0.12, ge: 0.01, dur: 0.1 } };
  function C(i = "success") {
    if (!(typeof window > "u")) try {
      k || (k = new (window.AudioContext || window.webkitAudioContext)()), k.state === "suspended" && k.resume();
      let n = te[i] || te.pop, r = k.createOscillator(), o = k.createGain(), t = k.currentTime;
      r.type = n.wave, r.frequency.setValueAtTime(n.sf, t), r.frequency.exponentialRampToValueAtTime(n.ef, t + n.dur), o.gain.setValueAtTime(n.gs, t), o.gain.exponentialRampToValueAtTime(n.ge, t + n.dur), r.connect(o), o.connect(k.destination), r.start(), r.stop(t + n.dur);
    } catch {
    }
  }
  var re = { "top-left": "top:1.5rem;left:1.5rem;flex-direction:column", "top-center": "top:1.5rem;left:50%;transform:translateX(-50%);flex-direction:column", "top-right": "top:1.5rem;right:1.5rem;flex-direction:column", "middle-left": "top:50%;left:1.5rem;transform:translateY(-50%);flex-direction:column", "middle-center": "top:50%;left:50%;transform:translate(-50%,-50%);flex-direction:column", "middle-right": "top:50%;right:1.5rem;transform:translateY(-50%);flex-direction:column", "bottom-left": "bottom:1.5rem;left:1.5rem;flex-direction:column-reverse", "bottom-center": "bottom:1.5rem;left:50%;transform:translateX(-50%);flex-direction:column-reverse", "bottom-right": "bottom:1.5rem;right:1.5rem;flex-direction:column-reverse" };
  var D = /* @__PURE__ */ new Map();
  var z = /* @__PURE__ */ new Map();
  var he = 0;
  function be(i) {
    if (typeof document > "u") return null;
    if (D.has(i)) return D.get(i);
    let n = document.createElement("div");
    return n.id = `__tk_c_${i.replace(/-/g, "_")}`, n.setAttribute("style", "position:fixed;z-index:9999;display:flex;gap:10px;pointer-events:none;" + (re[i] || re["bottom-right"])), document.body.appendChild(n), D.set(i, n), n;
  }
  function x(i) {
    let n = z.get(i);
    if (!n) return;
    clearTimeout(n.timer);
    let r = n.el;
    r.style.opacity = "0", r.style.transform = "scale(0.9) translateY(6px)", r.style.maxHeight = r.offsetHeight + "px", setTimeout(() => {
      r.style.transition = "all 0.32s ease-out", r.style.maxHeight = "0", r.style.padding = "0", r.style.margin = "0", r.style.overflow = "hidden";
    }, 60), setTimeout(() => {
      r.remove(), z.delete(i);
    }, 420);
  }
  function L() {
    [...z.keys()].forEach(x);
  }
  function xe() {
    return `__tk_${++he}`;
  }
  function u(i, n = {}) {
    var Z, ee;
    if (typeof document > "u") return () => {
    };
    let r = m, o = r.toast, t = { type: "success", position: r.position, duration: r.duration, sound: r.sound, dismissOnClick: r.dismissOnClick, showProgress: r.showProgress, description: null, icon: null, action: null, id: null, showIcon: void 0, showClose: void 0, ...n }, e = t.type, d = r.theme[e] || r.theme.success, a = (y) => t[y] !== void 0 ? t[y] : o[y], c = a("bg"), s = a("color"), l = a("descColor"), f = a("borderColor"), p = a("borderWidth"), P = a("borderRadius"), H = a("shadow"), F = a("padding"), O = a("width"), h = a("fontSize"), M = a("fontWeight"), $ = a("descFontSize"), S = a("descFontWeight"), w = a("closeColor"), B = a("closeHoverColor"), v = a("progressH"), ne = a("progressRadius"), ie = t.progressColor || d.bg, se = t.iconBg || d.light, ae = t.iconColor || d.bg, q = a("iconSize"), de = a("iconRadius"), ce = t.actionColor || d.bg, le = a("actionFontSize"), fe = t.icon || g[e] || g.success, j = t.id || xe(), pe = t.showIcon !== void 0 ? t.showIcon : o.showIcon !== void 0 ? o.showIcon : true, me = t.showClose !== void 0 ? t.showClose : o.showClose !== void 0 ? o.showClose : true, J = !!(t.description || t.action), ue = J ? "align-items:flex-start" : "align-items:center", ge = J ? "padding-top:2px" : "";
    t.sound && C(e);
    let Q = be(t.position);
    if (!Q) return () => {
    };
    z.size >= r.maxToasts && x(z.keys().next().value);
    let b = document.createElement("div");
    b.id = j, b.setAttribute("style", ["pointer-events:auto", "position:relative", "overflow:hidden", "display:flex", ue, "gap:12px", `width:${O}`, `background:${c}`, `border:${p} solid ${f}`, `padding:${F}`, `border-radius:${P}`, `box-shadow:${H}`, "transition:all 0.38s cubic-bezier(0.34,1.56,0.64,1)", "opacity:0", "transform:scale(0.9) translateY(10px)", `cursor:${t.dismissOnClick ? "pointer" : "default"}`, "box-sizing:border-box"].join(";")), b.innerHTML = `
    ${pe ? `<div style="display:flex;align-items:center;justify-content:center;width:${q};height:${q};min-width:${q};border-radius:${de};background:${se};color:${ae};flex-shrink:0">
      <div style="width:16px;height:16px;display:flex">${fe}</div>
    </div>` : ""}
    <div style="flex:1;min-width:0;${ge}">
      <p style="font-size:${h};font-weight:${M};color:${s};line-height:1.4;margin:0">${i}</p>
      ${t.description ? `<p style="font-size:${$};font-weight:${S};color:${l};margin:3px 0 0;line-height:1.5">${t.description}</p>` : ""}
      ${t.action ? `<button data-tk-action style="margin-top:7px;font-size:${le};font-weight:700;color:${ce};background:none;border:none;cursor:pointer;padding:0;display:block;line-height:1">${t.action.label}</button>` : ""}
    </div>
    ${me ? `<button data-tk-close
      style="flex-shrink:0;background:none;border:none;cursor:pointer;color:${w};padding:2px;line-height:0;border-radius:4px;transition:color 0.15s"
      title="Dismiss"
      onmouseenter="this.style.color='${B}'"
      onmouseleave="this.style.color='${w}'"
    >
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/>
      </svg>
    </button>` : ""}
    ${t.showProgress && t.duration > 0 ? `<div data-tk-bar style="position:absolute;bottom:0;left:0;height:${v};border-radius:${ne};background:${ie};width:100%;transition:width ${t.duration}ms linear"></div>` : ""}
  `.trim(), Q.appendChild(b), requestAnimationFrame(() => requestAnimationFrame(() => {
      if (b.style.opacity = "1", b.style.transform = "scale(1) translateY(0)", t.showProgress && t.duration > 0) {
        let y = b.querySelector("[data-tk-bar]");
        y && (y.style.width = "0%");
      }
    }));
    let T = () => x(j), I = null;
    return t.duration > 0 && (I = setTimeout(T, t.duration)), b.addEventListener("mouseenter", () => clearTimeout(I)), b.addEventListener("mouseleave", () => {
      t.duration > 0 && (I = setTimeout(T, Math.min(t.duration, 1500)));
    }), (Z = b.querySelector("[data-tk-close]")) == null || Z.addEventListener("click", (y) => {
      y.stopPropagation(), T();
    }), t.dismissOnClick && b.addEventListener("click", T), (ee = b.querySelector("[data-tk-action]")) == null || ee.addEventListener("click", (y) => {
      var R, oe;
      y.stopPropagation(), (oe = (R = t.action) == null ? void 0 : R.onClick) == null || oe.call(R), T();
    }), z.set(j, { el: b, timer: I }), T;
  }
  var ye = 0;
  function E(i, n = {}, r = {}) {
    let { loading: o = "Loading\u2026", success: t = "Done!", error: e = "Something went wrong" } = n, d = `__tk_p${++ye}`;
    return u(o, { type: "info", duration: 0, icon: g.loading, sound: false, id: d, ...r }), Promise.resolve(i).then((a) => {
      x(d);
      let c = typeof t == "function" ? t(a) : t;
      u(c, { type: "success", ...r });
    }).catch((a) => {
      x(d);
      let c = typeof e == "function" ? e(a) : e;
      u(c, { type: "error", ...r });
    }), () => x(d);
  }
  function V(i, n, r, o) {
    let t = n.getBoundingClientRect(), e = i.getBoundingClientRect(), d = window.scrollX, a = window.scrollY, c, s;
    switch (r) {
      case "bottom":
        c = t.bottom + a + o, s = t.left + d + t.width / 2 - e.width / 2;
        break;
      case "left":
        c = t.top + a + t.height / 2 - e.height / 2, s = t.left + d - e.width - o;
        break;
      case "right":
        c = t.top + a + t.height / 2 - e.height / 2, s = t.right + d + o;
        break;
      default:
        c = t.top + a - e.height - o, s = t.left + d + t.width / 2 - e.width / 2;
    }
    i.style.top = `${c}px`, i.style.left = `${s}px`;
  }
  function U(i, n, r, o, t) {
    let e = n / 2, d = t && t !== "0px" ? t : "0px", a = o || "transparent", c = d !== "0px", s = "";
    if (c) {
      let f = `${d} solid ${a}`, p = `${d} solid transparent`;
      switch (i) {
        case "top":
          s = `border-top:${p};border-right:${f};border-bottom:${f};border-left:${p};`;
          break;
        case "bottom":
          s = `border-top:${f};border-right:${p};border-bottom:${p};border-left:${f};`;
          break;
        case "left":
          s = `border-top:${f};border-right:${f};border-bottom:${p};border-left:${p};`;
          break;
        case "right":
          s = `border-top:${p};border-right:${p};border-bottom:${f};border-left:${f};`;
          break;
      }
    }
    let l = `${s}position:absolute;width:${n}px;height:${n}px;background:${r};transform:rotate(45deg);`;
    switch (i) {
      case "bottom":
        return `${l}top:${-e}px;left:50%;margin-left:${-e}px`;
      case "left":
        return `${l}top:50%;right:${-e}px;margin-top:${-e}px`;
      case "right":
        return `${l}top:50%;left:${-e}px;margin-top:${-e}px`;
      default:
        return `${l}bottom:${-e}px;left:50%;margin-left:${-e}px`;
    }
  }
  function X(i) {
    switch (i) {
      case "bottom":
        return "scale(0.88) translateY(-8px)";
      case "left":
        return "scale(0.88) translateX(8px)";
      case "right":
        return "scale(0.88) translateX(-8px)";
      default:
        return "scale(0.88) translateY(8px)";
    }
  }
  function K(i) {
    switch (i) {
      case "bottom":
        return "scale(0.9) translateY(-4px)";
      case "left":
        return "scale(0.9) translateX(4px)";
      case "right":
        return "scale(0.9) translateX(-4px)";
      default:
        return "scale(0.9) translateY(4px)";
    }
  }
  var N = /* @__PURE__ */ new WeakMap();
  function A(i, n, r = {}) {
    if (typeof document > "u" || !n) return () => {
    };
    let o = { type: "success", duration: 2500, sound: m.sound, icon: null, ...m.anchored, ...r }, t = m.theme[o.type] || m.theme.success, e = o.icon || g[o.type], d = o.position;
    o.sound && C(o.type);
    let a = N.get(n);
    a && a();
    let c = o.showIcon !== void 0 ? o.showIcon : true, s = document.createElement("div");
    s.setAttribute("style", ["position:absolute", "z-index:9998", `padding:${o.padding}`, `background:${o.bg}`, `color:${o.color}`, `font-size:${o.fontSize}`, `font-weight:${o.fontWeight}`, `border-radius:${o.borderRadius}`, `border:${o.borderWidth} solid ${o.borderColor}`, `box-shadow:${o.shadow}`, "pointer-events:none", "white-space:nowrap", "transition:opacity 0.3s ease,transform 0.3s cubic-bezier(0.34,1.56,0.64,1)", "opacity:0", `transform:${X(d)}`].join(";")), s.innerHTML = `
    <div style="display:flex;align-items:center;gap:7px">
      ${c ? `<div style="width:15px;height:15px;display:flex;flex-shrink:0;color:${t.bg}">${e}</div>` : ""}
      <span>${i}</span>
    </div>
    ${o.showArrow ? `<div style="${U(d, o.arrowSize, o.bg, o.borderColor, o.borderWidth)}"></div>` : ""}
  `.trim(), document.body.appendChild(s), V(s, n, d, o.gap), requestAnimationFrame(() => requestAnimationFrame(() => {
      s.style.opacity = "1", s.style.transform = "scale(1) translate(0,0)";
    }));
    let l = () => {
      N.delete(n), s.style.opacity = "0", s.style.transform = K(d), setTimeout(() => s.remove(), 300);
    };
    return N.set(n, l), o.duration > 0 && setTimeout(l, o.duration), l;
  }
  var G = /* @__PURE__ */ new WeakMap();
  function _(i, n, r, o, t = {}) {
    if (typeof document > "u" || !n) return;
    let e = { confirmLabel: "Yes", cancelLabel: "No", sound: m.sound, ...m.anchoredConfirm, ...t }, d = e.position;
    e.sound && C("warning");
    let a = G.get(n);
    a && a();
    let c = e.showIcon !== void 0 ? e.showIcon : true, s = document.createElement("div");
    s.setAttribute("style", ["position:absolute", "z-index:9998", `padding:${e.padding}`, `background:${e.bg}`, `color:${e.color}`, `font-size:${e.fontSize}`, `border-radius:${e.borderRadius}`, `border:${e.borderWidth} solid ${e.borderColor}`, `box-shadow:${e.shadow}`, `min-width:${e.minWidth}`, "pointer-events:auto", "transition:opacity 0.3s ease,transform 0.3s cubic-bezier(0.34,1.56,0.64,1)", "opacity:0", `transform:${X(d)}`].join(";")), s.innerHTML = `
    <div style="display:flex;align-items:center;gap:8px;margin-bottom:10px">
      ${c ? `<div style="width:15px;height:15px;flex-shrink:0;color:#fca5a5">${g.warning}</div>` : ""}
      <span>${i}</span>
    </div>
    <div style="display:flex;gap:7px">
      <button data-cancel style="
        flex:1;padding:${e.btnPadding};
        background:${e.cancelBg};
        border:1px solid ${e.cancelBorder};
        color:${e.cancelColor};
        border-radius:${e.btnRadius};
        cursor:pointer;
        font-weight:${e.btnFontWeight};
        font-size:${e.btnFontSize};
        transition:background 0.15s,color 0.15s;
      ">${e.cancelLabel}</button>
      <button data-confirm style="
        flex:1;padding:${e.btnPadding};
        background:${e.confirmBg};
        border:none;
        color:${e.confirmColor};
        border-radius:${e.btnRadius};
        cursor:pointer;
        font-weight:${e.btnFontWeight};
        font-size:${e.btnFontSize};
        transition:background 0.15s;
      ">${e.confirmLabel}</button>
    </div>
    ${e.showArrow ? `<div style="${U(d, e.arrowSize, e.bg, e.borderColor, e.borderWidth)}"></div>` : ""}
  `.trim(), document.body.appendChild(s), V(s, n, d, e.gap), requestAnimationFrame(() => requestAnimationFrame(() => {
      s.style.opacity = "1", s.style.transform = "scale(1) translate(0,0)";
    }));
    let l = () => {
      G.delete(n), s.style.opacity = "0", s.style.transform = K(d), setTimeout(() => s.remove(), 280);
    };
    G.set(n, l);
    let f = s.querySelector("[data-confirm]"), p = s.querySelector("[data-cancel]");
    f.addEventListener("mouseenter", () => {
      f.style.background = e.confirmHoverBg;
    }), f.addEventListener("mouseleave", () => {
      f.style.background = e.confirmBg;
    }), p.addEventListener("mouseenter", () => {
      p.style.background = e.cancelHoverBg;
    }), p.addEventListener("mouseleave", () => {
      p.style.background = e.cancelBg;
    }), f.onclick = () => {
      l(), r == null || r();
    }, p.onclick = () => {
      l(), o == null || o();
    };
  }
  function W(i, n, r = {}) {
    return new Promise((o) => {
      if (typeof document > "u") return o(false);
      let e = { ...m.modal, confirmLabel: "Confirm", cancelLabel: "Cancel", confirmVariant: "danger", icon: null, sound: m.sound, ...r }, d = e.position === "top", a = d ? "flex-start" : "center", c = d ? "padding:5vh 16px 16px" : "padding:16px";
      e.sound && C("warning");
      let s = e.confirmVariant !== "primary", l = e.confirmBg || (s ? "#ef4444" : "#3b82f6"), f = e.confirmHoverBg || (s ? "#dc2626" : "#2563eb"), p = e.iconColor || (s ? "#ef4444" : "#3b82f6"), P = e.iconBg || (s ? "#fee2e2" : "#dbeafe"), H = e.icon || g.warning, F = e.padding.trim().split(/\s+/), O = F[1] || F[0], h = document.createElement("div");
      h.setAttribute("style", ["position:fixed", "inset:0", "z-index:99999", "display:flex", `align-items:${a}`, "justify-content:center", c, `background:${e.overlayBg}`, `backdrop-filter:blur(${e.overlayBlur})`, "-webkit-backdrop-filter:blur(" + e.overlayBlur + ")", "transition:opacity 0.25s ease", "opacity:0"].join(";")), h.innerHTML = `
      <div data-mb style="
        width:100%;max-width:${e.maxWidth};
        background:${e.bg};
        border-radius:${e.borderRadius};
        border:${e.borderWidth} solid ${e.borderColor};
        box-shadow:${e.shadow};
        overflow:hidden;
        transition:all 0.32s cubic-bezier(0.34,1.56,0.64,1);
        transform:scale(0.9) translateY(18px);
      ">
        <!-- Body -->
        <div style="padding:${e.padding}">
          <div style="
            width:${e.iconSize};height:${e.iconSize};
            border-radius:${e.iconRadius};
            background:${P};
            display:flex;align-items:center;justify-content:center;
            margin-bottom:16px;
            color:${p};
          ">
            <div style="width:24px;height:24px">${H}</div>
          </div>
          <h3 style="
            font-size:${e.titleSize};
            font-weight:${e.titleWeight};
            color:${e.titleColor};
            margin:${e.titleMargin};
            line-height:1.3;
          ">${i}</h3>
          <p style="
            font-size:${e.messageSize};
            color:${e.messageColor};
            margin:0;
            line-height:1.65;
          ">${n}</p>
        </div>

        <!-- Footer -->
        <div style="
          background:${e.footerBg};
          padding:${e.footerPaddingV} ${O} ${e.footerPaddingVB};
          display:flex;gap:10px;justify-content:flex-end;
          border-top:1px solid ${e.footerBorder};
        ">
          <button data-mc style="
            padding:${e.cancelPadding};
            font-size:${e.cancelSize};
            font-weight:${e.cancelWeight};
            color:${e.cancelColor};
            background:${e.cancelBg};
            border:${e.cancelBorder};
            border-radius:${e.cancelRadius};
            cursor:pointer;
            transition:background 0.15s, color 0.15s;
          ">${e.cancelLabel}</button>
          <button data-mok style="
            padding:${e.confirmPadding};
            font-size:${e.confirmSize};
            font-weight:${e.confirmWeight};
            color:${e.confirmColor};
            background:${l};
            border:none;
            border-radius:${e.confirmRadius};
            cursor:pointer;
            box-shadow:0 2px 8px ${l}55;
            transition:background 0.15s, transform 0.1s;
          ">${e.confirmLabel}</button>
        </div>
      </div>
    `.trim(), document.body.appendChild(h);
      let M = h.querySelector("[data-mb]"), $ = h.querySelector("[data-mok]"), S = h.querySelector("[data-mc]");
      $.addEventListener("mouseenter", () => {
        $.style.background = f, $.style.transform = "translateY(-1px)";
      }), $.addEventListener("mouseleave", () => {
        $.style.background = l, $.style.transform = "";
      }), S.addEventListener("mouseenter", () => {
        S.style.background = e.cancelHoverBg;
      }), S.addEventListener("mouseleave", () => {
        S.style.background = e.cancelBg;
      }), requestAnimationFrame(() => {
        h.style.opacity = "1", M.style.transform = "scale(1) translateY(0)";
      });
      let w = (v) => {
        document.removeEventListener("keydown", B), h.style.opacity = "0", M.style.transform = "scale(0.9) translateY(16px)", setTimeout(() => {
          h.remove(), o(v);
        }, 280);
      };
      S.onclick = () => w(false), $.onclick = () => w(true), h.addEventListener("click", (v) => {
        v.target === h && w(false);
      });
      let B = (v) => {
        v.key === "Escape" && (w(false), document.removeEventListener("keydown", B)), v.key === "Enter" && (w(true), document.removeEventListener("keydown", B));
      };
      document.addEventListener("keydown", B);
    });
  }
  var we = { success: (i, n) => u(i, { type: "success", ...n }), error: (i, n) => u(i, { type: "error", ...n }), warning: (i, n) => u(i, { type: "warning", ...n }), info: (i, n) => u(i, { type: "info", ...n }), notify: u, promise: E, anchored: A, anchoredConfirm: _, modal: W, dismiss: x, dismissAll: L, configure: Y, ICONS: g };
  var ao = we;

  // src/task-manager/notifications/toast-anchor.js
  var DARK_STUDIO = {
    position: "bottom-right",
    toast: {
      bg: "#121c27",
      color: "#f3f6f8",
      descColor: "#bac4ce",
      borderColor: "#344657",
      borderWidth: "1px",
      borderRadius: "8px",
      shadow: "0 30px 90px rgba(0, 0, 0, 0.52)",
      width: "min(400px, calc(100vw - 36px))"
    },
    modal: {
      overlayBg: "rgba(10, 16, 23, 0.78)",
      overlayBlur: "5px",
      bg: "#182431",
      borderColor: "#344657",
      borderWidth: "1px",
      borderRadius: "8px",
      shadow: "0 30px 90px rgba(0, 0, 0, 0.62)",
      titleColor: "#f3f6f8",
      messageColor: "#bac4ce",
      footerBg: "#121c27",
      footerBorder: "#253342",
      cancelColor: "#bac4ce",
      cancelHoverBg: "#1b2937"
    },
    theme: {
      success: { bg: "#58d392", light: "rgba(88, 211, 146, 0.16)" },
      error: { bg: "#ff7373", light: "rgba(255, 115, 115, 0.16)" },
      warning: { bg: "#f2bf59", light: "rgba(242, 191, 89, 0.16)" },
      info: { bg: "#62a7ff", light: "rgba(98, 167, 255, 0.16)" }
    }
  };
  ao.configure(DARK_STUDIO);
  function escapeHtml(value) {
    return String(value ?? "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  var openConfirms = 0;
  function isConfirmOpen() {
    return openConfirms > 0;
  }
  function notifySuccess(message) {
    return ao.success(escapeHtml(message));
  }
  function notifyError(message) {
    return ao.error(escapeHtml(message));
  }
  function notifyInfo(message) {
    return ao.info(escapeHtml(message));
  }
  function confirmAction(title, message, options = {}) {
    openConfirms += 1;
    return ao.modal(escapeHtml(title), escapeHtml(message), {
      confirmLabel: "Eliminar",
      cancelLabel: "Cancelar",
      confirmVariant: "danger",
      ...options
    }).finally(() => {
      openConfirms -= 1;
    });
  }

  // src/task-manager/state/tag-actions.js
  var tagActions = {
    computed: {
      tagCatalog() {
        return [...this.tags].sort((a, b) => a.name.localeCompare(b.name, "es"));
      },
      visibleTags() {
        const query = this.normalizeSearchText(this.tagSearchQuery);
        return this.tagCatalog.filter((tag) => !query || this.normalizeSearchText(`${tag.name} ${tag.description || ""}`).includes(query));
      }
    },
    methods: {
      tagColor(name) {
        const colorId = this.tags.find((tag) => tag.name === name)?.color || "blue";
        return this.colors.find((color) => color.id === colorId)?.hex || this.colors[0].hex;
      },
      colorHex(color) {
        return this.colors.find((item) => item.id === color)?.hex || this.colors[0].hex;
      },
      openCreateTag() {
        this.tagFormMode = "create";
        this.editingTagName = "";
        this.tagDraft = this.emptyTagDraft();
        this.tagFormError = "";
      },
      openEditTag(tag) {
        this.tagFormMode = "edit";
        this.editingTagName = tag.name;
        this.tagDraft = { name: tag.name, description: tag.description || "", color: tag.color || "blue" };
        this.tagFormError = "";
      },
      closeTagForm() {
        this.tagFormMode = "";
        this.editingTagName = "";
        this.tagFormError = "";
      },
      async saveTag() {
        const name = this.tagDraft.name.trim().replace(/\s+/g, " ");
        if (!name) {
          this.tagFormError = "El nombre es obligatorio.";
          return;
        }
        const original = this.editingTagName;
        const key = `tag-save:${original || name}`;
        const tag = {
          name,
          description: this.tagDraft.description.trim(),
          color: this.tagDraft.color
        };
        const renombrada = Boolean(original) && original !== name;
        const filtros = { advancedTags: this.advancedTags, quickTagFilter: this.quickTagFilter };
        if (renombrada) {
          this.advancedTags = this.advancedTags.map((item) => item === original ? name : item);
          if (this.quickTagFilter === original) this.quickTagFilter = name;
        }
        const result = await this.mutate(key, () => original ? updateTag(original, tag) : createTag(tag), original ? "Etiqueta actualizada." : "Etiqueta creada.");
        if (result) {
          this.closeTagForm();
        } else if (renombrada) {
          this.advancedTags = filtros.advancedTags;
          this.quickTagFilter = filtros.quickTagFilter;
        }
      },
      async requestDeleteTag(tag) {
        if (!tag) return;
        const affectedCount = this.mappedTasks.filter((task) => task.chips.some((name) => this.sameTagName(name, tag.name))).length;
        const confirmed = await confirmAction(
          "Eliminar etiqueta",
          `Vas a eliminar \u201C${tag.name}\u201D de ${affectedCount} ${affectedCount === 1 ? "tarea" : "tareas"}.`
        );
        if (confirmed) await this.deleteTag(tag);
      },
      async deleteTag(tag) {
        if (!tag) return;
        const result = await this.mutate(`tag-delete:${tag.name}`, () => deleteTag(tag.name), "Etiqueta eliminada.");
        if (result) {
          this.advancedTags = this.advancedTags.filter((name) => name !== tag.name);
          if (this.quickTagFilter === tag.name) this.quickTagFilter = "";
        }
      }
    }
  };

  // src/shared/tasks/task-model.js
  var STATUS_LABELS = {
    open: "Pendiente",
    in_progress: "En curso",
    done: "Hecha",
    blocked: "Bloqueada"
  };
  var PRIORITY_LABELS = ["Cr\xEDtica", "Alta", "Media", "Baja"];
  var statusLabel = (status) => STATUS_LABELS[status] || status || "Sin estado";
  var priorityLabel = (priority) => PRIORITY_LABELS[Number(priority)] || "Sin prioridad";
  var statusBadgeClass = (status) => ({
    open: "tk-badge--warning",
    in_progress: "tk-badge--info",
    done: "tk-badge--success",
    blocked: "tk-badge--danger"
  })[status] || "tk-badge--neutral";
  var TASK_KINDS = {
    epic: { type: "\xC9pica", icon: "E" },
    subtask: { type: "Subtarea", icon: "S" },
    task: { type: "Tarea", icon: "T" }
  };
  var hasSubtasks = (task) => Array.isArray(task.children) ? task.children.length > 0 : Boolean(task.children);
  function mapTask(task = {}, subtasks = hasSubtasks(task)) {
    const chips = Array.isArray(task.tags) ? task.tags : Array.isArray(task.chips) ? task.chips : [];
    const parentId = task.parent_id || task.parentId || null;
    const blocked = Boolean(task.blocked) && task.status !== "done";
    const epic = subtasks || chips.some((tag) => tag.toLocaleLowerCase("es") === "epic");
    const kind = epic ? "epic" : parentId ? "subtask" : "task";
    const { parent_id: _parentId, tags: _tags, blockers = [], dependents = [], comments = [], ...rest } = task;
    return {
      ...rest,
      id: task.ref || task.id || "",
      title: task.title || "",
      description: task.description || "",
      priority: String(task.priority ?? 2),
      status: blocked ? "blocked" : task.status || "open",
      parentId,
      chips,
      kind,
      ...TASK_KINDS[kind],
      blockers: blockers.map((blocker) => mapRelation(blocker)),
      dependents: dependents.map((dependent) => mapRelation(dependent)),
      comments: comments.map(mapComment)
    };
  }
  var COLD_TASK_DAYS = 7;
  var DAY_MS = 24 * 60 * 60 * 1e3;
  function coldTasks(tasks = [], comments = [], now = Date.now(), thresholdDays = COLD_TASK_DAYS) {
    const lastCommentAt = /* @__PURE__ */ new Map();
    for (const comment of comments) {
      const timestamp = Date.parse(comment?.created_at);
      if (!comment?.ref || !Number.isFinite(timestamp)) continue;
      if (timestamp > (lastCommentAt.get(comment.ref) ?? -Infinity)) lastCommentAt.set(comment.ref, timestamp);
    }
    const threshold = thresholdDays * DAY_MS;
    return tasks.filter((task) => task.status === "open" || task.status === "in_progress").map((task) => {
      const updatedAt = Date.parse(task.updated_at);
      const activity = Math.max(
        Number.isFinite(updatedAt) ? updatedAt : -Infinity,
        lastCommentAt.get(task.id) ?? -Infinity
      );
      if (!Number.isFinite(activity) || now - activity < threshold) return null;
      return { id: task.id, title: task.title, days: Math.floor((now - activity) / DAY_MS) };
    }).filter(Boolean).sort((left, right) => right.days - left.days);
  }
  function mapRelation(relation) {
    if (typeof relation === "string") return { id: relation, title: "" };
    return { id: relation?.ref || relation?.id || "", title: relation?.title || "" };
  }
  function mapComment(comment) {
    if (typeof comment === "string") return { id: null, body: comment, createdAt: "" };
    return {
      id: comment?.id ?? null,
      body: comment?.body || comment?.text || "",
      createdAt: comment?.created_at || comment?.createdAt || ""
    };
  }

  // src/task-manager/state/task-actions.js
  var taskActions = {
    methods: {
      /**
       * Sustituye el workspace solo si esta sigue siendo la recarga mas reciente.
       *
       * @param {string} [preferredId] Seleccion que se intenta conservar.
       * @returns {Promise<boolean|null>} `true` si confirma, `false` si falla y
       * `null` si una recarga posterior vuelve obsoleta esta respuesta.
       */
      async reloadWorkspace(preferredId = this.selectedTaskId) {
        const requestSequence = ++this.reloadRequestSequence;
        const initial = this.loading;
        if (!initial) this.refreshing = true;
        this.errorMessage = "";
        try {
          const [rawTasks, tags, comments] = await Promise.all([
            listTasks(),
            listTags(),
            listAllComments()
          ]);
          if (!Array.isArray(rawTasks) || !Array.isArray(tags) || !Array.isArray(comments)) throw new ApiError("El servidor devolvi\xF3 datos de tareas no v\xE1lidos.");
          if (requestSequence !== this.reloadRequestSequence) return null;
          const tasks = rawTasks.map(mapTask);
          this.commentIndex = comments.reduce((index, comment) => {
            if (comment?.ref && comment.body) index.set(comment.ref, `${index.get(comment.ref) || ""} ${comment.body}`);
            return index;
          }, /* @__PURE__ */ new Map());
          const selectedId = preferredId && tasks.some((task) => task.id === preferredId) ? preferredId : tasks[0]?.id || "";
          this.tasks = tasks;
          this.coldTaskList = coldTasks(tasks, comments);
          this.tags = tags;
          this.reconcileUiPreferences(tags);
          this.selectedTaskId = selectedId;
          if (this.detail?.id !== selectedId) this.detail = null;
          this.unavailable = false;
          const parentIds = [...new Set(tasks.map((task) => task.parentId).filter(Boolean))];
          const hadTreeState = this.expandedIds.length > 0 || this.collapsedIds.length > 0;
          this.expandedIds = hadTreeState ? this.expandedIds.filter((id) => parentIds.includes(id)) : parentIds;
          this.collapsedIds = this.collapsedIds.filter((id) => parentIds.includes(id));
          if (selectedId) await this.loadTaskDetail(selectedId, false);
          else this.syncEditDraft();
          return true;
        } catch (error) {
          if (requestSequence !== this.reloadRequestSequence) return null;
          this.handleError(error);
          return false;
        } finally {
          if (requestSequence === this.reloadRequestSequence) {
            this.loading = false;
            this.refreshing = false;
          }
        }
      },
      /**
       * Recarga pedida por el canal. Varios avisos pueden llegar mientras una
       * recarga está en vuelo: el primero la hace y los demás se juntan en una
       * sola ronda más, en vez de una recarga completa por aviso.
       */
      noticeReload() {
        if (this.noticeReloadRunning) {
          this.noticeReloadPending = true;
          return;
        }
        this.noticeReloadRunning = true;
        this.reloadWorkspace().finally(() => {
          this.noticeReloadRunning = false;
          if (this.noticeReloadPending) {
            this.noticeReloadPending = false;
            this.noticeReload();
          }
        });
      },
      async loadTaskDetail(id, openOnMobile = true) {
        if (!id || this.isBusy(`detail:${id}`)) return;
        const requestSequence = ++this.detailRequestSequence;
        this.selectedTaskId = id;
        if (this.detail?.id !== id) this.cancelCommentEdit();
        if (this.detail?.id !== id) this.detail = null;
        this.setBusy(`detail:${id}`, true);
        this.errorMessage = "";
        try {
          const detail = await showTask(id);
          if (requestSequence !== this.detailRequestSequence || this.selectedTaskId !== id) return;
          this.detail = mapTask(detail);
          this.syncEditDraft();
          this.isMobileInspectorOpen = openOnMobile;
        } catch (error) {
          if (requestSequence !== this.detailRequestSequence || this.selectedTaskId !== id) return;
          this.handleError(error);
        } finally {
          this.setBusy(`detail:${id}`, false);
        }
      },
      /**
       * Ejecuta una escritura sin optimismo y recarga el estado confirmado. Una
       * escritura puede estar confirmada aunque la recarga posterior falle; ese
       * caso se conserva como error explicito para evitar reintentos a ciegas.
       *
       * @param {string} key Clave compartida con los controles que consultan busy.
       * @param {() => Promise<*>} request Escritura HTTP diferida.
       * @param {string} message Confirmacion que se muestra tras recargar.
       * @param {string} [preferredId] Seleccion deseada despues de la escritura.
       * @param {{ useResultId?: boolean }} [options]
       * @returns {Promise<*|null>} Respuesta de escritura o `null` si no se ejecuto.
       */
      async mutate(key, request, message, preferredId = this.selectedTaskId, options = {}) {
        if (this.isBusy(key)) return null;
        this.setBusy(key, true);
        this.errorMessage = "";
        try {
          const result = await request();
          if (result?.ref) {
            this.lastTaskWriteRef = result.ref;
            this.lastTaskWriteAt = Date.now();
          }
          const nextId = options.useResultId && (result?.ref || result?.id) ? result.ref || result.id : preferredId;
          const reloaded = await this.reloadWorkspace(nextId);
          if ((reloaded === true || reloaded === null) && !this.errorMessage && !this.unavailable) {
            this.showToast(message);
          }
          if (reloaded === false) {
            const reason = this.errorMessage || "No se pudo recuperar el estado actualizado.";
            this.errorMessage = `${message} La operaci\xF3n fue confirmada, pero no se pudieron actualizar los datos. ${reason}`;
          }
          return result;
        } catch (error) {
          this.handleError(error);
          return null;
        } finally {
          this.setBusy(key, false);
        }
      },
      async createTask() {
        const title = this.newTaskDraft.title.trim();
        if (!title) return;
        const result = await this.mutate("create", async () => createTask({
          title,
          priority: Number(this.newTaskDraft.priority),
          description: this.newTaskDraft.description.trim(),
          parent: this.newTaskDraft.parent || void 0,
          tags: [...this.newTaskDraft.tags]
        }), "Tarea creada.", "", { useResultId: true });
        if (result) {
          this.isNewTaskOpen = false;
          this.newTaskDraft = this.emptyTaskDraft();
          this.activeInspectorTab = "detail";
        }
      },
      openSubtaskDialog(parentId = this.selectedTaskId) {
        this.newTaskDraft = this.emptyTaskDraft(parentId);
        this.isNewTaskOpen = true;
        this.isInspectorMenuOpen = false;
      },
      syncEditDraft() {
        const task = this.detail;
        if (!task) {
          this.editDraft = this.emptyEditDraft();
          return;
        }
        const listed = this.mappedTasks.find((entry) => entry.id === task.id);
        this.editDraft = {
          title: task.title || "",
          description: task.description || "",
          notes: task.notes || "",
          assignee: task.assignee || "",
          parent: listed?.parentId || "",
          tags: Array.isArray(task.chips) ? [...task.chips] : []
        };
      },
      openEditTask() {
        if (!this.hasSelection) return;
        this.syncEditDraft();
        this.activeInspectorTab = "edit";
        this.isInspectorMenuOpen = false;
      },
      cancelTaskEdit() {
        this.syncEditDraft();
        this.activeInspectorTab = "detail";
      },
      async saveSelectedTask() {
        const title = this.editDraft.title.trim();
        if (!title || !this.hasSelection) return;
        const changes = {
          title,
          description: this.editDraft.description.trim(),
          notes: this.editDraft.notes.trim(),
          assignee: this.editDraft.assignee.trim(),
          tags: [...this.editDraft.tags]
        };
        if ("parent" in this.editDraft) changes.parent = this.editDraft.parent;
        const result = await this.mutate(`edit:${this.selectedTaskId}`, () => updateTask(this.selectedTaskId, changes), "Cambios guardados.");
        if (result) this.activeInspectorTab = "detail";
      },
      async updateTaskField(id, field, value) {
        if (!id) return;
        if (field === "status" && value === "done") {
          await this.closeTask(id);
          return;
        }
        await this.mutate(`${field}:${id}`, () => updateTask(id, {
          [field]: field === "priority" ? Number(value) : value
        }), field === "priority" ? "Prioridad actualizada." : "Estado actualizado.", this.selectedTaskId);
      },
      async closeTask(id = this.selectedTaskId) {
        if (!id) return;
        await this.mutate(`close:${id}`, () => closeTask(id), "Tarea cerrada.", id);
        this.isInspectorMenuOpen = false;
      },
      async addDependency() {
        const parentId = this.dependencyDraft.trim();
        if (!parentId || !this.hasSelection) return;
        const result = await this.mutate(`dep-add:${this.selectedTaskId}`, () => addTaskBlocker(this.selectedTaskId, parentId), "Dependencia a\xF1adida.");
        if (result) this.dependencyDraft = "";
      },
      async removeDependency(parentId) {
        if (!parentId || !this.hasSelection) return;
        await this.mutate(`dep-remove:${parentId}`, () => removeTaskBlocker(this.selectedTaskId, parentId), "Dependencia retirada.");
      },
      async removeDependent(childId) {
        if (!childId || !this.hasSelection) return;
        await this.mutate(`dep-remove:${childId}`, () => removeTaskBlocker(childId, this.selectedTaskId), "Dependencia retirada.");
      },
      async addComment() {
        const body = this.commentDraft.trim();
        if (!body || !this.hasSelection) return;
        const result = await this.mutate(`comment:${this.selectedTaskId}`, () => addTaskComment(this.selectedTaskId, body), "Comentario a\xF1adido.");
        if (result) this.commentDraft = "";
      },
      startCommentEdit(comment) {
        this.editingCommentId = comment?.id ?? null;
        this.editingCommentDraft = comment?.body || "";
      },
      cancelCommentEdit() {
        this.editingCommentId = null;
        this.editingCommentDraft = "";
      },
      async saveCommentEdit() {
        const body = this.editingCommentDraft.trim();
        if (!body || !this.hasSelection || !this.editingCommentId) return;
        const result = await this.mutate(`comment-edit:${this.selectedTaskId}`, () => editTaskComment(this.selectedTaskId, this.editingCommentId, body), "Comentario actualizado.");
        if (result) this.cancelCommentEdit();
      },
      async copyTaskId(id = this.selectedTaskId) {
        this.isInspectorMenuOpen = false;
        try {
          await navigator.clipboard.writeText(id);
          this.showToast(`ID ${id} copiado.`);
        } catch {
          this.errorMessage = "No se pudo copiar el identificador al portapapeles.";
        }
      },
      async runTaskMenuAction(action, id) {
        this.openTaskMenuId = "";
        if (action === "copy") await this.copyTaskId(id);
        if (action === "close") await this.closeTask(id);
        if (action === "edit") {
          if (this.selectedTaskId !== id || this.detail?.id !== id) await this.loadTaskDetail(id);
          if (this.selectedTaskId === id && this.detail?.id === id) this.openEditTask();
        }
      }
    }
  };

  // src/task-manager/state/task-tree.js
  var CURRENT_PERIODS = ["today", "week", "month", "year"];
  function startOfPeriod(now, period) {
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    if (period === "week") {
      const daysSinceMonday = (start.getDay() + 6) % 7;
      start.setDate(start.getDate() - daysSinceMonday);
    } else if (period === "month") {
      start.setDate(1);
    } else if (period === "year") {
      start.setMonth(0, 1);
    }
    return start;
  }
  function endOfPeriod(start, period) {
    const end = new Date(start);
    if (period === "today") end.setDate(end.getDate() + 1);
    if (period === "week") end.setDate(end.getDate() + 7);
    if (period === "month") end.setMonth(end.getMonth() + 1, 1);
    if (period === "year") end.setFullYear(end.getFullYear() + 1, 0, 1);
    return end;
  }
  function taskInCurrentPeriod(task, period, now = /* @__PURE__ */ new Date()) {
    if (!CURRENT_PERIODS.includes(period)) return false;
    const start = startOfPeriod(now, period).getTime();
    const end = endOfPeriod(new Date(start), period).getTime();
    return ["created_at", "updated_at"].some((field) => {
      const timestamp = Date.parse(task[field]);
      return Number.isFinite(timestamp) && timestamp >= start && timestamp < end;
    });
  }
  var taskTree = {
    computed: {
      mappedTasks() {
        const source = this.tasks.map((task) => ({ ...task }));
        const byId = new Map(source.map((task) => [task.id, task]));
        const childrenByParent = /* @__PURE__ */ new Map();
        for (const task of source) {
          if (!task.parentId || !byId.has(task.parentId)) continue;
          const children = childrenByParent.get(task.parentId) || [];
          children.push(task);
          childrenByParent.set(task.parentId, children);
        }
        const levelFor = (task, seen = /* @__PURE__ */ new Set()) => {
          if (!task.parentId || !byId.has(task.parentId) || seen.has(task.id)) return 1;
          seen.add(task.id);
          return levelFor(byId.get(task.parentId), seen) + 1;
        };
        return source.map((task) => {
          const children = childrenByParent.has(task.id);
          return { ...mapTask(task, children), level: levelFor(task), children };
        });
      },
      filteredTaskIds() {
        const query = this.normalizeSearchText(this.taskSearchQuery);
        return new Set(this.mappedTasks.filter(
          (task) => (this.activeTaskFilter === "all" || this.activeTaskFilter === "current" && taskInCurrentPeriod(task, this.currentPeriod, /* @__PURE__ */ new Date()) || this.activeTaskFilter !== "current" && task.status === this.activeTaskFilter) && (!this.advancedStatuses.length || this.advancedStatuses.includes(task.status)) && (!this.advancedPriorities.length || this.advancedPriorities.includes(task.priority)) && (!this.advancedTags.length || task.chips.some((tag) => this.advancedTags.includes(tag))) && (!this.quickTagFilter || task.chips.includes(this.quickTagFilter)) && // La búsqueda cubre descripción, notas y comentarios (tk-124): lo que
          // la lista trae y lo que el índice de comentarios agrupa por referencia.
          (!query || this.normalizeSearchText([
            task.id,
            task.title,
            task.chips.join(" "),
            task.description,
            task.notes,
            this.commentIndex?.get(task.id)
          ].join(" ")).includes(query))
        ).map((task) => task.id));
      },
      visibleTasks() {
        const byId = new Map(this.mappedTasks.map((task) => [task.id, task]));
        const childrenByParent = /* @__PURE__ */ new Map();
        const roots = [];
        for (const task of this.mappedTasks) {
          if (task.parentId && byId.has(task.parentId)) {
            const children = childrenByParent.get(task.parentId) || [];
            children.push(task);
            childrenByParent.set(task.parentId, children);
          } else {
            roots.push(task);
          }
        }
        const hasFilter = this.activeTaskFilter !== "all" || this.advancedStatuses.length || this.advancedPriorities.length || this.advancedTags.length || this.quickTagFilter || this.taskSearchQuery.trim();
        const reachable = /* @__PURE__ */ new Set();
        const markReachable = (task) => {
          if (reachable.has(task.id)) return;
          reachable.add(task.id);
          for (const child of childrenByParent.get(task.id) || []) markReachable(child);
        };
        for (const root of roots) markReachable(root);
        for (const task of this.mappedTasks) {
          if (!reachable.has(task.id)) roots.push(task);
        }
        const rows = [];
        const visited = /* @__PURE__ */ new Set();
        const visit = (task) => {
          if (visited.has(task.id)) return;
          visited.add(task.id);
          if (this.filteredTaskIds.has(task.id)) rows.push(task);
          const canTraverse = this.expandedIds.includes(task.id) || hasFilter && !this.collapsedIds.includes(task.id);
          if (canTraverse) {
            for (const child of childrenByParent.get(task.id) || []) visit(child);
          }
        };
        for (const root of roots) visit(root);
        return rows;
      },
      selectedTask() {
        const listed = this.mappedTasks.find((task2) => task2.id === this.selectedTaskId);
        const task = this.detail || listed || {};
        return { ...mapTask(task), status: listed?.status || task.status || "open" };
      },
      // El detalle trae el padre como numero interno; la lista lo trae ya
      // traducido a su referencia visible. La navegacion vale la referencia.
      selectedParentTask() {
        const listed = this.mappedTasks.find((task) => task.id === this.selectedTaskId);
        const parentId = listed?.parentId || this.selectedTask.parentId;
        if (!parentId) return null;
        const parent = this.mappedTasks.find((task) => task.id === parentId);
        return parent ? { id: parent.id, title: parent.title } : null;
      },
      hasSelection() {
        return Boolean(this.selectedTaskId && this.selectedTask.id);
      },
      activeTaskCount() {
        return this.tasks.filter((task) => task.status !== "done").length;
      },
      selectedSubtasks() {
        return this.mappedTasks.filter((task) => task.parentId === this.selectedTaskId);
      },
      hasAdvancedFilters() {
        return Boolean(this.advancedStatuses.length || this.advancedPriorities.length || this.advancedTags.length);
      },
      taskFilterProps() {
        return {
          unavailable: this.unavailable,
          errorMessage: this.errorMessage,
          refreshing: this.refreshing,
          coldTasks: this.coldTaskList,
          taskSearchQuery: this.taskSearchQuery,
          advancedFiltersOpen: this.isAdvancedFiltersOpen,
          hasAdvancedFilters: this.hasAdvancedFilters,
          advancedStatuses: this.advancedStatuses,
          advancedPriorities: this.advancedPriorities,
          advancedTags: this.advancedTags,
          tagCatalog: this.tagCatalog,
          activeTaskFilter: this.activeTaskFilter,
          quickTagFilter: this.quickTagFilter,
          taskChipsVisible: this.areTaskChipsVisible,
          currentPeriod: this.currentPeriod,
          visibleCount: this.visibleTasks.length,
          listMenuOpen: this.isListMenuOpen,
          tagColor: this.tagColor
        };
      }
    },
    methods: {
      toggleExpanded(id) {
        if (this.expandedIds.includes(id)) {
          this.expandedIds = this.expandedIds.filter((item) => item !== id);
          if (!this.collapsedIds.includes(id)) this.collapsedIds = [...this.collapsedIds, id];
          return;
        }
        this.expandedIds = [...this.expandedIds, id];
        this.collapsedIds = this.collapsedIds.filter((item) => item !== id);
      },
      setAllExpanded(expanded) {
        const parentIds = this.mappedTasks.filter((task) => task.children).map((task) => task.id);
        this.expandedIds = expanded ? parentIds : [];
        this.collapsedIds = expanded ? [] : parentIds;
        this.isListMenuOpen = false;
      },
      clearAdvancedFilters() {
        this.advancedStatuses = [];
        this.advancedPriorities = [];
        this.advancedTags = [];
      }
    }
  };

  // src/task-manager/state/workspace-data.js
  var COLORS = [
    { id: "blue", hex: "#62a7ff" },
    { id: "purple", hex: "#b58bff" },
    { id: "green", hex: "#58d392" },
    { id: "yellow", hex: "#f2bf59" },
    { id: "orange", hex: "#ff9a55" },
    { id: "red", hex: "#ff7373" }
  ];
  var UI_PREFERENCES_STORAGE_KEY = "project-task-manager.ui-preferences.v1";
  var UI_PREFERENCES_VERSION = 1;
  var ACTIVE_TASK_FILTERS = ["all", "current", "open", "in_progress", "blocked", "done"];
  var CURRENT_PERIODS2 = ["today", "week", "month", "year"];
  var INSPECTOR_TABS = ["detail", "edit", "subtasks", "dependencies", "comments"];
  var TASK_STATUSES = ["open", "in_progress", "done", "blocked"];
  var TASK_PRIORITIES = ["0", "1", "2", "3"];
  var PERSISTED_FIELDS = [
    "activeTaskFilter",
    "currentPeriod",
    "activeInspectorTab",
    "selectedTaskId",
    "advancedStatuses",
    "advancedPriorities",
    "advancedTags",
    "quickTagFilter",
    "taskSearchQuery",
    "expandedIds",
    "collapsedIds",
    "areTaskChipsVisible"
  ];
  function getStorage() {
    try {
      return globalThis.localStorage || null;
    } catch {
      return null;
    }
  }
  function validString(value, maxLength = 200) {
    return typeof value === "string" && value.length <= maxLength;
  }
  function validStringArray(value, allowedValues = null) {
    if (!Array.isArray(value)) return [];
    return [...new Set(value.filter((item) => validString(item) && (!allowedValues || allowedValues.includes(item))))];
  }
  function normalizePreferences(value) {
    const source = value && typeof value === "object" && !Array.isArray(value) ? value : {};
    const expandedIds = validStringArray(source.expandedIds);
    const collapsedIds = validStringArray(source.collapsedIds).filter((id) => !expandedIds.includes(id));
    return {
      activeTaskFilter: ACTIVE_TASK_FILTERS.includes(source.activeTaskFilter) ? source.activeTaskFilter : "all",
      currentPeriod: CURRENT_PERIODS2.includes(source.currentPeriod) ? source.currentPeriod : "today",
      activeInspectorTab: INSPECTOR_TABS.includes(source.activeInspectorTab) ? source.activeInspectorTab : "detail",
      selectedTaskId: validString(source.selectedTaskId) ? source.selectedTaskId : "",
      advancedStatuses: validStringArray(source.advancedStatuses, TASK_STATUSES),
      advancedPriorities: validStringArray(source.advancedPriorities, TASK_PRIORITIES),
      advancedTags: validStringArray(source.advancedTags),
      quickTagFilter: validString(source.quickTagFilter, 40) ? source.quickTagFilter : "",
      taskSearchQuery: validString(source.taskSearchQuery, 500) ? source.taskSearchQuery : "",
      expandedIds,
      collapsedIds,
      areTaskChipsVisible: typeof source.areTaskChipsVisible === "boolean" ? source.areTaskChipsVisible : true
    };
  }
  var workspaceData = {
    data() {
      return {
        tasks: [],
        // Índice de comentarios por referencia de tarea (tk-124): alimenta la
        // búsqueda en comentarios. No es preferencia de interfaz: se recarga con
        // el workspace.
        commentIndex: /* @__PURE__ */ new Map(),
        // Tareas abiertas o en curso sin actividad de tablero reciente. Se
        // recalcula con cada carga del workspace (tk-272).
        coldTaskList: [],
        tags: [],
        detail: null,
        selectedTaskId: "",
        colors: COLORS,
        loading: true,
        refreshing: false,
        unavailable: false,
        errorMessage: "",
        busy: {},
        activeInspectorTab: "detail",
        activeTaskFilter: "all",
        currentPeriod: "today",
        taskSearchQuery: "",
        quickTagFilter: "",
        advancedStatuses: [],
        advancedPriorities: [],
        advancedTags: [],
        isAdvancedFiltersOpen: false,
        areTaskChipsVisible: true,
        expandedIds: [],
        collapsedIds: [],
        isListMenuOpen: false,
        openTaskMenuId: "",
        isInspectorMenuOpen: false,
        isMobileInspectorOpen: false,
        isNewTaskOpen: false,
        isTagManagerOpen: false,
        isDecisionsOpen: false,
        // Aviso de cambio ajeno: el panel de decisiones relee el registro cuando
        // el canal cuenta algo nuevo.
        decisionsToken: 0,
        newTaskDraft: { title: "", description: "", priority: "2", parent: "", tags: [] },
        editDraft: { title: "", description: "", notes: "", assignee: "", tags: [] },
        dependencyDraft: "",
        commentDraft: "",
        editingCommentId: null,
        editingCommentDraft: "",
        tagSearchQuery: "",
        tagFormMode: "",
        editingTagName: "",
        tagDraft: { name: "", description: "", color: "blue" },
        tagFormError: "",
        detailRequestSequence: 0,
        reloadRequestSequence: 0,
        // La última tarea que esta ventana escribió, con el momento en que se
        // confirmó: con ella la ventana evita repetirse el aviso de su propio
        // cambio, sin identidad viajando al servidor.
        lastTaskWriteRef: "",
        lastTaskWriteAt: 0,
        // Una ronda de recarga pedida por el canal, con otra pendiente si
        // llegaron más avisos mientras corría.
        noticeReloadRunning: false,
        noticeReloadPending: false,
        uiPreferencesPersistTimer: null,
        workspaceEventSource: null
      };
    },
    methods: {
      emptyTaskDraft(parent = "") {
        return { title: "", description: "", priority: "2", parent, tags: [] };
      },
      emptyEditDraft() {
        return { title: "", description: "", notes: "", assignee: "", tags: [] };
      },
      emptyTagDraft() {
        return { name: "", description: "", color: "blue" };
      },
      getUiPreferencesStorage() {
        return getStorage();
      },
      readUiPreferences() {
        const storage = this.getUiPreferencesStorage();
        if (!storage) return null;
        try {
          const parsed = JSON.parse(storage.getItem(UI_PREFERENCES_STORAGE_KEY));
          if (parsed?.version !== UI_PREFERENCES_VERSION) return null;
          return normalizePreferences(parsed);
        } catch {
          return null;
        }
      },
      restoreUiPreferences() {
        const preferences = this.readUiPreferences();
        if (!preferences) return false;
        for (const field of PERSISTED_FIELDS) this[field] = preferences[field];
        return true;
      },
      reconcileUiPreferences(tags = []) {
        const availableTags = new Set(tags.map((tag) => tag?.name).filter((name) => validString(name, 40)));
        this.advancedTags = this.advancedTags.filter((tag) => availableTags.has(tag));
        if (this.quickTagFilter && !availableTags.has(this.quickTagFilter)) this.quickTagFilter = "";
      },
      persistUiPreferences() {
        const storage = this.getUiPreferencesStorage();
        if (!storage) return;
        const preferences = { version: UI_PREFERENCES_VERSION };
        for (const field of PERSISTED_FIELDS) preferences[field] = this[field];
        try {
          storage.setItem(UI_PREFERENCES_STORAGE_KEY, JSON.stringify(preferences));
        } catch {
        }
      },
      scheduleUiPreferencesPersistence() {
        if (this.uiPreferencesPersistTimer) clearTimeout(this.uiPreferencesPersistTimer);
        this.uiPreferencesPersistTimer = setTimeout(() => {
          this.uiPreferencesPersistTimer = null;
          this.persistUiPreferences();
        }, 150);
      },
      cancelUiPreferencesPersistence() {
        if (!this.uiPreferencesPersistTimer) return;
        clearTimeout(this.uiPreferencesPersistTimer);
        this.uiPreferencesPersistTimer = null;
      }
    }
  };

  // src/task-manager/state/ui.js
  var workspaceUi = {
    methods: {
      handleError(error) {
        if (error?.unavailable) {
          this.unavailable = true;
          this.errorMessage = "";
        } else {
          this.errorMessage = error?.message || "No se pudo completar la operaci\xF3n.";
          notifyError(this.errorMessage);
        }
      },
      setBusy(key, value) {
        const next = { ...this.busy };
        if (value) next[key] = true;
        else delete next[key];
        this.busy = next;
      },
      isBusy(key) {
        return Boolean(this.busy[key]);
      },
      showToast(message) {
        notifySuccess(message);
      },
      showWorkspaceChangeToast(changes = []) {
        if (!Array.isArray(changes) || !changes.length) {
          notifyInfo("Cambios detectados. No se pudo identificar la tarea. La lista se ha recargado.");
          return;
        }
        changes.forEach((change) => {
          notifyInfo(`Tarea ${change.id} ${change.action}: \u201C${change.title}\u201D. La lista se ha recargado.`);
        });
      },
      normalizeSearchText(value) {
        return String(value || "").normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLocaleLowerCase("es").trim();
      },
      sameTagName(left, right) {
        return String(left || "").normalize("NFC").trim().localeCompare(String(right || "").normalize("NFC").trim(), "es", { sensitivity: "accent" }) === 0;
      },
      statusLabel(status) {
        return statusLabel(status);
      },
      priorityLabel(priority) {
        return priorityLabel(priority);
      },
      statusBadgeClass(status) {
        return statusBadgeClass(status);
      },
      toggleTaskMenu(id) {
        this.openTaskMenuId = this.openTaskMenuId === id ? "" : id;
        this.isListMenuOpen = false;
        this.isInspectorMenuOpen = false;
      },
      handleWorkspaceClick() {
        this.isListMenuOpen = false;
        this.openTaskMenuId = "";
        this.isInspectorMenuOpen = false;
      },
      handleEscape(event) {
        if (event.key !== "Escape") return;
        if (isConfirmOpen()) return;
        this.isListMenuOpen = false;
        this.openTaskMenuId = "";
        this.isInspectorMenuOpen = false;
        this.isAdvancedFiltersOpen = false;
        this.isNewTaskOpen = false;
        this.isTagManagerOpen = false;
        this.isDecisionsOpen = false;
      }
    }
  };

  // src/task-manager/state/workspace.js
  var OWN_WRITE_MARGIN_MS = 2e3;
  var workspaceOptions = {
    name: "AppShell",
    mixins: [
      workspaceData,
      workspaceUi,
      taskTree,
      taskActions,
      tagActions,
      inspectorState
    ],
    watch: {
      activeInspectorTab() {
        this.scheduleUiPreferencesPersistence();
      },
      activeTaskFilter() {
        this.scheduleUiPreferencesPersistence();
      },
      currentPeriod() {
        this.scheduleUiPreferencesPersistence();
      },
      selectedTaskId() {
        this.scheduleUiPreferencesPersistence();
      },
      advancedStatuses() {
        this.scheduleUiPreferencesPersistence();
      },
      advancedPriorities() {
        this.scheduleUiPreferencesPersistence();
      },
      advancedTags() {
        this.scheduleUiPreferencesPersistence();
      },
      quickTagFilter() {
        this.scheduleUiPreferencesPersistence();
      },
      taskSearchQuery() {
        this.scheduleUiPreferencesPersistence();
      },
      expandedIds() {
        this.scheduleUiPreferencesPersistence();
      },
      collapsedIds() {
        this.scheduleUiPreferencesPersistence();
      },
      areTaskChipsVisible() {
        this.scheduleUiPreferencesPersistence();
      }
    },
    mounted() {
      window.addEventListener("keydown", this.handleEscape);
      this.restoreUiPreferences();
      this.connectWorkspaceEvents();
      this.reloadWorkspace();
    },
    beforeUnmount() {
      window.removeEventListener("keydown", this.handleEscape);
      this.cancelUiPreferencesPersistence();
      this.disconnectWorkspaceEvents();
    },
    methods: {
      connectWorkspaceEvents() {
        if (typeof EventSource === "undefined") return;
        this.disconnectWorkspaceEvents();
        const source = new EventSource("/api/events");
        source.addEventListener("workspace.changed", (event) => {
          let change;
          try {
            change = JSON.parse(event.data);
          } catch {
            return;
          }
          this.decisionsToken += 1;
          this.noticeReload();
          if (change.resource === "tasks" && !this.suppressOwnWriteNotice(change.changes)) {
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
        this.lastTaskWriteRef = "";
        this.lastTaskWriteAt = 0;
        return true;
      }
    }
  };

  // src/task-manager/components/dialog-focus.js
  var FOCUSABLE_SELECTOR = [
    "a[href]",
    "button:not([disabled])",
    'input:not([disabled]):not([type="hidden"])',
    "select:not([disabled])",
    "textarea:not([disabled])",
    '[tabindex]:not([tabindex="-1"])'
  ].join(",");
  function isVisible(element) {
    return Boolean(element.offsetWidth || element.offsetHeight || element.getClientRects().length);
  }
  var dialogFocus = {
    data() {
      return { focusReturnTarget: null };
    },
    watch: {
      open(value) {
        if (value) this.enterDialog();
        else this.exitDialog();
      }
    },
    mounted() {
      if (this.open) this.enterDialog();
    },
    beforeUnmount() {
      this.exitDialog();
    },
    methods: {
      /**
       * Elemento que actua como raiz del dialogo. Los componentes marcan su
       * elemento modal con `ref="dialogRoot"`; si no lo hacen, se usa la raiz
       * del propio componente.
       */
      dialogElement() {
        return this.$refs.dialogRoot || this.$el;
      },
      focusableElements(scope = this.dialogElement()) {
        if (!scope?.querySelectorAll) return [];
        return [...scope.querySelectorAll(FOCUSABLE_SELECTOR)].filter(isVisible);
      },
      /** Entra en el dialogo guardando de donde venimos para poder volver. */
      enterDialog(scope) {
        const active = document.activeElement;
        this.focusReturnTarget = active && active !== document.body ? active : null;
        this.$nextTick(() => this.focusInside(scope));
      },
      /** Devuelve el foco al control que abrio el dialogo, si sigue en la pagina. */
      exitDialog() {
        const target = this.focusReturnTarget;
        this.focusReturnTarget = null;
        if (target?.isConnected && typeof target.focus === "function") target.focus();
      },
      /**
       * El foco entra en el contenedor del dialogo, no en su primer control.
       * Asi el lector de pantalla anuncia el dialogo por su nombre y no se abre
       * directamente sobre la "x" de cerrar.
       */
      focusInside(scope = this.dialogElement()) {
        const root = scope || this.dialogElement();
        if (root?.focus) return root.focus();
        const [first] = this.focusableElements(root);
        if (first) first.focus();
      },
      /** Mantiene el tabulador dentro del dialogo: del ultimo vuelve al primero. */
      trapDialogTab(event) {
        const scope = this.dialogElement();
        const focusables = this.focusableElements(scope);
        if (!focusables.length) return;
        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        const active = document.activeElement;
        if (event.shiftKey && (active === first || active === scope)) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && active === last) {
          event.preventDefault();
          first.focus();
        }
      }
    }
  };

  // src/task-manager/components/tag-manager-drawer.js
  var TagManagerDrawer = {
    name: "TagManagerDrawer",
    mixins: [dialogFocus],
    props: {
      open: { type: Boolean, required: true },
      tags: { type: Array, required: true },
      visibleTags: { type: Array, required: true },
      colors: { type: Array, required: true },
      searchQuery: { type: String, required: true },
      formMode: { type: String, required: true },
      draft: { type: Object, required: true },
      editingName: { type: String, required: true },
      formError: { type: String, required: true },
      busy: { type: Function, required: true },
      colorHex: { type: Function, required: true }
    },
    emits: ["close", "update-search", "open-create", "open-edit", "request-delete", "update-draft", "close-form", "save"],
    methods: {
      update(field, value) {
        this.$emit("update-draft", { ...this.draft, [field]: value });
      }
    },
    template: `
    <section v-show="open" class="tag-manager-layer is-open" @click.self="$emit('close')">
  <aside class="tag-manager-drawer" ref="dialogRoot" tabindex="-1" @keydown.tab="trapDialogTab" role="dialog" aria-modal="true" aria-labelledby="tag-manager-title">
    <header class="tag-manager-header">
      <div>
        <h2 id="tag-manager-title">Gestor de etiquetas</h2>
        <p>Cat\xE1logo del proyecto</p>
      </div><button class="tk-icon-button" type="button" aria-label="Cerrar gestor" title="Cerrar gestor"
        @click="$emit('close')"><tk-icon name="x-mark" /></button>
    </header>
    <div class="tag-manager-body">
      <div class="tag-manager-toolbar"><input :value="searchQuery" class="tk-input" type="search"
          placeholder="Buscar etiquetas" @input="$emit('update-search', $event.target.value)" /><button
          class="tk-button tk-button--secondary tk-button--compact" type="button" title="Nueva etiqueta"
          @click="$emit('open-create')"><tk-icon name="plus" />Nueva</button></div>
      <div class="tag-manager-summary"><span>{{ tags.length }}
          {{ tags.length === 1 ? 'etiqueta' : 'etiquetas' }}</span><span>A\u2013Z</span></div>
      <div class="tag-manager-list">
        <article v-for="tag in visibleTags" :key="tag.name" class="tag-manager-row"><span class="tag-manager-dot"
            :style="{ '--tag-color': colorHex(tag.color) }"></span>
          <div class="tag-manager-copy">
            <div class="tag-manager-name">{{ tag.name }}</div>
            <div v-show="tag.description" class="tag-manager-description">{{ tag.description }}</div>
          </div><span class="tag-manager-uses">{{ tag.uses || 0 }} usos</span>
          <div class="tag-manager-actions"><button class="tk-icon-button tk-icon-button--compact" type="button"
              :aria-label="'Editar ' + tag.name" :title="'Editar ' + tag.name"
              @click="$emit('open-edit', tag)"><tk-icon name="pencil" /></button><button
              class="tk-icon-button tk-icon-button--compact" type="button" :aria-label="'Eliminar ' + tag.name"
              :title="'Eliminar ' + tag.name"
              @click="$emit('request-delete', tag)"><tk-icon name="trash" /></button></div>
        </article>
        <div v-show="!visibleTags.length" class="tag-manager-empty">No hay etiquetas que coincidan.</div>
      </div>
      <form v-show="formMode" class="tag-manager-form" @submit.prevent="$emit('save')">
        <div class="tag-manager-form-header">
          <strong>{{ formMode === 'edit' ? 'Editar etiqueta' : 'Nueva etiqueta' }}</strong><button
            class="tk-icon-button" type="button" aria-label="Cerrar formulario" title="Cerrar formulario"
            @click="$emit('close-form')"><tk-icon name="x-mark" /></button>
        </div><label class="tag-manager-field"><span class="tag-manager-label">Nombre</span><input :value="draft.name"
            class="tk-input" maxlength="40" required @input="update('name', $event.target.value)" /></label><label
          class="tag-manager-field"><span class="tag-manager-label">Descripci\xF3n</span><input :value="draft.description"
            class="tk-input" maxlength="90" @input="update('description', $event.target.value)" /></label>
        <fieldset class="tag-manager-field">
          <legend class="tag-manager-label">Color</legend>
          <div class="tag-manager-colors" role="radiogroup"><button v-for="color in colors" :key="color.id"
              class="tag-manager-color" type="button" role="radio" :aria-label="'Color ' + color.id"
              :title="'Seleccionar color ' + color.id"
              :aria-checked="draft.color === color.id" :style="{ '--tag-color': color.hex }"
              @click="update('color', color.id)"></button></div>
        </fieldset>
        <p v-show="formError" class="tag-manager-error">{{ formError }}</p>
        <div class="tag-manager-form-actions"><button class="tk-button tk-button--ghost" type="button"
            @click="$emit('close-form')">Cancelar</button><button class="tk-button tk-button--primary" type="submit"
            :disabled="busy('tag-save:' + (editingName || draft.name))">Guardar</button></div>
      </form>
    </div>
    <footer class="tag-manager-footer">Los usos proceden de las tareas reales.</footer>
  </aside>
</section>`
  };

  // src/task-manager/components/tag-picker-field.js
  var TagPickerField = {
    name: "TagPickerField",
    props: {
      modelValue: { type: Array, required: true },
      tags: { type: Array, required: true },
      colors: { type: Array, required: true },
      busy: { type: Boolean, default: false },
      wide: { type: Boolean, default: false },
      resetKey: { type: [String, Number, Boolean], default: "" }
    },
    emits: ["update:modelValue"],
    data() {
      return { open: false, search: "" };
    },
    computed: {
      availableTags() {
        const query = this.normalize(this.search);
        return this.tags.filter((tag) => !this.modelValue.includes(tag.name) && (!query || this.normalize(tag.name).includes(query)));
      }
    },
    watch: {
      resetKey() {
        this.open = false;
        this.search = "";
      }
    },
    methods: {
      normalize(value) {
        return String(value || "").normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLocaleLowerCase("es").trim();
      },
      tagColor(name) {
        const colorId = this.tags.find((tag) => tag.name === name)?.color || "blue";
        return this.colors.find((color) => color.id === colorId)?.hex || this.colors[0]?.hex;
      },
      add(name) {
        if (name && !this.modelValue.includes(name)) this.$emit("update:modelValue", [...this.modelValue, name]);
        this.search = "";
      },
      remove(name) {
        this.$emit("update:modelValue", this.modelValue.filter((tag) => tag !== name));
      },
      handleEscape(event) {
        if (event.key !== "Escape" || !this.open) return;
        this.open = false;
        event.stopPropagation();
      }
    },
    mounted() {
      window.addEventListener("keydown", this.handleEscape, true);
    },
    beforeUnmount() {
      window.removeEventListener("keydown", this.handleEscape, true);
    },
    template: `
    <label :class="['tk-field', { 'span-2': wide }]">
  <span class="tk-field__label">Etiquetas</span>
  <input class="tk-input tag-manager-source" data-tag-manager-bound="true" readonly>
  <div class="tag-manager-chip-input">
    <span v-for="tag in modelValue" :key="tag" class="tag-manager-chip"><span class="tag-manager-chip-dot"
        :style="{ '--tag-color': tagColor(tag) }"></span>{{ tag }}<button type="button" :aria-label="'Retirar ' + tag"
        :disabled="busy" @click="remove(tag)"><tk-icon name="x-mark" /></button></span>
    <span v-show="!modelValue.length">Sin etiquetas</span>
    <button class="tk-icon-button tk-icon-button--compact tag-manager-add" type="button" aria-label="A\xF1adir etiqueta"
      :disabled="busy" @click.stop="open = !open"><tk-icon name="plus" /></button>
    <div v-show="open" class="tag-manager-picker" role="dialog" aria-label="Seleccionar etiqueta existente" @click.stop>
      <input v-model="search" class="tk-input tag-manager-picker-search" type="search" placeholder="Buscar etiquetas"
        aria-label="Buscar etiquetas existentes">
      <div class="tag-manager-picker-results">
        <button v-for="tag in availableTags" :key="tag.name" class="tag-manager-picker-option" type="button"
          :aria-label="'A\xF1adir ' + tag.name" @click="add(tag.name)"><span class="tag-manager-chip-dot"
            :style="{ '--tag-color': tagColor(tag.name) }"></span>{{ tag.name }}</button>
        <div v-show="!availableTags.length" class="tag-manager-picker-empty">No hay etiquetas disponibles.</div>
      </div>
    </div>
  </div>
</label>`
  };

  // src/task-manager/components/new-task-dialog.js
  var NewTaskDialog = {
    name: "NewTaskDialog",
    components: { TagPickerField },
    mixins: [dialogFocus],
    props: {
      open: { type: Boolean, required: true },
      draft: { type: Object, required: true },
      tasks: { type: Array, required: true },
      tags: { type: Array, required: true },
      colors: { type: Array, required: true },
      busy: { type: Boolean, required: true }
    },
    emits: ["close", "update-draft", "submit"],
    methods: {
      update(field, value) {
        this.$emit("update-draft", { ...this.draft, [field]: value });
      }
    },
    template: `
    <div v-show="open" class="modal-layer" role="dialog" aria-modal="true" aria-labelledby="new-task-title"
  @click.self="$emit('close')">
  <form class="tk-modal tk-modal--wide" ref="dialogRoot" tabindex="-1" @keydown.tab="trapDialogTab"
    @submit.prevent="$emit('submit')">
    <header class="tk-modal__header">
      <h2 id="new-task-title">{{ draft.parent ? 'Nueva subtarea' : 'Nueva tarea' }}</h2><button class="tk-icon-button"
        type="button" aria-label="Cerrar" @click="$emit('close')"><tk-icon name="x-mark" /></button>
    </header>
    <div class="tk-modal__body">
      <div class="form-grid"><label class="tk-field span-2"><span class="tk-field__label">T\xEDtulo</span><input
            :value="draft.title" class="tk-input" required
            @input="update('title', $event.target.value)" /></label><label class="tk-field"><span
            class="tk-field__label">Prioridad</span><select :value="draft.priority" class="tk-select"
            @change="update('priority', $event.target.value)">
            <option value="0">Cr\xEDtica</option>
            <option value="1">Alta</option>
            <option value="2">Media</option>
            <option value="3">Baja</option>
          </select></label><label class="tk-field"><span class="tk-field__label">Tarea padre</span><select
            :value="draft.parent" class="tk-select" @change="update('parent', $event.target.value)">
            <option value="">Sin padre</option>
            <option v-for="task in tasks" :key="task.id" :value="task.id">{{ task.id }} \xB7 {{ task.title }}</option>
          </select></label><label class="tk-field span-2"><span class="tk-field__label">Descripci\xF3n</span><textarea
            :value="draft.description" class="tk-textarea"
            @input="update('description', $event.target.value)"></textarea></label>
        <tag-picker-field :model-value="draft.tags" :tags="tags" :colors="colors" :busy="busy" :reset-key="open" wide
          @update:model-value="update('tags', $event)"></tag-picker-field>
      </div>
    </div>
    <footer class="tk-modal__footer"><button class="tk-button tk-button--ghost" type="button" :disabled="busy"
        @click="$emit('close')">Cancelar</button><button class="tk-button tk-button--primary" type="submit"
        :disabled="busy">{{ busy ? 'Creando\u2026' : 'Crear tarea' }}</button></footer>
  </form>
</div>`
  };

  // src/task-manager/components/task-filters.js
  var TaskFilters = {
    name: "TaskFilters",
    props: {
      unavailable: { type: Boolean, required: true },
      errorMessage: { type: String, required: true },
      refreshing: { type: Boolean, required: true },
      coldTasks: { type: Array, required: true },
      taskSearchQuery: { type: String, required: true },
      advancedFiltersOpen: { type: Boolean, required: true },
      hasAdvancedFilters: { type: Boolean, required: true },
      advancedStatuses: { type: Array, required: true },
      advancedPriorities: { type: Array, required: true },
      advancedTags: { type: Array, required: true },
      tagCatalog: { type: Array, required: true },
      activeTaskFilter: { type: String, required: true },
      currentPeriod: { type: String, required: true },
      quickTagFilter: { type: String, required: true },
      taskChipsVisible: { type: Boolean, required: true },
      visibleCount: { type: Number, required: true },
      listMenuOpen: { type: Boolean, required: true },
      tagColor: { type: Function, required: true }
    },
    emits: ["refresh", "dismiss-error", "update-search", "toggle-filters", "close-filters", "clear-filters", "update-statuses", "update-priorities", "update-tags", "update-quick-filter", "update-active-filter", "update-current-period", "toggle-chips", "toggle-list-menu", "set-expanded"],
    watch: {
      listMenuOpen(value) {
        if (value) this.$nextTick(() => this.$refs.listMenu?.focus());
      }
    },
    computed: {
      // El aviso no puede acaparar la pantalla: las primeras tareas y el resto
      // contado. El orden ya viene de la más fría a la menos.
      coldNotice() {
        const shown = this.coldTasks.slice(0, 3).map((task) => `${task.title} (sin movimiento desde hace ${task.days} ${task.days === 1 ? "d\xEDa" : "d\xEDas"})`);
        const rest = this.coldTasks.length - shown.length;
        return `${shown.join("; ")}${rest > 0 ? `; +${rest} m\xE1s` : ""}.`;
      }
    },
    methods: {
      toggleValue(values, value, checked, eventName) {
        this.$emit(eventName, checked ? [...values, value] : values.filter((item) => item !== value));
      }
    },
    template: `
    <div class="task-explorer__toolbar">
  <div v-show="unavailable" class="workspace-notice workspace-notice--unavailable" role="status"><strong>TK no est\xE1
      disponible</strong><span>La capacidad de tareas no est\xE1 habilitada en este anfitri\xF3n.</span><button
      class="tk-button tk-button--small tk-button--secondary" type="button"
      @click="$emit('refresh')">Reintentar</button></div>
  <div v-show="errorMessage" class="workspace-notice workspace-notice--error" role="alert"><strong>No se pudo completar
      la operaci\xF3n</strong><span>{{ errorMessage }}</span><button class="tk-icon-button tk-icon-button--compact"
      type="button" aria-label="Cerrar error" title="Cerrar error"
      @click="$emit('dismiss-error')"><tk-icon name="x-mark" /></button></div>
  <div v-show="refreshing" class="workspace-notice workspace-notice--loading" role="status"><strong>Actualizando
      datos</strong><span>Se mantiene visible el \xFAltimo estado confirmado.</span></div>
  <div v-if="coldTasks.length" class="workspace-notice workspace-notice--warning" role="status"><strong>Tablero
      fr\xEDo</strong><span>{{ coldNotice }}</span></div>
  <div class="task-search-row">
    <label><span class="sr-only">Buscar tareas</span><input :value="taskSearchQuery" class="tk-input" type="search"
        placeholder="Buscar por t\xEDtulo, ID, etiqueta, descripci\xF3n, notas o comentarios" @input="$emit('update-search', $event.target.value)" /></label>
    <button class="tk-icon-button task-filter-button" type="button" aria-label="Filtros" title="Mostrar filtros avanzados" :aria-expanded="advancedFiltersOpen"
      @click.stop="$emit('toggle-filters')"><tk-icon name="funnel" /><span v-if="hasAdvancedFilters" class="advanced-filter-indicator" aria-hidden="true"></span></button>
    <div class="menu-anchor"><button class="tk-icon-button" type="button" aria-label="Opciones del listado"
        title="Opciones del listado" aria-haspopup="menu" :aria-expanded="listMenuOpen"
        @click.stop="$emit('toggle-list-menu')"><tk-icon name="ellipsis-vertical" /></button>
      <div v-show="listMenuOpen" ref="listMenu" tabindex="-1" class="tk-dropdown tk-dropdown--anchored is-open" role="menu" @click.stop><button
          class="tk-dropdown__item" type="button" role="menuitem" title="Expandir todas las tareas" @click="$emit('set-expanded', true)">Expandir todo</button><button
          class="tk-dropdown__item" type="button" role="menuitem" title="Plegar todas las tareas" @click="$emit('set-expanded', false)">Plegar todo</button></div>
    </div>
  </div>
  <section v-show="advancedFiltersOpen" class="task-filter-popover" aria-label="Filtros avanzados" @click.stop>
    <div class="task-filter-popover__header"><strong>Filtros</strong>
      <div class="task-filter-popover__actions"><button v-show="hasAdvancedFilters" class="task-filter-clear"
          type="button" title="Limpiar filtros" @click="$emit('clear-filters')"><tk-icon
            name="arrow-path" /><span>Limpiar
            filtros</span></button><button class="tk-icon-button tk-icon-button--compact" type="button"
          aria-label="Cerrar filtros" title="Cerrar filtros"
          @click="$emit('close-filters')"><tk-icon name="x-mark" /></button></div>
    </div>
    <fieldset>
      <legend>Estado</legend><label
        v-for="item in [['open', 'Pendiente'], ['in_progress', 'En curso'], ['done', 'Hecha'], ['blocked', 'Bloqueada']]"
        :key="item[0]"><input type="checkbox" :value="item[0]" :checked="advancedStatuses.includes(item[0])"
          @change="toggleValue(advancedStatuses, item[0], $event.target.checked, 'update-statuses')" />
        {{ item[1] }}</label>
    </fieldset>
    <fieldset>
      <legend>Prioridad</legend><label v-for="item in [['0', 'Cr\xEDtica'], ['1', 'Alta'], ['2', 'Media'], ['3', 'Baja']]"
        :key="item[0]"><input type="checkbox" :value="item[0]" :checked="advancedPriorities.includes(item[0])"
          @change="toggleValue(advancedPriorities, item[0], $event.target.checked, 'update-priorities')" />
        {{ item[1] }}</label>
    </fieldset>
    <fieldset>
      <legend>Etiquetas</legend><label v-for="tag in tagCatalog" :key="tag.name"
        :style="{ '--filter-option-color': tagColor(tag.name) }"><input type="checkbox" :value="tag.name"
          :checked="advancedTags.includes(tag.name)"
          @change="toggleValue(advancedTags, tag.name, $event.target.checked, 'update-tags')" /> {{ tag.name }}</label>
    </fieldset>
    <p class="task-filter-popover__summary">{{ visibleCount }} {{ visibleCount === 1 ? 'resultado' : 'resultados' }}</p>
  </section>
  <nav class="tk-tabs task-filter-tabs" aria-label="Filtros r\xE1pidos"><button
      v-for="item in [['all', 'Todas'], ['current', 'Actuales'], ['open', 'Pendientes'], ['in_progress', 'En curso'], ['blocked', 'Bloqueadas'], ['done', 'Hechas']]"
      :key="item[0]" class="tk-tab" type="button" :title="item[1]" :aria-pressed="activeTaskFilter === item[0]"
      @click="$emit('update-active-filter', item[0])">{{ item[1] }}</button></nav>
  <nav v-show="activeTaskFilter === 'current'" class="tk-tabs current-period-tabs" aria-label="Periodo de tareas actuales"><button
      v-for="item in [['today', 'Hoy'], ['week', 'Semana'], ['month', 'Mes'], ['year', 'A\xF1o']]"
      :key="item[0]" class="tk-tab" type="button" :title="item[1]" :aria-pressed="currentPeriod === item[0]"
      @click="$emit('update-current-period', item[0])">{{ item[1] }}</button></nav>
  <div class="tag-manager-filter-controls"><select :value="quickTagFilter" class="tk-select tag-manager-filter"
      aria-label="Filtrar por etiqueta" @change="$emit('update-quick-filter', $event.target.value)">
      <option value="">Todas las etiquetas</option>
      <option v-for="tag in tagCatalog" :key="tag.name" :value="tag.name">{{ tag.name }}</option>
    </select><button class="tk-icon-button tk-icon-button--compact tag-manager-visibility" type="button"
      aria-label="Mostrar u ocultar etiquetas" title="Mostrar u ocultar etiquetas" :aria-pressed="taskChipsVisible"
      @click="$emit('toggle-chips')"><tk-icon v-if="taskChipsVisible" name="eye" /><tk-icon v-else
        name="eye-slash" /></button></div>
</div>`
  };

  // src/task-manager/components/overflow-menu.js
  var OverflowMenu = {
    name: "OverflowMenu",
    props: {
      open: { type: Boolean, required: true },
      items: { type: Array, required: true },
      label: { type: String, default: "M\xE1s opciones" },
      menuClass: { type: String, default: "tk-dropdown--anchored" },
      compact: { type: Boolean, default: true }
    },
    emits: ["toggle", "select"],
    watch: {
      open(value) {
        if (value) this.$nextTick(() => this.$refs.menu?.focus());
      }
    },
    template: `
    <div class="menu-anchor">
  <button :class="['tk-icon-button', { 'tk-icon-button--compact': compact }]" type="button" :aria-label="label"
    aria-haspopup="menu" :aria-expanded="open" @click.stop="$emit('toggle')"><tk-icon
      name="ellipsis-vertical" /></button>
  <div v-show="open" ref="menu" tabindex="-1" :class="['tk-dropdown', menuClass, 'is-open']" role="menu" @click.stop>
    <template v-for="item in items" :key="item.id">
      <div v-if="item.separatorBefore" class="tk-dropdown__separator"></div>
      <button :class="['tk-dropdown__item', { 'tk-dropdown__item--danger': item.danger }]" type="button" role="menuitem"
        :disabled="item.disabled" @click="$emit('select', item.id)">{{ item.label }}</button>
    </template>
  </div>
</div>`
  };

  // src/task-manager/components/task-row.js
  var TaskRow = {
    name: "TaskRow",
    components: { OverflowMenu },
    props: {
      task: { type: Object, required: true },
      selected: { type: Boolean, required: true },
      expanded: { type: Boolean, required: true },
      chipsVisible: { type: Boolean, required: true },
      menuOpen: { type: Boolean, required: true },
      busy: { type: Function, required: true },
      tagColor: { type: Function, required: true }
    },
    emits: ["select", "toggle-expanded", "update-field", "toggle-menu", "menu-action"],
    computed: {
      menuItems() {
        return [
          { id: "edit", label: "Editar" },
          { id: "copy", label: "Copiar ID" },
          { id: "close", label: "Cerrar tarea", danger: true, disabled: this.task.status === "done" || this.busy(`close:${this.task.id}`) }
        ];
      }
    },
    template: `
    <article :class="['task-row', 'task-row--level-' + Math.min(task.level, 3), { 'is-selected': selected }]"
  role="treeitem" tabindex="0" :aria-level="task.level" :aria-selected="selected" @click="$emit('select', task.id)"
  @keydown.enter.self.prevent="$emit('select', task.id)" @keydown.space.self.prevent="$emit('select', task.id)">
  <button v-show="task.children" class="tree-expander" type="button" :aria-expanded="expanded"
    :aria-label="expanded ? 'Contraer subtareas' : 'Expandir subtareas'"
    @click.stop="$emit('toggle-expanded', task.id)"><tk-icon name="chevron-right" /></button>
  <button v-show="!task.children" class="tree-expander tree-expander--placeholder" type="button" disabled tabindex="-1"
    aria-hidden="true"><tk-icon name="chevron-right" /></button>
  <div :class="['task-type-icon', 'task-type-icon--' + task.kind]">{{ task.icon }}</div>
  <div class="task-row__copy">
    <div class="task-row__title">{{ task.title }}</div>
    <div class="task-row__meta"><span>{{ task.id }}</span><span>{{ task.type }}</span></div>
    <div v-show="chipsVisible" class="tag-manager-task-chips"><span v-for="tag in task.chips" :key="tag"
        class="tag-manager-chip"><span class="tag-manager-chip-dot"
          :style="{ '--tag-color': tagColor(tag) }"></span>{{ tag }}</span></div>
  </div>
  <select class="tk-select tk-select--compact tk-priority-select task-row__priority" aria-label="Prioridad"
    :value="task.priority" :disabled="busy('priority:' + task.id)" @click.stop
    @change="$emit('update-field', task.id, 'priority', $event.target.value)">
    <option value="0">Cr\xEDtica</option>
    <option value="1">Alta</option>
    <option value="2">Media</option>
    <option value="3">Baja</option>
  </select>
  <div class="task-row__meta-column"><select class="tk-select tk-select--compact" :aria-label="'Estado de ' + task.id"
      :title="task.status === 'blocked' ? 'Retira las dependencias para cambiar el estado' : ''" :value="task.status"
      :disabled="task.status === 'blocked' || busy('status:' + task.id) || busy('close:' + task.id)" @click.stop
      @change="$emit('update-field', task.id, 'status', $event.target.value)">
      <option value="open">Pendiente</option>
      <option value="in_progress">En curso</option>
      <option value="done">Hecha</option>
      <option v-show="task.status === 'blocked'" value="blocked" disabled>Bloqueada</option>
    </select></div>
  <overflow-menu class="task-row__menu" :open="menuOpen" :items="menuItems" @toggle="$emit('toggle-menu', task.id)"
    @select="$emit('menu-action', $event, task.id)"></overflow-menu>
</article>`
  };

  // src/task-manager/components/task-list.js
  var TaskList = {
    name: "TaskList",
    components: { TaskFilters, TaskRow },
    props: {
      loading: { type: Boolean, required: true },
      hasTasks: { type: Boolean, required: true },
      tasks: { type: Array, required: true },
      activeCount: { type: Number, required: true },
      selectedId: { type: String, required: true },
      expandedIds: { type: Array, required: true },
      chipsVisible: { type: Boolean, required: true },
      openMenuId: { type: String, required: true },
      busy: { type: Function, required: true },
      tagColor: { type: Function, required: true },
      filterProps: { type: Object, required: true }
    },
    emits: ["refresh", "dismiss-error", "update-search", "toggle-filters", "close-filters", "clear-filters", "update-statuses", "update-priorities", "update-tags", "update-quick-filter", "update-active-filter", "update-current-period", "toggle-chips", "toggle-list-menu", "set-expanded", "select", "toggle-expanded", "update-field", "toggle-menu", "menu-action"],
    template: `
    <aside class="task-explorer" aria-label="Explorador de tareas">
  <task-filters v-bind="filterProps" @refresh="$emit('refresh')" @dismiss-error="$emit('dismiss-error')"
    @update-search="$emit('update-search', $event)" @toggle-filters="$emit('toggle-filters')"
    @close-filters="$emit('close-filters')" @clear-filters="$emit('clear-filters')"
    @update-statuses="$emit('update-statuses', $event)" @update-priorities="$emit('update-priorities', $event)"
    @update-tags="$emit('update-tags', $event)" @update-quick-filter="$emit('update-quick-filter', $event)"
    @update-active-filter="$emit('update-active-filter', $event)" @update-current-period="$emit('update-current-period', $event)"
    @toggle-chips="$emit('toggle-chips')"
    @toggle-list-menu="$emit('toggle-list-menu')" @set-expanded="$emit('set-expanded', $event)"></task-filters>
  <div class="task-tree-scroll">
    <div v-show="loading" class="skeleton-stack" aria-label="Cargando tareas">
      <div class="skeleton"></div>
      <div class="skeleton skeleton--short"></div>
      <div class="skeleton"></div>
    </div>
    <div v-show="!loading" class="task-tree" role="tree">
      <task-row v-for="task in tasks" :key="task.id" :task="task" :selected="selectedId === task.id"
        :expanded="expandedIds.includes(task.id)" :chips-visible="chipsVisible" :menu-open="openMenuId === task.id"
        :busy="busy" :tag-color="tagColor" @select="$emit('select', $event)"
        @toggle-expanded="$emit('toggle-expanded', $event)"
        @update-field="(id, field, value) => $emit('update-field', id, field, value)"
        @toggle-menu="$emit('toggle-menu', $event)" @menu-action="(action, id) => $emit('menu-action', action, id)">
      </task-row>
      <div v-show="!tasks.length && !loading" class="empty-state">
        <div>
          <div class="empty-state__icon"><tk-icon name="magnifying-glass" /></div>
          <h3>{{ hasTasks ? 'Sin resultados' : 'No hay tareas' }}</h3>
          <p>{{ hasTasks ? 'No hay tareas que coincidan con los filtros activos.' : 'Crea una tarea para comenzar.' }}
          </p>
        </div>
      </div>
    </div>
  </div>
  <footer class="task-explorer__footer"><span>{{ tasks.length }} elementos visibles</span><span>{{ activeCount }}
      activas</span></footer>
</aside>`
  };

  // src/shared/tasks/task-badge.js
  var TaskBadge = {
    name: "TaskBadge",
    props: {
      kind: { type: String, required: true },
      value: { type: [String, Number], required: true }
    },
    computed: {
      label() {
        return this.kind === "status" ? statusLabel(this.value) : priorityLabel(this.value);
      },
      badgeClass() {
        return this.kind === "status" ? statusBadgeClass(this.value) : `tk-priority--${this.value}`;
      }
    },
    template: `<span :class="['tk-badge', badgeClass]">{{ label }}</span>`
  };

  // src/shared/tasks/rich-text.js
  var ALLOWED_TAGS = /* @__PURE__ */ new Set([
    "A",
    "B",
    "BLOCKQUOTE",
    "BR",
    "CODE",
    "DD",
    "DEL",
    "DIV",
    "DL",
    "DT",
    "EM",
    "H1",
    "H2",
    "H3",
    "H4",
    "H5",
    "H6",
    "HR",
    "I",
    "LI",
    "MARK",
    "OL",
    "P",
    "PRE",
    "S",
    "SMALL",
    "SPAN",
    "STRIKE",
    "STRONG",
    "SUB",
    "SUP",
    "TABLE",
    "TBODY",
    "TD",
    "TH",
    "THEAD",
    "TR",
    "U",
    "UL"
  ]);
  var DROPPED_TAGS = /* @__PURE__ */ new Set([
    "APPLET",
    "BASE",
    "EMBED",
    "FORM",
    "IFRAME",
    "INPUT",
    "LINK",
    "MATH",
    "META",
    "NOSCRIPT",
    "OBJECT",
    "SCRIPT",
    "STYLE",
    "SVG",
    "TEMPLATE",
    "TEXTAREA"
  ]);
  var ALLOWED_ATTRIBUTES = /* @__PURE__ */ new Set(["href", "title"]);
  var SAFE_URL = /^(https?:|mailto:|#|\/)/i;
  var parser = null;
  function clean(node) {
    for (const child of Array.from(node.childNodes)) {
      if (child.nodeType === Node.TEXT_NODE) continue;
      if (child.nodeType !== Node.ELEMENT_NODE) {
        child.remove();
        continue;
      }
      const tag = child.tagName.toUpperCase();
      if (DROPPED_TAGS.has(tag)) {
        child.remove();
        continue;
      }
      if (!ALLOWED_TAGS.has(tag)) {
        clean(child);
        while (child.firstChild) child.parentNode.insertBefore(child.firstChild, child);
        child.remove();
        continue;
      }
      for (const attribute of Array.from(child.attributes)) {
        const name = attribute.name.toLowerCase();
        const safe = ALLOWED_ATTRIBUTES.has(name) && (name !== "href" || SAFE_URL.test(attribute.value.trim()));
        if (!safe) child.removeAttribute(attribute.name);
      }
      clean(child);
    }
  }
  function isPlainText(value) {
    const source = String(value ?? "");
    return source !== "" && !/<[a-z][a-z0-9-]*[\s/>]/i.test(source);
  }
  function richTextHtml(value) {
    const source = String(value ?? "");
    if (!source) return "";
    parser = parser || new DOMParser();
    const doc = parser.parseFromString(source, "text/html");
    clean(doc.body);
    return doc.body.innerHTML;
  }

  // src/task-manager/components/task-inspector.js
  var TaskInspector = {
    name: "TaskInspector",
    components: { OverflowMenu, TagPickerField, TaskBadge },
    props: {
      mobileOpen: { type: Boolean, required: true },
      hasSelection: { type: Boolean, required: true },
      task: { type: Object, required: true },
      parentTask: { type: Object, default: null },
      activeTab: { type: String, required: true },
      menuOpen: { type: Boolean, required: true },
      busy: { type: Function, required: true },
      statusBadgeClass: { type: Function, required: true },
      statusLabel: { type: Function, required: true },
      priorityLabel: { type: Function, required: true },
      subtasks: { type: Array, required: true },
      parentOptions: { type: Array, required: true },
      blockers: { type: Array, required: true },
      dependents: { type: Array, required: true },
      comments: { type: Array, required: true },
      editDraft: { type: Object, required: true },
      tagCatalog: { type: Array, required: true },
      colors: { type: Array, required: true },
      dependencyDraft: { type: String, required: true },
      commentDraft: { type: String, required: true },
      editingCommentId: { type: Number, default: null },
      editingCommentDraft: { type: String, default: "" }
    },
    emits: ["update:mobileOpen", "update:activeTab", "update:editDraft", "update:dependencyDraft", "update:commentDraft", "update:editingCommentDraft", "action"],
    computed: {
      menuItems() {
        return [
          { id: "copy", label: "Copiar ID" },
          { id: "subtask", label: "Crear subtarea" }
        ];
      }
    },
    methods: {
      updateDraft(field, value) {
        this.$emit("update:editDraft", { ...this.editDraft, [field]: value });
      },
      emitAction(type, payload = {}) {
        this.$emit("action", { type, payload });
      },
      selectMenuAction(action) {
        if (action === "copy") this.emitAction("task.copy");
        if (action === "subtask") this.emitAction("subtask.open");
      },
      selectTab(tab) {
        if (tab === "edit") this.emitAction("edit.open");
        else this.$emit("update:activeTab", tab);
      },
      // El texto libre se dibuja con su formato: el saneado lo hace rich-text.js.
      descriptionHtml() {
        return richTextHtml(this.task.description) || "Sin descripci\xF3n.";
      },
      notesHtml() {
        return richTextHtml(this.task.notes) || "Sin notas.";
      },
      commentHtml(body) {
        return richTextHtml(body);
      },
      plainText(value) {
        return isPlainText(value);
      },
      commentMeta(comment) {
        return ["#" + comment.id, comment.createdAt && new Date(comment.createdAt).toLocaleString("es")].filter(Boolean).join(" \xB7 ");
      }
    },
    template: `
    <section :class="['task-inspector', { 'is-mobile-open': mobileOpen }]">
  <div v-show="!hasSelection" class="empty-state inspector-empty">
    <div>
      <div class="empty-state__icon">T</div>
      <h3>Selecciona una tarea</h3>
      <p>El detalle confirmado por TK aparecer\xE1 aqu\xED.</p>
    </div>
  </div>
  <div v-show="hasSelection" class="task-inspector-content">
    <header class="task-inspector__header">
      <div class="task-inspector__heading-row">
        <div class="task-inspector__heading"><button class="task-mobile-back tk-button tk-button--ghost" type="button"
            @click="$emit('update:mobileOpen', false)">Volver a tareas</button>
          <div class="task-inspector__kicker">{{ task.id }} \xB7 {{ task.type }}<template v-if="parentTask"> de <button
              class="task-parent-link" type="button" :title="parentTask.title"
              @click="emitAction('task.select', { id: parentTask.id })">{{ parentTask.id }}</button></template></div>
          <h1 class="task-inspector__title">{{ task.title }}</h1>
          <div class="task-inspector__description tk-rich-text tk-scrollable-text"
            :class="{ 'tk-rich-text--plain': plainText(task.description) }" v-html="descriptionHtml()"></div>
        </div>
        <div class="task-inspector__actions"><button class="tk-button tk-button--secondary tk-button--compact"
            type="button" title="Editar tarea" @click="emitAction('edit.open')"><tk-icon
              name="pencil" />Editar</button><button class="tk-button tk-button--danger tk-button--compact" type="button"
            title="Cerrar tarea" :disabled="task.status === 'done' || busy('close:' + task.id)"
            @click="emitAction('task.close')"><tk-icon name="check" />Cerrar tarea</button><task-badge kind="status" :value="task.status"></task-badge>
          <overflow-menu :open="menuOpen" :items="menuItems" label="M\xE1s opciones de la tarea"
            menu-class="tk-dropdown--inspector" :compact="false" @toggle="emitAction('menu.toggle')"
            @select="selectMenuAction"></overflow-menu>
        </div>
      </div>
      <div class="task-property-bar"><label class="task-property-control"><span
            class="task-property-control__label">Prioridad</span><select class="tk-select" :value="task.priority"
            :disabled="busy('priority:' + task.id)"
            @change="emitAction('task.update-field', { id: task.id, field: 'priority', value: $event.target.value })">
            <option value="0">Cr\xEDtica</option>
            <option value="1">Alta</option>
            <option value="2">Media</option>
            <option value="3">Baja</option>
          </select></label><label class="task-property-control"><span
            class="task-property-control__label">Estado</span><select class="tk-select"
            :title="task.status === 'blocked' ? 'Retira las dependencias para cambiar el estado' : ''"
            :value="task.status"
            :disabled="task.status === 'blocked' || busy('status:' + task.id) || busy('close:' + task.id)"
            @change="emitAction('task.update-field', { id: task.id, field: 'status', value: $event.target.value })">
            <option value="open">Pendiente</option>
            <option value="in_progress">En curso</option>
            <option value="done">Hecha</option>
            <option v-show="task.status === 'blocked'" value="blocked" disabled>Bloqueada</option>
          </select></label>
        <div class="task-property-control"><span class="task-property-control__label">Responsable</span><span
            class="task-property-control__value">{{ task.assignee || 'Sin asignar' }}</span></div>
      </div>
    </header>
    <nav class="tk-tabs task-inspector__tabs" aria-label="Detalles de la tarea"><button
        v-for="item in [['detail', 'Detalle'], ['edit', 'Editar'], ['subtasks', 'Subtareas'], ['dependencies', 'Dependencias'], ['comments', 'Comentarios']]"
        :key="item[0]" class="tk-tab" type="button" :aria-pressed="activeTab === item[0]"
        @click="selectTab(item[0])">{{ item[1] }}</button></nav>
    <div class="task-inspector__scroll">
      <div v-show="activeTab === 'detail'" class="tk-panel-stack">
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Propiedades</h3>
          </header>
          <div class="tk-panel__content">
            <div class="property-grid">
              <div class="property-card">
                <div class="property-card__label">Tipo</div>
                <div class="property-card__value">{{ task.type }}<template v-if="parentTask"> de <button
                    class="task-parent-link" type="button" :title="parentTask.title"
                    @click="emitAction('task.select', { id: parentTask.id })">{{ parentTask.id }}</button></template></div>
              </div>
              <div class="property-card">
                <div class="property-card__label">Etiquetas</div>
                <div class="property-card__value">{{ task.chips.join(', ') || 'Sin etiquetas' }}</div>
              </div>
              <div class="property-card">
                <div class="property-card__label">Bloqueos</div>
                <div class="property-card__value">{{ blockers.length }}</div>
              </div>
              <div class="property-card">
                <div class="property-card__label">Subtareas</div>
                <div class="property-card__value">{{ subtasks.length }}</div>
              </div>
            </div>
          </div>
        </section>
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Notas</h3>
          </header>
          <div class="tk-panel__content tk-rich-text tk-scrollable-text"
            :class="{ 'tk-rich-text--plain': plainText(task.notes) }" v-html="notesHtml()"></div>
        </section>
      </div>
      <div v-show="activeTab === 'edit'" class="tk-panel-stack">
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Editar tarea</h3>
          </header>
          <form class="tk-panel__content form-grid" @submit.prevent="emitAction('edit.save')"><label
              class="tk-field span-2"><span class="tk-field__label">T\xEDtulo</span><input :value="editDraft.title"
                class="tk-input" required @input="updateDraft('title', $event.target.value)" /></label><label
              class="tk-field span-2"><span class="tk-field__label">Descripci\xF3n</span><textarea
                :value="editDraft.description" class="tk-textarea"
                @input="updateDraft('description', $event.target.value)"></textarea></label><label
              class="tk-field span-2"><span class="tk-field__label">Notas</span><textarea :value="editDraft.notes"
                class="tk-textarea" @input="updateDraft('notes', $event.target.value)"></textarea></label><label
              class="tk-field"><span class="tk-field__label">Responsable</span><input :value="editDraft.assignee"
                class="tk-input" placeholder="Sin asignar"
                @input="updateDraft('assignee', $event.target.value)" /></label><label class="tk-field"><span
              class="tk-field__label">Tarea padre</span><select :value="editDraft.parent || ''" class="tk-select"
              :disabled="busy('edit:' + task.id)" @change="updateDraft('parent', $event.target.value)">
              <option value="">Sin padre</option>
              <option v-for="option in parentOptions" :key="option.id" :value="option.id">{{ option.id }} \xB7
                {{ option.title }}</option>
            </select></label>
            <tag-picker-field :model-value="editDraft.tags" :tags="tagCatalog" :colors="colors"
              :busy="busy('edit:' + task.id)" :reset-key="task.id + ':' + activeTab" wide
              @update:model-value="updateDraft('tags', $event)"></tag-picker-field>
            <div class="span-2 task-form-actions"><button class="tk-button tk-button--ghost" type="button"
                :disabled="busy('edit:' + task.id)" @click="emitAction('edit.cancel')">Cancelar</button><button
                class="tk-button tk-button--primary" type="submit"
                :disabled="busy('edit:' + task.id)">{{ busy('edit:' + task.id) ? 'Guardando\u2026' : 'Guardar cambios' }}</button>
            </div>
          </form>
        </section>
      </div>
      <div v-show="activeTab === 'subtasks'" class="tk-panel-stack">
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Subtareas</h3><span></span><button class="tk-button tk-button--small tk-button--secondary" type="button"
              @click="emitAction('subtask.open')">Nueva subtarea</button>
          </header>
          <div class="tk-panel__content">
            <div class="subtask-list"><button v-for="subtask in subtasks" :key="subtask.id" class="subtask-list__item"
                type="button" @click="emitAction('task.select', { id: subtask.id })"><span
                  class="subtask-list__type">{{ subtask.icon }}</span><span class="subtask-list__copy"><span
                    class="subtask-list__title">{{ subtask.title }}</span><span
                    class="subtask-list__meta">{{ subtask.id }}</span></span><span
                  class="subtask-list__priority">{{ priorityLabel(subtask.priority) }}</span><span
                  class="subtask-list__status">{{ statusLabel(subtask.status) }}</span></button>
              <p v-show="!subtasks.length">Esta tarea no tiene subtareas.</p>
            </div>
          </div>
        </section>
      </div>
      <div v-show="activeTab === 'dependencies'" class="tk-panel-stack">
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Dependencias</h3>
          </header>
          <div class="tk-panel__content">
            <form class="task-search-row" @submit.prevent="emitAction('dependency.add')"><input :value="dependencyDraft"
                class="tk-input" placeholder="ID de tarea bloqueante" required
                @input="$emit('update:dependencyDraft', $event.target.value)" /><button
                class="tk-button tk-button--secondary" type="submit"
                :disabled="busy('dep-add:' + task.id)">{{ busy('dep-add:' + task.id) ? 'A\xF1adiendo\u2026' : 'A\xF1adir' }}</button>
            </form>
            <div class="task-panel-stack">
              <h4>Bloqueada por</h4>
              <article v-for="blocker in blockers" :key="blocker.id" class="relation-row">
                <div><strong>{{ blocker.id }}</strong><span>{{ blocker.title }}</span></div><button
                  class="tk-button tk-button--small tk-button--danger" type="button"
                  :disabled="busy('dep-remove:' + blocker.id)"
                  @click="emitAction('dependency.remove-blocker', { id: blocker.id })">Retirar</button>
              </article>
              <p v-show="!blockers.length">Sin tareas bloqueantes.</p>
              <h4>Bloquea a</h4>
              <article v-for="dependent in dependents" :key="dependent.id" class="relation-row">
                <div><strong>{{ dependent.id }}</strong><span>{{ dependent.title }}</span></div><button
                  class="tk-button tk-button--small tk-button--danger" type="button"
                  :disabled="busy('dep-remove:' + dependent.id)"
                  @click="emitAction('dependency.remove-dependent', { id: dependent.id })">Retirar</button>
              </article>
              <p v-show="!dependents.length">No bloquea otras tareas.</p>
            </div>
          </div>
        </section>
      </div>
      <div v-show="activeTab === 'comments'" class="tk-panel-stack">
        <section class="tk-panel">
          <header class="tk-panel__header">
            <h3>Comentarios</h3>
          </header>
          <div class="tk-panel__content">
            <div class="comment-list">
              <article v-for="comment in comments" :key="comment.id" class="comment-card">
                <template v-if="editingCommentId === comment.id">
                  <textarea :value="editingCommentDraft" class="tk-textarea"
                    @input="$emit('update:editingCommentDraft', $event.target.value)"></textarea>
                  <div class="comment-card__actions">
                    <button class="tk-button tk-button--primary" type="button"
                      :disabled="busy('comment-edit:' + task.id) || !editingCommentDraft.trim()"
                      @click="emitAction('comment.edit.save')">{{ busy('comment-edit:' + task.id) ? 'Guardando\u2026' : 'Guardar' }}</button>
                    <button class="tk-button" type="button" :disabled="busy('comment-edit:' + task.id)"
                      @click="emitAction('comment.edit.cancel')">Cancelar</button>
                  </div>
                </template>
                <template v-else>
                  <div class="tk-rich-text tk-scrollable-text"
                    :class="{ 'tk-rich-text--plain': plainText(comment.body) }" v-html="commentHtml(comment.body)"></div>
                  <div class="comment-card__meta"><span v-show="commentMeta(comment)">{{ commentMeta(comment) }}</span><button
                      class="tk-icon-button" type="button" title="Editar comentario" aria-label="Editar comentario"
                      :disabled="busy('comment-edit:' + task.id) || busy('comment:' + task.id)"
                      @click="emitAction('comment.edit.start', { comment })"><tk-icon name="pencil" /></button></div>
                </template>
              </article>
              <p v-show="!comments.length">Sin comentarios.</p>
            </div>
            <form class="comment-form" @submit.prevent="emitAction('comment.add')"><textarea :value="commentDraft"
                class="tk-textarea" placeholder="A\xF1adir comentario" required
                @input="$emit('update:commentDraft', $event.target.value)"></textarea><button
                class="tk-button tk-button--primary" type="submit"
                :disabled="busy('comment:' + task.id)">{{ busy('comment:' + task.id) ? 'A\xF1adiendo\u2026' : 'A\xF1adir comentario' }}</button>
            </form>
          </div>
        </section>
      </div>
    </div>
  </div>
</section>`
  };

  // src/shared/tasks/markdown.js
  var ESCAPES = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
  function escapeHtml2(text) {
    return text.replace(/[&<>"']/g, (letter) => ESCAPES[letter]);
  }
  function inline(text) {
    const codes = [];
    let out = text.replace(/`([^`]+)`/g, (_2, code) => {
      codes.push(`<code>${code}</code>`);
      return `\0${codes.length - 1}\0`;
    });
    out = out.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '<a href="$2">$1</a>');
    out = out.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
    out = out.replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");
    out = out.replace(/\u0000(\d+)\u0000/g, (_2, index) => codes[Number(index)]);
    return out;
  }
  function markdownHtml(value) {
    const source = escapeHtml2(String(value ?? "").replace(/\r\n?/g, "\n"));
    const blocks = [];
    let paragraph = [];
    let list = [];
    let listOpen = false;
    const flushParagraph = () => {
      if (!paragraph.length) return;
      blocks.push(`<p>${inline(paragraph.join(" "))}</p>`);
      paragraph = [];
    };
    const flushList = () => {
      if (!listOpen) return;
      blocks.push(`<ul>${list.map((item) => `<li>${inline(item)}</li>`).join("")}</ul>`);
      list = [];
      listOpen = false;
    };
    for (const raw of source.split("\n")) {
      const line = raw.trim();
      if (!line) {
        flushParagraph();
        flushList();
        continue;
      }
      if (listOpen && /^\s+/.test(raw)) {
        list[list.length - 1] += ` ${line}`;
        continue;
      }
      if (/^[-*]\s+/.test(line)) {
        flushParagraph();
        listOpen = true;
        list.push(line.replace(/^[-*]\s+/, ""));
        continue;
      }
      const heading = line.match(/^(#{1,6})\s+(.*)$/);
      if (heading) {
        flushParagraph();
        flushList();
        blocks.push(`<h${heading[1].length}>${inline(heading[2])}</h${heading[1].length}>`);
        continue;
      }
      if (/^([-*_])\1{2,}$/.test(line)) {
        flushParagraph();
        flushList();
        blocks.push("<hr>");
        continue;
      }
      flushList();
      paragraph.push(line);
    }
    flushParagraph();
    flushList();
    return blocks.join("");
  }

  // src/task-manager/components/decisions-panel.js
  var STATUSES = ["proposed", "accepted", "rejected", "deprecated", "superseded"];
  var STATUS_LABELS2 = {
    proposed: "Propuesta",
    accepted: "Aceptada",
    rejected: "Rechazada",
    deprecated: "Obsoleta",
    superseded: "Sustituida"
  };
  var STATUS_BADGES = {
    proposed: "tk-badge--warning",
    accepted: "tk-badge--success",
    rejected: "tk-badge--danger",
    deprecated: "tk-badge--neutral",
    superseded: "tk-badge--info"
  };
  var CREATE_STATUSES = ["proposed", "accepted", "rejected", "deprecated"];
  var BODY_FIELDS = ["title", "context", "decision", "consequences", "alternatives"];
  var PAGE_SIZES = [10, 20, 40, 60];
  var SECTIONS = [
    { heading: "Contexto", field: "context" },
    { heading: "Decisi\xF3n", field: "decision" },
    { heading: "Consecuencias", field: "consequences" },
    { heading: "Alternativas consideradas", field: "alternatives" },
    { heading: "M\xE1s informaci\xF3n", field: "more_info" }
  ];
  var SEARCH_DELAY_MS = 220;
  var TO_TOP_AFTER_PX = 480;
  var DecisionsPanel = {
    name: "DecisionsPanel",
    mixins: [dialogFocus],
    props: {
      open: { type: Boolean, required: true },
      // Aviso de un cambio hecho desde fuera de esta ventana: el tablero lo
      // incrementa con cada aviso del canal, y aquí relee el registro.
      refreshToken: { type: Number, default: 0 }
    },
    emits: ["close"],
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
        formError: "",
        search: "",
        statusFilter: "",
        listError: "",
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
        if (!total) return "0 decisiones";
        if (this.pageCount === 1) return total === 1 ? "1 decisi\xF3n" : `${total} decisiones`;
        const last = Math.min(this.page * this.pageSize, total);
        return `${(this.page - 1) * this.pageSize + 1}\u2013${last} de ${total}`;
      },
      // El cuerpo se congela al salir de proposed: la pantalla deja de ofrecerlo.
      frozenBody() {
        const detail = this.editing ? this.details[this.editing] : null;
        return Boolean(detail) && detail.status !== "proposed";
      }
    },
    mounted() {
      window.addEventListener("keydown", this.handleEscape, true);
    },
    beforeUnmount() {
      window.removeEventListener("keydown", this.handleEscape, true);
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
        return STATUS_LABELS2[status] || status;
      },
      statusBadge(status) {
        return STATUS_BADGES[status] || "tk-badge--neutral";
      },
      // El cuerpo se escribe en markdown; lo que sale se pasa por la lista blanca
      // de siempre, igual que el texto de una tarea.
      html(value) {
        return richTextHtml(markdownHtml(value));
      },
      // El extracto del plegado es el texto del contexto, sin sus etiquetas: el
      // recorte a tres líneas lo hace el estilo, no este método.
      excerpt(value) {
        const box = document.createElement("div");
        box.innerHTML = this.html(value).replace(/<\/(p|li|ul|ol|h[1-6]|blockquote|pre|div)>/gi, " ");
        return String(box.textContent || "").replace(/\s+/g, " ").trim();
      },
      sectionsOf(detail) {
        return SECTIONS.filter((section) => String(detail[section.field] || "").trim());
      },
      isOpen(number) {
        return this.expanded.includes(number);
      },
      isEditing(number) {
        return this.editing === number;
      },
      async load() {
        const seq = ++this.loadSeq;
        this.listError = "";
        try {
          const adrs = await listADRs(this.search.trim());
          if (seq !== this.loadSeq) return;
          this.adrs = adrs;
          this.page = Math.min(this.page, this.pageCount);
        } catch (error) {
          if (seq !== this.loadSeq) return;
          this.listError = error?.message || "No se pudieron leer las decisiones.";
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
        const reduce = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
        scroller.scrollTo({ top, behavior: reduce ? "auto" : "smooth" });
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
          this.listError = error?.message || "No se pudo leer la decisi\xF3n.";
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
          title: "",
          deciders: "",
          status: "proposed",
          context: "",
          decision: "",
          consequences: "",
          alternatives: "",
          note: ""
        };
      },
      startNew() {
        this.creating = true;
        this.editing = 0;
        this.draft = this.emptyForm();
        this.formError = "";
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
          supersededBy: detail.superseded_by || "",
          context: detail.context || "",
          decision: detail.decision || "",
          consequences: detail.consequences || "",
          alternatives: detail.alternatives || "",
          note: ""
        };
        this.formError = "";
      },
      cancel() {
        this.editing = 0;
        this.creating = false;
        this.draft = null;
        this.formError = "";
      },
      // Escape cancela lo que se está escribiendo y deja el segundo Escape para
      // cerrar la capa, que escucha en la ventana.
      handleEscape(event) {
        if (event.key !== "Escape" || !this.open) return;
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
        const number = Number(String(reference || "").replace(/\D/g, ""));
        if (!number) return;
        this.search = "";
        this.statusFilter = "";
        this.page = 1;
        await this.load();
        this.revealPage(number);
        await this.expand(number);
        await this.$nextTick();
        document.getElementById(`adr-card-${number}`)?.scrollIntoView({ block: "start" });
      },
      async save() {
        if (!this.draft || this.busy) return;
        this.busy = true;
        this.formError = "";
        try {
          if (this.creating) await this.create();
          else await this.edit();
        } catch (error) {
          this.formError = error?.message || "No se pudo guardar la decisi\xF3n.";
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
        this.search = "";
        this.statusFilter = "";
        this.page = 1;
        await this.load();
        this.revealPage(created.number);
        await this.expand(created.number);
        await this.$nextTick();
        document.getElementById(`adr-card-${created.number}`)?.scrollIntoView({ block: "start" });
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
            if ((this.draft[field] || "") !== (detail[field] || "")) changes[field] = this.draft[field];
          }
        }
        if (this.draft.status !== detail.status) changes.status = this.draft.status;
        if (this.draft.status === "superseded" && this.draft.supersededBy !== (detail.superseded_by || "")) {
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
          type="button" title="Nueva decisi\xF3n" @click="startNew()"><tk-icon
            name="plus" />Nueva decisi\xF3n</button><button class="tk-icon-button" type="button"
          aria-label="Cerrar decisiones" title="Cerrar decisiones"
          @click="$emit('close')"><tk-icon name="x-mark" /></button></div>
    </header>
    <div class="decisions-body" ref="scroller" @scroll="onScroll">
      <div class="decisions-stack">
        <div class="decisions-toolbar"><input class="tk-input" type="search" aria-label="Buscar decisiones"
            placeholder="Buscar en todas las decisiones" :value="search"
            @input="search = $event.target.value" /><select class="tk-select" aria-label="Filtrar por estado"
            :value="statusFilter" @change="statusFilter = $event.target.value">
            <option value="">Todos los estados</option>
            <option v-for="status in statuses" :key="status" :value="status">{{ statusLabel(status) }}</option>
          </select><select class="tk-select tk-select--compact" aria-label="Decisiones por p\xE1gina" :value="pageSize"
            @change="setPageSize($event.target.value)">
            <option v-for="size in pageSizes" :key="size" :value="size">{{ size }} por p\xE1gina</option>
          </select></div>
        <div class="decisions-summary"><span>{{ rangeLabel }}</span>
          <div class="decisions-actions"><button class="tk-button tk-button--ghost" type="button"
              @click="expandAll()">Expandir todas</button><button class="tk-button tk-button--ghost" type="button"
              @click="collapseAll()">Plegar todas</button></div>
        </div>
        <p v-show="listError" class="decisions-error">{{ listError }}</p>

        <form v-if="creating" class="decisions-item is-open decisions-create" @submit.prevent="save()">
          <div class="decisions-form__bar"><span class="decisions-item__ref">Nueva decisi\xF3n</span>
            <div class="decisions-actions"><button class="tk-button tk-button--ghost tk-button--compact" type="button"
                title="Descartar cambios" @click="cancel()"><tk-icon
                  name="x-mark" />Cancelar</button><button class="tk-button tk-button--primary tk-button--compact"
                type="submit" title="Guardar decisi\xF3n" :disabled="busy"><tk-icon
                  name="check" />Guardar</button></div>
          </div>
          <div class="form-grid"><label class="tk-field span-2"><span class="tk-field__label">T\xEDtulo</span><input
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
              class="tk-field span-2"><span class="tk-field__label">Decisi\xF3n</span><textarea class="tk-textarea"
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
                aria-hidden="true"><tk-icon name="chevron-down" /></span></span></div>
            <h3 class="decisions-item__title">{{ adr.title }}</h3>
            <p v-if="!isOpen(adr.number)" class="decisions-item__excerpt">{{ excerpt(adr.context) }}</p>
          </div>

          <div v-if="isOpen(adr.number)" :id="'adr-' + adr.number" class="decisions-content">
            <div class="decisions-separator"></div>
            <p v-if="!details[adr.number] && !isEditing(adr.number)" class="decisions-empty">Leyendo la decisi\xF3n\u2026</p>
            <form v-if="isEditing(adr.number) && details[adr.number]" class="decisions-form" @submit.prevent="save()">
              <div class="decisions-form__bar"><span class="decisions-item__ref">Editando {{ adr.ref }}</span>
                <div class="decisions-actions"><button class="tk-button tk-button--ghost tk-button--compact"
                    type="button" title="Descartar cambios" @click="cancel()"><tk-icon
                      name="x-mark" />Cancelar</button><button class="tk-button tk-button--primary tk-button--compact"
                    type="submit" title="Guardar decisi\xF3n" :disabled="busy"><tk-icon
                      name="check" />Guardar</button></div>
              </div>
              <div class="form-grid"><label class="tk-field span-2"><span class="tk-field__label">T\xEDtulo</span><input
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
                  class="tk-field span-2"><span class="tk-field__label">Decisi\xF3n</span><textarea class="tk-textarea"
                    rows="4" :value="draft.decision" :disabled="frozenBody"
                    @input="draft.decision = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Consecuencias</span><textarea
                    class="tk-textarea" rows="4" :value="draft.consequences" :disabled="frozenBody"
                    @input="draft.consequences = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">Alternativas consideradas</span><textarea
                    class="tk-textarea" rows="4" :value="draft.alternatives" :disabled="frozenBody"
                    @input="draft.alternatives = $event.target.value"></textarea></label><label
                  class="tk-field span-2"><span class="tk-field__label">A\xF1adir a m\xE1s informaci\xF3n</span><textarea
                    class="tk-textarea" rows="3" :value="draft.note"
                    @input="draft.note = $event.target.value"></textarea></label></div>
              <p v-show="frozenBody" class="decisions-hint">El cuerpo est\xE1 congelado: una decisi\xF3n que ya no est\xE1
                propuesta solo admite a\xF1adidos en \xABM\xE1s informaci\xF3n\xBB.</p>
              <p v-show="formError" class="decisions-error">{{ formError }}</p>
            </form>
            <div v-if="!isEditing(adr.number) && details[adr.number]" class="decisions-document">
              <div class="decisions-editbar">
                <p class="decisions-document__meta"><span>Decisores: {{ details[adr.number].deciders }}</span><span
                    v-if="details[adr.number].superseded_by">Sustituido por <button class="decisions-link"
                      type="button"
                      @click="openReference(details[adr.number].superseded_by)">{{ details[adr.number].superseded_by
                      }}</button></span></p><button class="tk-button tk-button--secondary tk-button--compact"
                  type="button" title="Editar decisi\xF3n" @click="startEdit(adr.number)"><tk-icon
                    name="pencil" />Editar</button>
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

        <nav v-if="pageCount > 1" class="decisions-pager" aria-label="P\xE1ginas de decisiones"><button
            class="tk-button tk-button--ghost" type="button" :disabled="page === 1"
            @click="goToPage(page - 1)">Anterior</button><span>P\xE1gina {{ page }} de {{ pageCount }}</span><button
            class="tk-button tk-button--ghost" type="button" :disabled="page === pageCount"
            @click="goToPage(page + 1)">Siguiente</button></nav>
      </div>
    </div>
  </div>
  <button class="decisions-to-top" :class="{ 'is-visible': showToTop }" type="button" aria-label="Volver arriba"
    title="Volver arriba" @click="scrollToTop()"><tk-icon name="chevron-up" /></button>
</section>`
  };

  // src/task-manager/app-shell.js
  var AppShell = {
    ...workspaceOptions,
    name: "AppShell",
    components: { NewTaskDialog, TagManagerDrawer, TaskInspector, TaskList, DecisionsPanel },
    template: `
    <main class="workspace-frame" @click="handleWorkspaceClick">
  <div class="workspace-app">
    <header class="workspace-header">
      <div class="workspace-header__brand">
        <!-- El s\xEDmbolo aprobado vive en web/public/brand/, copiado del paquete de marca de docs/tasks-ai-brand. -->
        <div class="brand-mark"><img src="../brand/tasks-ai-icon-96.png" alt="" width="38" height="38" /></div>
        <div><strong>Gestor de tareas</strong><span>Local</span></div>
      </div>
      <div class="workspace-header__actions">
        <button class="tk-button tk-button--ghost tk-button--compact" type="button" aria-label="Actualizar datos"
          title="Actualizar datos" :disabled="refreshing || loading"
          @click.stop="reloadWorkspace()"><tk-icon name="arrow-path" :spinning="refreshing" /></button>
        <button class="tk-button tk-button--secondary tk-button--compact" type="button" data-decisions-open=""
          aria-label="Decisiones" title="Decisiones" :disabled="unavailable || loading"
          @click.stop="isDecisionsOpen = true"><tk-icon
            name="document-text" /><span class="tk-button__label">Decisiones</span></button>
        <button class="tk-button tk-button--secondary tk-button--compact" type="button" data-tag-manager-open=""
          aria-label="Etiquetas" title="Etiquetas" :disabled="unavailable || loading"
          @click.stop="isTagManagerOpen = true"><tk-icon
            name="tag" /><span class="tk-button__label">Etiquetas</span></button>
        <button class="tk-button tk-button--primary tk-button--compact" type="button" aria-label="Nueva tarea"
          title="Nueva tarea" :disabled="unavailable || loading"
          @click.stop="newTaskDraft = emptyTaskDraft(); isNewTaskOpen = true"><tk-icon
            name="plus" /><span class="tk-button__label">Nueva tarea</span></button>
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

  // src/task-manager/components/icon.js
  var TkIcon = {
    name: "TkIcon",
    props: {
      name: { type: String, required: true },
      spinning: { type: Boolean, default: false }
    },
    template: `
    <span :class="['tk-icon tk-icon--mask', 'tk-icon--' + name, { 'is-spinning': spinning }]"
      aria-hidden="true"></span>`
  };

  // src/task-manager/main.js
  var app = globalThis.Vue.createApp(AppShell);
  app.component("TkIcon", TkIcon);
  app.mount("#app");
})();
/**
 * toast-anchor v1.0.9
 * (c) 2026 Akash Prajapati
 * @license MIT
 * Vendorizado desde node_modules/toast-anchor/dist/index.mjs (build ESM) para no
 * declarar dependencias de ejecución, según RNF-01 del PRD. La licencia completa
 * acompaña al fichero en toast-anchor.LICENSE.
 */
