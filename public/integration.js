document.addEventListener('submit', async (e) => {
	if (e.target.id === 'telegram-auth-form') {
		e.preventDefault();
		const form = e.target;
		const status = document.getElementById('telegram-auth-status');
		const btn = form.querySelector('button[type="submit"]');

		btn.disabled = true;

		try {
			const formData = new FormData(form);
			const response = await fetch('/api/telegram/auth', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					bot_token: formData.get('bot_token'),
					chat_id: formData.get('chat_id'),
				}),
			});

			if (response.ok) {
				location.reload();
			} else {
				const error = await response.json();
				status.textContent = `Error: ${error.error || 'Failed to save'}`;
				status.className = 'auth-status error';
				btn.disabled = false;
			}
		} catch (err) {
			status.textContent = `Error: ${err.message}`;
			status.className = 'auth-status error';
			btn.disabled = false;
		}
	}

	if (e.target.id === 'telegram-chatid-form') {
		e.preventDefault();
		const form = e.target;
		const status = document.getElementById('telegram-chatid-status');
		const btn = form.querySelector('button[type="submit"]');

		btn.disabled = true;

		try {
			const formData = new FormData(form);
			const botToken = form.dataset.botToken;
			const response = await fetch('/api/telegram/auth', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					bot_token: botToken,
					chat_id: formData.get('chat_id'),
				}),
			});

			if (response.ok) {
				location.reload();
			} else {
				const error = await response.json();
				status.textContent = `Error: ${error.error || 'Failed to save'}`;
				status.className = 'auth-status error';
				btn.disabled = false;
			}
		} catch (err) {
			status.textContent = `Error: ${err.message}`;
			status.className = 'auth-status error';
			btn.disabled = false;
		}
	}

	if (e.target.id === 'proton-auth-form') {
		e.preventDefault();
		const form = e.target;
		const status = document.getElementById('proton-auth-status');
		const btn = form.querySelector('button[type="submit"]');

		btn.disabled = true;

		try {
			const formData = new FormData(form);
			const response = await fetch('/api/proton/auth', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					email: formData.get('email'),
					password: formData.get('password'),
					totp: formData.get('totp'),
				}),
			});

			if (response.ok) {
				location.reload();
			} else {
				const error = await response.json();
				status.textContent = `Error: ${error.error || 'Authentication failed'}`;
				status.className = 'auth-status error';
				btn.disabled = false;
			}
		} catch (err) {
			status.textContent = `Error: ${err.message}`;
			status.className = 'auth-status error';
			btn.disabled = false;
		}
	}
});

let signalLinkingPoll = null;

document.addEventListener('click', async (e) => {
	if (e.target.id === 'signal-link-btn') {
		const btn = e.target;
		const qrContainer = document.getElementById('signal-qr-container');
		const qrCode = document.getElementById('signal-qr-code');

		btn.style.display = 'none';
		qrContainer.style.display = 'none';

		try {
			const response = await fetch('/api/signal/link', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ device_name: 'Prism' }),
			});

			if (!response.ok) {
				const error = await response.json();
				qrContainer.innerHTML = `<p class="channel-not-configured">Error: ${error.error || 'Failed to generate link'}</p>`;
				qrContainer.style.display = 'block';
				btn.style.display = 'inline-block';
				return;
			}

			const data = await response.json();
			const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=400x400&data=${encodeURIComponent(data.qr_code)}`;
			qrCode.src = qrUrl;
			qrContainer.style.display = 'block';

			signalLinkingPoll = setInterval(async () => {
				try {
					const statusResp = await fetch('/api/signal/status');
					const statusData = await statusResp.json();

					if (statusData.linked) {
						clearInterval(signalLinkingPoll);
						qrContainer.innerHTML =
							'<p class="auth-status success">Linked! Refreshing...</p>';
						setTimeout(() => location.reload(), 1000);
					}
				} catch (err) {
					console.error('Status check failed:', err);
				}
			}, 2000);
		} catch (err) {
			qrContainer.innerHTML = `<p class="channel-not-configured">Error: ${err.message}</p>`;
			qrContainer.style.display = 'block';
			btn.style.display = 'inline-block';
		}
	}

	if (e.target.classList.contains('reload-btn')) {
		location.reload();
	}

	if (e.target.classList.contains('delete-telegram-btn')) {
		e.preventDefault();

		if (!confirm('Unlink Telegram integration?')) {
			return;
		}

		const btn = e.target;
		btn.disabled = true;

		try {
			const response = await fetch('/api/telegram/auth', {
				method: 'DELETE',
			});

			if (response.ok) {
				location.reload();
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

	if (e.target.classList.contains('delete-proton-btn')) {
		if (!confirm('Unlink Proton Mail integration?')) {
			return;
		}

		try {
			const response = await fetch('/api/proton/auth', {
				method: 'DELETE',
			});

			if (response.ok) {
				location.reload();
			} else {
				const error = await response.json();
				alert(`Error: ${error.error || 'Failed to unlink integration'}`);
			}
		} catch (err) {
			alert(`Error: ${err.message}`);
		}
	}
});
