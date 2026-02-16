function showToast(message, type = 'info') {
	const container = document.getElementById('toast-container');
	if (!container) return;

	const toast = document.createElement('div');
	toast.className = `toast ${type}`;
	toast.textContent = message;

	container.appendChild(toast);

	setTimeout(() => {
		toast.classList.add('hiding');
		setTimeout(() => toast.remove(), 300);
	}, 3000);
}

document.addEventListener('DOMContentLoaded', () => {
	document.body.addEventListener('htmx:afterRequest', (event) => {
		const xhr = event.detail.xhr;
		const triggerHeader = xhr.getResponseHeader('HX-Trigger');

		if (triggerHeader) {
			try {
				const trigger = JSON.parse(triggerHeader);
				if (trigger.showToast) {
					showToast(trigger.showToast.message, trigger.showToast.type);
				}
			} catch (_e) {
				// Not JSON, ignore
			}
		}
	});
});
