const themes = ['system', 'light', 'dark'];
let currentIndex = 0;

const savedTheme = localStorage.getItem('theme') || 'system';
currentIndex = themes.indexOf(savedTheme);
if (currentIndex === -1) currentIndex = 0;

function applyTheme(theme) {
  if (theme === 'system') {
    document.documentElement.removeAttribute('data-theme');
  } else {
    document.documentElement.setAttribute('data-theme', theme);
  }
  localStorage.setItem('theme', theme);
}

window.cycleTheme = () => {
  currentIndex = (currentIndex + 1) % themes.length;
  const newTheme = themes[currentIndex];
  applyTheme(newTheme);
  updateButtonText(newTheme);
};

function updateButtonText(theme) {
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.textContent = theme === 'system' ? '🌓' : theme === 'light' ? '☀️' : '🌙';
  }
}

applyTheme(savedTheme);

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    updateButtonText(themes[currentIndex]);
    const btn = document.getElementById('theme-toggle');
    if (btn) btn.addEventListener('click', window.cycleTheme);
  });
} else {
  updateButtonText(themes[currentIndex]);
  const btn = document.getElementById('theme-toggle');
  if (btn) btn.addEventListener('click', window.cycleTheme);
}
