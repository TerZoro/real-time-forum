/**
 * auth.js
 * Handles registration, login, logout, and session state.
 * Exposes:
 *   Auth.currentUser        – the logged-in user object or null
 *   Auth.init()             – check /api/me on page load
 *   Auth.logout()
 *   Auth.onAuthChange(fn)   – called with (user|null) on auth state changes
 */
const Auth = (() => {
	let currentUser = null;
	const authListeners = new Set();

	function onAuthChange(fn) { authListeners.add(fn); }

	function _notify(user) {
		currentUser = user;
		authListeners.forEach(fn => fn(user));
	}

	/* ── Check existing session ── */
	async function init() {
		try {
			const res = await fetch('/api/me');
			if (res.ok) {
				const user = await res.json();
				_notify(user);
			} else {
				_notify(null);
			}
		} catch {
			_notify(null);
		}
	}

	/* ── Login form ── */
	const loginForm = document.getElementById('loginForm');
	const loginError = document.getElementById('login-error');

	loginForm.addEventListener('submit', async (e) => {
		e.preventDefault();
		loginError.classList.add('hidden');

		const fd = new FormData(loginForm);
		const body = {
			identifier: fd.get('identifier').trim(),
			password:   fd.get('password'),
		};

		try {
			const res = await fetch('/api/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body),
			});
			const data = await res.json();
			if (!res.ok) {
				loginError.textContent = data.error || 'Login failed';
				loginError.classList.remove('hidden');
				return;
			}
			loginForm.reset();
			_notify(data);
		} catch {
			loginError.textContent = 'Network error. Please try again.';
			loginError.classList.remove('hidden');
		}
	});

	/* ── Register form ── */
	const registerForm = document.getElementById('registerForm');
	const registerError = document.getElementById('register-error');

	registerForm.addEventListener('submit', async (e) => {
		e.preventDefault();
		registerError.classList.add('hidden');

		const fd = new FormData(registerForm);
		const body = {
			nickname:   fd.get('nickname').trim(),
			first_name: fd.get('first_name').trim(),
			last_name:  fd.get('last_name').trim(),
			email:      fd.get('email').trim(),
			age:        parseInt(fd.get('age'), 10),
			gender:     fd.get('gender'),
			password:   fd.get('password'),
		};

		if (!body.gender) {
			registerError.textContent = 'Please select a gender.';
			registerError.classList.remove('hidden');
			return;
		}

		try {
			const res = await fetch('/api/register', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body),
			});
			const data = await res.json();
			if (!res.ok) {
				registerError.textContent = data.error || 'Registration failed';
				registerError.classList.remove('hidden');
				return;
			}
			registerForm.reset();
			_notify(data);
		} catch {
			registerError.textContent = 'Network error. Please try again.';
			registerError.classList.remove('hidden');
		}
	});

	/* ── Toggle between login / register ── */
	document.getElementById('go-register').addEventListener('click', (e) => {
		e.preventDefault();
		document.getElementById('login-form').classList.add('hidden');
		document.getElementById('register-form').classList.remove('hidden');
		registerError.classList.add('hidden');
	});

	document.getElementById('go-login').addEventListener('click', (e) => {
		e.preventDefault();
		document.getElementById('register-form').classList.add('hidden');
		document.getElementById('login-form').classList.remove('hidden');
		loginError.classList.add('hidden');
	});

	/* ── Logout ── */
	async function logout() {
		await fetch('/api/logout', { method: 'POST' });
		WS.disconnect();
		_notify(null);
	}

	document.getElementById('btn-logout').addEventListener('click', logout);

	return { init, logout, onAuthChange, get currentUser() { return currentUser; } };
})();
