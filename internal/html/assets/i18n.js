const translations = {{TRANSLATIONS_JSON}};
{{EXTRA_DECLARATIONS}}

let currentLang = 'en';

// A bundle can carry a language code this page has no table for: a
// hand-edited project.yml, or a code saved by a newer version. Map it to
// the closest one we have (zh-tw → zh-TW, pt-BR → pt), else English.
function supportedLang(lang) {
  if (lang && translations[lang]) return lang;
  const lower = String(lang || '').toLowerCase();
  const codes = Object.keys(translations);
  return codes.find((c) => c.toLowerCase() === lower)
    || codes.find((c) => c.toLowerCase() === lower.split('-')[0])
    || 'en';
}

function t(key, ...args) {
  let text = (translations[currentLang] || {})[key] || translations['en'][key] || key;
  args.forEach((arg, i) => {
    text = text.replace(`{${i}}`, arg);
  });
  return text;
}

function setLanguage(lang) {
  lang = supportedLang(lang);
  currentLang = lang;
  localStorage.setItem('rememory-lang', lang);

  // Update select
  const sel = document.getElementById('lang-select');
  if (sel) sel.value = lang;

  // Update all translatable elements
  document.querySelectorAll('[data-i18n]').forEach(el => {
    const key = el.dataset.i18n;
    el.textContent = t(key);
  });

  // Update placeholder attributes
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
    const key = el.dataset.i18nPlaceholder;
    el.placeholder = t(key);
  });

  // Update page title (only if the component provides one)
  var translatedTitle = t('title');
  if (translatedTitle && translatedTitle !== 'title') {
    document.title = translatedTitle;
  }

  // Re-render dynamic content
  if (typeof window.rememoryUpdateUI === 'function') {
    window.rememoryUpdateUI();
  }

  {{SET_LANGUAGE_EXTRA}}

  // Update docs links to point to the correct language variant.
  // This runs after SET_LANGUAGE_EXTRA so that links injected via
  // data-i18n-html (innerHTML) are also rewritten.
  if (typeof docsLangs !== 'undefined') {
    const docsFile = (lang !== 'en' && docsLangs.indexOf(lang) !== -1)
      ? 'docs.' + lang + '.html' : 'docs.html';
    document.querySelectorAll('a[href^="docs."]').forEach(a => {
      const h = a.getAttribute('href');
      a.setAttribute('href', h.replace(/docs(?:\.[a-z]{2})?\.html/, docsFile));
    });
  }
}

// Set initial language immediately
(function() {
  const saved = localStorage.getItem('rememory-lang');
  const langs = {{LANG_DETECT_ARRAY}};
  const detected = navigator.languages.find((l) => langs.includes(l))
    || navigator.languages.map((l) => l.split('-')[0]).find((l) => langs.includes(l));
  currentLang = supportedLang(saved || detected);
})();

// Initialize language select after DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  {{DOM_CONTENT_LOADED_PRE}}

  // Attached first so the picker keeps working even if rendering throws.
  document.getElementById('lang-select')?.addEventListener('change', (e) => {
    setLanguage(e.target.value);
  });

  setLanguage(currentLang);
});
