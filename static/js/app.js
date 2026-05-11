/**
 * app.js
 * SPA router and orchestrator — ties everything together.
 *
 * Pages:
 *   feed       → #feed
 *   new-post   → #new-post
 *   post/:id   → #post/<id>
 *   chat/:uid  → #chat/<uid>/<nick>
 */


const App = (() => {

	/* ── DOM refs ── */
	const authScreen  = document.getElementById('auth-screen');
	const appShell    = document.getElementById('app-shell');
	const navNick     = document.getElementById('nav-nickname');
	const pages       = document.querySelectorAll('.page');
	const navLinks    = document.querySelectorAll('.nav-link');

	/* State */
	let _user = null;

	
	Auth.onAuthChange(async (user) => {
		_user = user;
		if (user) {
			_showApp(user);
		} else {
			_showAuth();
		}
	});

	async function _showApp(user) {
		authScreen.classList.add('hidden');
		appShell.classList.remove('hidden');
		navNick.textContent = user.nickname;

		// Start WebSocket
		WS.connect();

		// Wire WebSocket events
		WS.on('online_users', (payload) => Messages.updatePresence(payload));
		WS.on('presence',     (payload) => Messages.updatePresence(payload));
		WS.on('private_message', (msg)  => Messages.handleIncoming(msg));

		// Init messages sidebar
		await Messages.init(user);

		// Load categories for new-post form
		Posts.loadCategories();

		// Route based on current hash
		_route();
	}

	function _showAuth() {
		appShell.classList.add('hidden');
		authScreen.classList.remove('hidden');
		// Show login by default
		document.getElementById('login-form').classList.remove('hidden');
		document.getElementById('register-form').classList.add('hidden');
		WS.disconnect();
	}

	function navigate(page, params = {}) {
		switch (page) {
			case 'feed':
				location.hash = '#feed';
				break;
			case 'new-post':
				location.hash = '#new-post';
				break;
			case 'post':
				location.hash = `#post/${params.id}`;
				break;
			case 'chat':
				location.hash = `#chat/${params.userID}/${encodeURIComponent(params.nickname)}`;
				break;
		}
	}

	function _route() {
		if (!_user) return;

		const hash = location.hash || '#feed';
		const parts = hash.slice(1).split('/');
		const page  = parts[0];

		_setActivePage(page === 'post' || page === 'chat' ? page : page);

		switch (page) {
			case 'feed':
			case '':
				_showPage('page-feed');
				_setActiveNav('feed');
				Posts.loadFeed();
				break;

			case 'new-post':
				_showPage('page-new-post');
				_setActiveNav('new-post');
				break;

			case 'post':
				_showPage('page-post');
				_setActiveNav('');
				Posts.loadPost(parts[1]);
				break;

			case 'chat': {
				_showPage('page-chat');
				_setActiveNav('');
				const uid  = parts[1];
				const nick = decodeURIComponent(parts[2] || '');
				Messages.openChat(uid, nick);
				break;
			}

			default:
				location.hash = '#feed';
		}
	}

	window.addEventListener('hashchange', () => {
		if (_user) _route();
	});

	/* ── Show a single page section ── */
	function _showPage(id) {
		pages.forEach(p => p.classList.add('hidden'));
		const target = document.getElementById(id);
		if (target) target.classList.remove('hidden');
	}

	/* ── Highlight active nav link ── */
	function _setActivePage(page) {}

	function _setActiveNav(page) {
		navLinks.forEach(l => {
			l.classList.toggle('active', l.dataset.page === page);
		});
	}

	Auth.init();

	return { navigate };
})();
