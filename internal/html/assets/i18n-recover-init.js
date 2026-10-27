// If personalized with a language preference and no saved preference, use it
if (window.PERSONALIZATION && window.PERSONALIZATION.language && !localStorage.getItem('rememory-lang')) {
  currentLang = supportedLang(window.PERSONALIZATION.language);
}

// Hide "Recover" link (current page) from the default nav
document.querySelector('#nav-links-main a[href="recover.html"]')?.remove();

// Toggle nav links: bundle mode shows only Guide (absolute), standalone shows all (relative)
if (window.PERSONALIZATION) {
  document.getElementById('nav-links-main')?.classList.add('hidden');
  document.getElementById('nav-links-bundle')?.classList.remove('hidden');
}
