// Runs before first paint (loaded with strategy="beforeInteractive" from
// app/layout.tsx) so the page never flashes the wrong palette. Mirrors
// lib/theme.ts: light unless the visitor chose dark; the OS preference is not
// consulted. The storage keys and the fallback must stay in step with that
// module — test/theme.test.ts reads this file and checks.
(function () {
  var dark = false;
  try {
    var t = localStorage.getItem('mathua.theme');
    if (t !== 'dark' && t !== 'light') t = localStorage.getItem('mathua-theme');
    dark = t === 'dark';
  } catch (e) {}
  var root = document.documentElement;
  if (dark) {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
  var meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.setAttribute('content', dark ? '#0c0d0f' : '#ffffff');
})();
