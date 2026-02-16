document.addEventListener('DOMContentLoaded', () => {
	document.addEventListener('submit', (e) => {
		const form = e.target;
		if (form.classList.contains('auth-form')) {
			e.preventDefault();
			const handler = form.dataset.handler;
			if (handler === 'telegram') submitTelegramAuth(e);
			else if (handler === 'telegram-chatid') submitTelegramChatId(e);
			else if (handler === 'proton') submitProtonAuth(e);
		}
	});

	document.addEventListener('click', (e) => {
		const btn = e.target.closest('[data-action]');
		if (!btn) return;
		const action = btn.dataset.action;
		if (action === 'link-signal') linkSignal(btn);
		else if (action === 'delete-telegram') deleteTelegram(btn);
		else if (action === 'delete-proton') deleteProton(btn);
		else if (action === 'reload') reloadIntegrations();
	});
});

function reloadIntegrations() {
	const integrations = document.getElementById('integrations');
	if (integrations) {
		htmx.trigger(integrations, 'reload');
	}
}

async function handleAuthForm(form, endpoint, statusId, getPayload) {
	const status = document.getElementById(statusId);
	const btn = form.querySelector('button[type="submit"]');
	const showError = (msg) => {
		status.textContent = `Error: ${msg}`;
		status.className = 'auth-status error';
		btn.disabled = false;
	};

	btn.disabled = true;

	try {
		const formData = new FormData(form);
		const response = await fetch(endpoint, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(getPayload(formData, form)),
		});

		if (response.ok) {
			checkToast(response);
			reloadIntegrations();
		} else {
			const error = await response.json();
			showError(error.error || 'Failed to save');
		}
	} catch (err) {
		showError(err.message);
	}
}

function checkToast(response) {
	const trigger = response.headers.get('HX-Trigger');
	if (trigger) {
		try {
			const data = JSON.parse(trigger);
			if (data.showToast && window.showToast) {
				showToast(data.showToast.message, data.showToast.type);
			}
		} catch (_e) {
			// Ignore
		}
	}
}

async function submitTelegramAuth(e) {
	await handleAuthForm(
		e.target,
		'/api/v1/telegram/link',
		'telegram-auth-status',
		(fd) => ({
			bot_token: fd.get('bot_token'),
			chat_id: fd.get('chat_id'),
		}),
	);
}

async function submitTelegramChatId(e) {
	await handleAuthForm(
		e.target,
		'/api/v1/telegram/link',
		'telegram-chatid-status',
		(fd, form) => ({
			bot_token: form.dataset.botToken,
			chat_id: fd.get('chat_id'),
		}),
	);
}

async function submitProtonAuth(e) {
	await handleAuthForm(
		e.target,
		'/api/v1/proton/auth',
		'proton-auth-status',
		(fd) => ({
			email: fd.get('email'),
			password: fd.get('password'),
			totp: fd.get('totp'),
		}),
	);
}

let signalLinkingPoll = null;

async function linkSignal(btn) {
	const qrContainer = document.getElementById('signal-qr-container');
	const qrCode = document.getElementById('signal-qr-code');
	const showQrError = (msg) => {
		qrContainer.innerHTML = `<p class="channel-not-configured">Error: ${msg}</p>`;
		qrContainer.style.display = 'block';
		btn.style.display = 'inline-block';
	};

	btn.style.display = 'none';
	qrContainer.style.display = 'none';

	try {
		const response = await fetch('/api/v1/signal/link', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ device_name: 'Prism' }),
		});

		if (!response.ok) {
			const error = await response.json();
			showQrError(error.error || 'Failed to generate link');
			return;
		}

		const data = await response.json();
		const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=400x400&data=${encodeURIComponent(data.qr_code)}`;
		qrCode.src = qrUrl;
		qrContainer.style.display = 'block';

		let statusCheckInProgress = false;
		signalLinkingPoll = setInterval(async () => {
			if (statusCheckInProgress) return;

			statusCheckInProgress = true;
			try {
				const statusResp = await fetch('/api/v1/signal/status');
				const statusData = await statusResp.json();

				if (statusData.linked) {
					clearInterval(signalLinkingPoll);
					qrContainer.innerHTML =
						'<p class="auth-status success">Linked! Refreshing...</p>';
					showToast('Signal linked', 'success');
					setTimeout(() => reloadIntegrations(), 1000);
				}
			} catch (err) {
				console.error('Status check failed:', err);
			} finally {
				statusCheckInProgress = false;
			}
		}, 2000);
	} catch (err) {
		showQrError(err.message);
	}
}

async function deleteTelegram(btn) {
	if (!confirm('Unlink Telegram integration?')) return;

	btn.disabled = true;

	try {
		const response = await fetch('/api/v1/telegram/link', { method: 'DELETE' });

		if (response.ok) {
			checkToast(response);
			reloadIntegrations();
		} else {
			const error = await response.json();
			alert(`Error: ${error.error || 'Failed to unlink integration'}`);
			btn.disabled = false;
		}
	} catch (err) {
		alert(`Error: ${err.message}`);
		btn.disabled = false;
	}
}

async function deleteProton(btn) {
	if (!confirm('Unlink Proton Mail integration?')) return;

	btn.disabled = true;

	try {
		const response = await fetch('/api/v1/proton/auth', { method: 'DELETE' });

		if (response.ok) {
			checkToast(response);
			reloadIntegrations();
		} else {
			const error = await response.json();
			alert(`Error: ${error.error || 'Failed to unlink integration'}`);
			btn.disabled = false;
		}
	} catch (err) {
		alert(`Error: ${err.message}`);
		btn.disabled = false;
	}
}
