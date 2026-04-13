(() => {
	var t = localStorage.getItem('theme');
	if (t && t !== 'system')
		document.documentElement.setAttribute('data-theme', t);
})();
