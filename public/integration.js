function showConfirm(btn, message, onConfirm) {
	const wrapper = document.createElement('span');
	wrapper.className = 'confirm-inline';
	wrapper.setAttribute('role', 'group');
	wrapper.setAttribute('aria-label', message);
	wrapper.setAttribute('aria-live', 'polite');

	const msg = document.createElement('span');
	msg.className = 'confirm-message';
	msg.textContent = message;

	const yes = document.createElement('button');
	yes.type = 'button';
	yes.className = 'btn-danger btn-sm';
	yes.textContent = 'Yes';

	const cancel = document.createElement('button');
	cancel.type = 'button';
	cancel.className = 'btn-secondary btn-sm';
	cancel.textContent = 'Cancel';

	yes.addEventListener('click', () => {
		wrapper.replaceWith(btn);
		onConfirm();
	});
	cancel.addEventListener('click', () => {
		delete btn.dataset.confirming;
		wrapper.replaceWith(btn);
	});

	wrapper.append(msg, '\u00a0', yes, '\u00a0', cancel);
	btn.replaceWith(wrapper);
}

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
		else if (action === 'toggle-password') togglePassword(btn);
		else if (action === 'delete-app') deleteApp(btn);
		else if (action === 'delete-subscription') deleteSubscription(btn);
	});
});

function togglePassword(btn) {
	const input = btn.closest('.password-wrapper').querySelector('input');
	const isHidden = input.type === 'password';
	input.type = isHidden ? 'text' : 'password';
	btn.querySelector('.eye-show').style.display = isHidden ? 'none' : 'block';
	btn.querySelector('.eye-hide').style.display = isHidden ? 'block' : 'none';
	btn.setAttribute('aria-label', isHidden ? 'Hide password' : 'Show password');
}

function reloadIntegrations() {
	const integrations = document.getElementById('integrations');
	if (integrations) {
		htmx.trigger(integrations, 'reload');
	}
	const appsList = document.getElementById('apps-list');
	if (appsList) {
		htmx.trigger(appsList, 'reload');
	}
}

function deleteApp(btn) {
	const appName = btn.dataset.appName;
	showConfirm(btn, `Delete ${appName} and all subscriptions?`, () => {
		htmx.ajax('DELETE', `/apps/${appName}`, {
			target: '#apps-list',
			swap: 'innerHTML',
		});
	});
}

function deleteSubscription(btn) {
	const url = btn.dataset.url;
	showConfirm(btn, 'Delete this channel?', () => {
		htmx.ajax('DELETE', url, { target: '#apps-list', swap: 'innerHTML' });
	});
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

document.addEventListener('htmx:beforeSwap', (e) => {
	if (!signalLinkingPoll) return;
	const t = e.detail.target;
	if (t && (t.id === 'integrations' || t.id === 'signal-integration')) {
		clearInterval(signalLinkingPoll);
		signalLinkingPoll = null;
	}
});

document.addEventListener('htmx:confirm', (e) => {
	if (!e.detail.question) return;
	if (e.detail.elt.dataset.confirming) return;
	e.preventDefault();
	e.detail.elt.dataset.confirming = '1';
	showConfirm(e.detail.elt, e.detail.question, () => {
		delete e.detail.elt.dataset.confirming;
		e.detail.issueRequest(true);
	});
});

async function linkSignal(btn) {
	if (signalLinkingPoll) {
		clearInterval(signalLinkingPoll);
		signalLinkingPoll = null;
	}
	const qrContainer = document.getElementById('signal-qr-container');
	const qrCode = document.getElementById('signal-qr-code');
	const showQrError = (msg) => {
		const p = document.createElement('p');
		p.className = 'channel-not-configured';
		p.textContent = `Error: ${msg}`;
		qrContainer.replaceChildren(p);
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
		qrCode.src = data.qr_code;
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
					const linked = document.createElement('p');
					linked.className = 'auth-status success';
					linked.textContent = 'Linked! Refreshing...';
					qrContainer.replaceChildren(linked);
					showToast('Signal linked', 'success');
					setTimeout(() => reloadIntegrations(), 1000);
				}
			} catch (err) {
				console.error('Status check failed:', err);
			} finally {
				statusCheckInProgress = false;
			}
		}, 2000);

		setTimeout(() => {
			if (signalLinkingPoll) {
				clearInterval(signalLinkingPoll);
				signalLinkingPoll = null;
				showQrError('QR code expired. Try again.');
			}
		}, 300000);
	} catch (err) {
		showQrError(err.message);
	}
}

async function deleteTelegram(btn) {
	showConfirm(btn, 'Unlink Telegram?', async () => {
		try {
			const response = await fetch('/api/v1/telegram/link', {
				method: 'DELETE',
			});
			if (response.ok) {
				checkToast(response);
				reloadIntegrations();
			} else {
				const error = await response.json();
				showToast(error.error || 'Failed to unlink Telegram', 'error');
			}
		} catch (err) {
			showToast(err.message, 'error');
		}
	});
}

async function deleteProton(btn) {
	showConfirm(btn, 'Unlink Proton Mail?', async () => {
		try {
			const response = await fetch('/api/v1/proton/auth', { method: 'DELETE' });
			if (response.ok) {
				checkToast(response);
				reloadIntegrations();
			} else {
				const error = await response.json();
				showToast(error.error || 'Failed to unlink Proton', 'error');
			}
		} catch (err) {
			showToast(err.message, 'error');
		}
	});
}
