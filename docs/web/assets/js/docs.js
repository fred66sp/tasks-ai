/* Mobilia de las páginas de docs/web (ADR-018): arranque de tema, preferencias
   en tasks-ai:prefs, mecanismo de idioma, la cabecera y el pie como islas Vue,
   el botón de copiar, el índice que marca lo que se lee y el volver arriba.
   Guion clásico — file:// no carga módulos ES — y sin empaquetado: lo que se
   versiona aquí es lo que el navegador ejecuta. Cada página aporta su
   configuración en un <script type="application/json" id="docs-config"> y su
   diccionario inglés donde lo tiene. */
(function () {
  "use strict";

  var root = document.documentElement;
  var STORAGE_KEY = "tasks-ai:prefs";

  /* — Arranque: el tema se aplica antes del primer trazo, porque este guion
     viaja en el <head> como script clásico. — */
  var prefs = {};
  try { prefs = JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}"); } catch (e) { prefs = {}; }

  var theme = (prefs.theme === "dark" || prefs.theme === "light") ? prefs.theme :
    (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  root.setAttribute("data-theme", theme);

  function savePrefs() {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs)); } catch (e) { /* La página sigue siendo usable sin almacenamiento. */ }
  }

  function readConfig() {
    var el = document.getElementById("docs-config");
    if (!el) { return {}; }
    try { return JSON.parse(el.textContent); } catch (e) { return {}; }
  }

  /* — Mecanismo de idioma: el diccionario es contenido de cada página; aquí
     vive solo el mecanismo. Cada nodo [data-i18n] guarda su texto original una
     vez: cambiar de idioma es devolverlo tal cual, no volver a traducir. — */
  var dictionary = null;
  var originals = [];
  var ariaNodes = [];
  var titleNode = document.querySelector("title[data-title-en]");
  var descriptionNode = document.querySelector("meta[data-content-en]");
  var titleEs = titleNode ? titleNode.textContent : "";
  var descriptionEs = descriptionNode ? descriptionNode.getAttribute("content") : "";

  function copyLabel(lang) { return lang === "en" ? "Copy" : "Copiar"; }
  function copiedLabel(lang) { return lang === "en" ? "Copied" : "Copiado"; }

  function themeLabelFor(themeName, lang) {
    if (lang === "en") { return themeName === "dark" ? "Switch to light theme" : "Switch to dark theme"; }
    return themeName === "dark" ? "Cambiar a tema claro" : "Cambiar a tema oscuro";
  }

  function refreshCopyButtons(lang) {
    document.querySelectorAll(".code .copy").forEach(function (btn) {
      if (btn.classList.contains("copied")) { return; }
      btn.innerHTML = COPY_ICON + "<span>" + copyLabel(lang) + "</span>";
      btn.setAttribute("aria-label", copyLabel(lang));
    });
  }

  function applyLang(lang) {
    if (lang !== "en") { lang = "es"; }
    state.lang = lang;
    root.lang = lang;
    if (titleNode) { titleNode.textContent = lang === "en" ? titleNode.getAttribute("data-title-en") : titleEs; }
    if (descriptionNode) { descriptionNode.setAttribute("content", lang === "en" ? descriptionNode.getAttribute("data-content-en") : descriptionEs); }
    ariaNodes.forEach(function (item) {
      item.el.setAttribute("aria-label", lang === "en" ? item.en : item.es);
    });
    originals.forEach(function (item) {
      var translated = lang === "en" && dictionary ? dictionary[item.key] : null;
      item.el.innerHTML = translated == null ? item.html : translated;
    });
    document.querySelectorAll("[data-lang]").forEach(function (btn) {
      btn.setAttribute("aria-pressed", btn.getAttribute("data-lang") === lang ? "true" : "false");
    });
    // Los ejemplos llevan su versión en cada idioma: se enseña una y se oculta
    // la otra, y el bloque que solo tiene una se queda como está.
    document.querySelectorAll(".code").forEach(function (block) {
      var shown = null;
      block.querySelectorAll("pre[data-lang]").forEach(function (pre) {
        var mine = pre.getAttribute("data-lang") === lang;
        pre.hidden = !mine;
        if (mine) { shown = pre; }
      });
      var only = block.querySelector("pre:not([data-lang])");
      if (only && shown) { only.hidden = true; }
    });
    refreshCopyButtons(lang);
    updateThemeButton();
  }

  /* — El botón de tema: una sola escritura de la preferencia y de la
     accesibilidad, compartida por la isla Vue y por el arranque. — */
  function updateThemeButton() {
    var btn = document.getElementById("theme-toggle");
    if (!btn) { return; }
    btn.setAttribute("aria-label", themeLabelFor(root.getAttribute("data-theme"), root.lang === "en" ? "en" : "es"));
  }

  function setTheme(next) {
    root.setAttribute("data-theme", next);
    state.theme = next;
    prefs.theme = next;
    savePrefs();
    updateThemeButton();
  }

  function toggleTheme() {
    setTheme(root.getAttribute("data-theme") === "dark" ? "light" : "dark");
  }

  function setLang(next) {
    prefs.lang = next;
    savePrefs();
    applyLang(next);
  }

  var COPY_ICON = '<span class="tk-icon tk-icon--mask tk-icon--square-2-stack" aria-hidden="true"></span>';
  var CHECK_ICON = '<span class="tk-icon tk-icon--mask tk-icon--check" aria-hidden="true"></span>';

  /* — Botón de copiar: se crea aquí para que añadir un bloque de código no
     obligue a recordar su botón. El aspecto del bloque es de cada página. — */
  function initCopyButtons() {
    document.querySelectorAll(".code").forEach(function (block) {
      if (block.querySelector(".copy") || !block.querySelector("pre")) { return; }
      block.classList.add("has-copy");
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "copy";
      block.appendChild(btn);

      btn.addEventListener("click", function () {
        var visible = block.querySelector("pre:not([hidden])") || block.querySelector("pre");
        var code = (visible && (visible.querySelector("code") || visible)) || null;
        if (!code) { return; }
        var lang = root.lang === "en" ? "en" : "es";
        var done = function () {
          btn.classList.add("copied");
          btn.innerHTML = CHECK_ICON + "<span>" + copiedLabel(lang) + "</span>";
          btn.setAttribute("aria-label", copiedLabel(lang));
          setTimeout(function () {
            btn.classList.remove("copied");
            refreshCopyButtons(root.lang === "en" ? "en" : "es");
          }, 1600);
        };
        var text = code.textContent;
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(done, done);
        } else {
          var ta = document.createElement("textarea");
          ta.value = text;
          ta.style.position = "fixed";
          ta.style.opacity = "0";
          document.body.appendChild(ta);
          ta.select();
          try { document.execCommand("copy"); } catch (e) {}
          document.body.removeChild(ta);
          done();
        }
      });
    });
    refreshCopyButtons(root.lang === "en" ? "en" : "es");
  }

  /* — El índice que marca lo que se lee. Dos modos, escritos una vez:
     intersección (guía y portada) y posición (rail del catálogo: dos capítulos
     pueden cruzarse a la vez y entonces el último en avisar dejaba marcado el
     que ya se había leído). — */
  function initSpy() {
    document.querySelectorAll("[data-spy]").forEach(function (nav) {
      var links = {};
      Array.prototype.forEach.call(nav.querySelectorAll("a"), function (a) {
        var href = a.getAttribute("href");
        if (href && href.charAt(0) === "#" && href.length > 1) { links[href.slice(1)] = a; }
      });
      var ids = Object.keys(links);
      if (!ids.length) { return; }
      var clear = function () {
        ids.forEach(function (id) { links[id].classList.remove("active"); links[id].removeAttribute("aria-current"); });
      };
      var mark = function (id) {
        clear();
        if (links[id]) {
          if (nav.getAttribute("data-spy-mark") === "aria-current") { links[id].setAttribute("aria-current", "true"); }
          else { links[id].classList.add("active"); }
        }
      };
      var targets = (nav.getAttribute("data-spy-target") || "").split(",").filter(Boolean);
      if (nav.getAttribute("data-spy-mode") === "position") {
        var chapters = targets.map(function (sel) { return document.querySelector(sel); }).filter(Boolean);
        var byPosition = function () {
          var line = window.innerHeight * parseFloat(nav.getAttribute("data-spy-line") || "0.28");
          var active = null;
          chapters.forEach(function (el) {
            if (el.getBoundingClientRect().top <= line) { active = el; }
          });
          clear();
          if (active && links[active.id]) { links[active.id].setAttribute("aria-current", "true"); }
        };
        window.addEventListener("scroll", byPosition, { passive: true });
        window.addEventListener("resize", byPosition);
        byPosition();
        return;
      }
      if (!("IntersectionObserver" in window)) { return; }
      var observer = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) { mark(entry.target.id); }
        });
      }, { rootMargin: nav.getAttribute("data-spy-margin") || "-20% 0px -70% 0px", threshold: 0 });
      targets.forEach(function (sel) {
        document.querySelectorAll(sel).forEach(function (s) { observer.observe(s); });
      });
    });
  }

  /* — Volver arriba: visible al bajar de la primera pantalla. — */
  function initToTop() {
    var toTop = document.getElementById("to-top");
    if (!toTop) { return; }
    var update = function () { toTop.classList.toggle("is-visible", window.scrollY > 480); };
    window.addEventListener("scroll", update, { passive: true });
    update();
  }

  /* — Islas Vue: la cabecera y el pie. Sin router y sin SPA: cada página es
     estática y estos componentes montan solo la mobilia. El estado común
     (tema e idioma) es reactivo: los conmutadores se repintan solos. — */
  var state = window.Vue ? window.Vue.reactive({ theme: theme, lang: "es" }) : { theme: theme, lang: "es" };

  function makeComponents(Vue) {
    var LangSwitch = {
      props: ["mode", "aria"],
      computed: {
        pressed: function () { return state.lang; },
        locked: function () { return this.mode !== "live"; }
      },
      methods: {
        pick: function (lang) { if (this.mode === "live") { setLang(lang); } }
      },
      template:
        '<div class="lang-switch" role="group" :aria-label="aria">' +
        '<button type="button" data-lang="es" lang="es" :aria-pressed="pressed === \'es\' ? \'true\' : \'false\'" @click="pick(\'es\')">ES</button>' +
        '<button type="button" data-lang="en" lang="en" :aria-pressed="pressed === \'en\' ? \'true\' : \'false\'" :aria-disabled="locked ? \'true\' : undefined" @click="pick(\'en\')">EN</button>' +
        '</div>'
    };

    var ThemeToggle = {
      template:
        '<button class="icon-btn" id="theme-toggle" type="button" :aria-label="label" @click="toggle">' +
        '<span class="tk-icon tk-icon--mask tk-icon--moon icon-moon" aria-hidden="true"></span>' +
        '<span class="tk-icon tk-icon--mask tk-icon--sun icon-sun" aria-hidden="true"></span>' +
        '<span data-i18n="header.2">Tema</span>' +
        '</button>',
      computed: {
        label: function () { return themeLabelFor(state.theme, state.lang); }
      },
      methods: { toggle: toggleTheme }
    };

    var TopBar = {
      props: ["cfg"],
      components: { "lang-switch": LangSwitch, "theme-toggle": ThemeToggle },
      computed: {
        spyAttrs: function () {
          return this.cfg.spyNav ? { "data-spy": "", "data-spy-target": "main > section:not(.hero)", "data-spy-margin": "-15% 0px -70% 0px" } : {};
        }
      },
      // Sin <header> envolvente: la isla monta dentro del propio
      // <header class="topbar"> de la página, que así conserva el sticky (ADR-018).
      template:
        '<template v-if="true">' +
        '<a class="brand" :href="cfg.brand.href">' +
        '<span class="brand-mark" aria-hidden="true">' +
        '<img class="brand-icon-light" src="assets/img/app-icons/tasks-ai-light-64.png" alt="" width="34" height="34">' +
        '<img class="brand-icon-dark" src="assets/img/app-icons/tasks-ai-dark-64.png" alt="" width="34" height="34">' +
        '</span>' +
        // header.1 traduce el contenido completo de la marca, como en la página original:
        // la clave va en el span, no en el small, o el inglés anidaría una segunda marca.
        '<span class="brand-name" :data-i18n="cfg.brand.tagline.i18n">tasks-ai<small>{{ cfg.brand.tagline.es }}</small></span>' +
        '</a>' +
        '<span class="topbar-spacer"></span>' +
        '<a v-if="cfg.navStyle === \'catalog\'" class="icon-btn catalog-btn" :href="cfg.nav[0].href" :aria-label="cfg.nav[0].ariaEs" :data-aria-en="cfg.nav[0].ariaEn">' +
        '<span class="tk-icon tk-icon--mask" :class="\'tk-icon--\' + cfg.nav[0].icon" aria-hidden="true"></span>' +
        '<span :data-i18n="cfg.nav[0].i18n">{{ cfg.nav[0].es }}</span></a>' +
        '<nav v-else class="site-nav" :aria-label="cfg.navAria" v-bind="spyAttrs">' +
        '<template v-for="item in cfg.nav" :key="item.href">' +
        '<a v-if="item.icon" class="icon-btn" :class="item.cls" :href="item.href" :aria-label="item.es"><span class="tk-icon tk-icon--mask" :class="\'tk-icon--\' + item.icon" aria-hidden="true"></span><span :data-i18n="item.i18n">{{ item.es }}</span></a>' +
        '<a v-else :href="item.href" :class="item.cls" :data-i18n="item.i18n">{{ item.es }}</a>' +
        '</template>' +
        '</nav>' +
        '<lang-switch :mode="cfg.lang" :aria="cfg.langAria"></lang-switch>' +
        '<theme-toggle></theme-toggle>' +
        '</template>'
    };

    var PageFooter = {
      template: '<footer><slot></slot></footer>'
    };

    return { LangSwitch: LangSwitch, ThemeToggle: ThemeToggle, TopBar: TopBar, PageFooter: PageFooter };
  }

  function collectI18n() {
    // Después de montar las islas: los nodos que dibujan los componentes
    // también llevan sus claves data-i18n.
    originals = [];
    Array.prototype.forEach.call(document.querySelectorAll("[data-i18n]"), function (el) {
      originals.push({ el: el, key: el.getAttribute("data-i18n"), html: el.innerHTML });
    });
    ariaNodes = [];
    Array.prototype.forEach.call(document.querySelectorAll("[data-aria-en]"), function (el) {
      ariaNodes.push({ el: el, es: el.getAttribute("aria-label"), en: el.getAttribute("data-aria-en") });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    var config = readConfig();
    var components = null;
    var Vue = window.Vue;

    if (Vue) {
      components = makeComponents(Vue);

      var topbarEl = document.getElementById("docs-topbar");
      if (topbarEl && config.topbar) {
        Vue.createApp({
          components: { "top-bar": components.TopBar },
          template: '<top-bar :cfg="cfg"></top-bar>',
          data: function () { return { cfg: config.topbar }; }
        }).mount(topbarEl);
      }

      var footerEl = document.getElementById("docs-footer");
      if (footerEl) {
        Vue.createApp({
          components: { "page-footer": components.PageFooter }
          // La plantilla viaja en el propio elemento: el pie de cada página es
          // contenido suyo; el componente aporta el marco común.
        }).mount(footerEl);
      }
    }

    var dictEl = document.getElementById("tasks-ai-en");
    if (dictEl) {
      try { dictionary = JSON.parse(dictEl.textContent); } catch (e) { dictionary = null; }
    }

    collectI18n();
    if (dictionary) {
      state.lang = prefs.lang === "en" ? "en" : "es";
      applyLang(state.lang);
    } else {
      state.lang = "es";
      updateThemeButton();
    }

    if (config.copy) { initCopyButtons(); }
    initSpy();
    initToTop();
  });
})();
