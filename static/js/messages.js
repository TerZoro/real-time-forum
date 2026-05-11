/**
 * messages.js
 * Private messaging: sidebar user list, chat view, pagination, real-time.
 * Exposes:
 *   Messages.init(currentUser)
 *   Messages.openChat(userID, nickname)
 *   Messages.handleIncoming(msg)   – called by app.js when WS delivers a message
 *   Messages.updatePresence(status) – called by app.js on presence/online_users events
 */

const Messages = (() => {

	/* ── State ── */
	let me = null;
	let currentChatUserID = null;
	let currentChatNick   = null;
	let msgOffset = 0;
	let allLoaded = false;
	let isLoadingMore = false;

	// userID → { nickname, online, lastMessage, lastAt }
	const userMap = new Map();

	/* ── DOM refs (cached lazily so they exist after DOMContentLoaded) ── */
	const userListEl    = () => document.getElementById('user-list');
	const chatMsgsEl    = () => document.getElementById('chat-messages');
	const chatNameEl    = () => document.getElementById('chat-with-name');
	const chatStatusEl  = () => document.getElementById('chat-with-status');
	const chatInput     = () => document.getElementById('chat-input');
	const loadMoreBtn   = () => document.getElementById('btn-load-more');
	const loadMoreWrap  = () => document.getElementById('chat-load-more');

	async function init(user) {
		me = user;
		await _loadConversations();
		_bindChatForm();
		_bindLoadMore();
	}

	async function _loadConversations() {
		try {
			const res = await fetch('/api/conversations');
			const data = await res.json();
			data.forEach(entry => {
				userMap.set(entry.user_id, {
					nickname:    entry.nickname,
					online:      false, // will be updated by presence
					lastMessage: entry.last_message || '',
					lastAt:      entry.last_at ? new Date(entry.last_at) : null,
					hasMessages: entry.has_messages,
				});
			});
			_renderSidebar();
		} catch { /* silent */ }
	}

	function _renderSidebar() {
		const ul = userListEl();
		if (!ul) return;

		// Sort: users with messages by lastAt DESC, then alphabetical
		const sorted = [...userMap.entries()].sort(([, a], [, b]) => {
			if (a.hasMessages && b.hasMessages) {
				return (b.lastAt || 0) - (a.lastAt || 0);
			}
			if (a.hasMessages) return -1;
			if (b.hasMessages) return 1;
			return a.nickname.localeCompare(b.nickname);
		});

		ul.innerHTML = sorted.map(([uid, u]) => `
			<li data-uid="${uid}" class="${uid === currentChatUserID ? 'active' : ''}">
				<span class="status-dot ${u.online ? 'online' : ''}"></span>
				<span class="nick">${escHtml(u.nickname)}</span>
				${u.lastMessage
					? `<span class="last-msg">${escHtml(u.lastMessage)}</span>`
					: ''}
			</li>
		`).join('');

		ul.querySelectorAll('li[data-uid]').forEach(li => {
			li.addEventListener('click', () => {
				const uid  = li.dataset.uid;
				const nick = userMap.get(uid)?.nickname || '';
				App.navigate('chat', { userID: uid, nickname: nick });
			});
		});
	}

	function updatePresence(status) {
		if (Array.isArray(status)) {
			// online_users — full list of online users
			// First mark everyone offline
			userMap.forEach(u => { u.online = false; });
			status.forEach(s => {
				if (s.user_id === me?.id) return;
				if (!userMap.has(s.user_id)) {
					userMap.set(s.user_id, {
						nickname: s.nickname, online: true,
						lastMessage: '', lastAt: null, hasMessages: false,
					});
				} else {
					userMap.get(s.user_id).online = true;
				}
			});
		} else {
			// single presence event
			if (status.user_id === me?.id) return;
			if (!userMap.has(status.user_id)) {
				userMap.set(status.user_id, {
					nickname: status.nickname, online: status.online,
					lastMessage: '', lastAt: null, hasMessages: false,
				});
			} else {
				userMap.get(status.user_id).online = status.online;
			}
		}

		_renderSidebar();

		// Update chat header status dot if the chat partner changed presence
		if (currentChatUserID) {
			const u = userMap.get(currentChatUserID);
			const dot = chatStatusEl();
			if (dot && u) dot.className = `status-dot ${u.online ? 'online' : ''}`;
		}
	}

	async function openChat(userID, nickname) {
		currentChatUserID = userID;
		currentChatNick   = nickname;
		msgOffset         = 0;
		allLoaded         = false;

		// Update header
		chatNameEl().textContent = nickname;
		const u = userMap.get(userID);
		chatStatusEl().className = `status-dot ${u?.online ? 'online' : ''}`;

		// Clear messages and load first batch
		chatMsgsEl().innerHTML = '';
		await _fetchMessages(false);

		// Highlight sidebar
		_renderSidebar();

		// Focus input
		chatInput().focus();
	}

	async function _fetchMessages(loadMore) {
		if (isLoadingMore) return;
		isLoadingMore = true;

		try {
			const res = await fetch(`/api/messages?with=${currentChatUserID}&offset=${msgOffset}`);
			const msgs = await res.json();

			if (!Array.isArray(msgs) || msgs.length === 0) {
				if (loadMore) {
					allLoaded = true;
					loadMoreWrap().classList.add('hidden');
				}
				isLoadingMore = false;
				return;
			}

			// Track scroll position before inserting so we can restore it
			const box = chatMsgsEl();
			const prevScrollHeight = box.scrollHeight;

			if (loadMore) {
				// Prepend older messages
				const fragment = document.createDocumentFragment();
				msgs.forEach(m => {
					const el = _buildBubble(m);
					fragment.appendChild(el);
				});
				box.insertBefore(fragment, box.firstChild);
				// Restore scroll so user stays at same position
				box.scrollTop = box.scrollHeight - prevScrollHeight;
			} else {
				// Initial load: append and scroll to bottom
				msgs.forEach(m => box.appendChild(_buildBubble(m)));
				box.scrollTop = box.scrollHeight;
			}

			msgOffset += msgs.length;

			// Show/hide load-more button
			if (msgs.length < 10) {
				allLoaded = true;
				loadMoreWrap().classList.add('hidden');
			} else {
				loadMoreWrap().classList.remove('hidden');
			}
		} catch { /* silent */ }

		isLoadingMore = false;
	}

	function handleIncoming(msg) {
		// Update sidebar last-message preview
		const partnerID = msg.sender_id === me?.id ? msg.receiver_id : msg.sender_id;
		const u = userMap.get(partnerID);
		if (u) {
			u.lastMessage = msg.content;
			u.lastAt      = new Date(msg.created_at);
			u.hasMessages = true;
		}
		_renderSidebar();

		// If this chat is open, append message
		if (partnerID === currentChatUserID || msg.sender_id === currentChatUserID) {
			const box = chatMsgsEl();
			const atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 60;
			box.appendChild(_buildBubble(msg));
			msgOffset++;
			if (atBottom) box.scrollTop = box.scrollHeight;
		}
	}

	function _bindChatForm() {
		document.getElementById('chatForm').addEventListener('submit', (e) => {
			e.preventDefault();
			const input = chatInput();
			const content = input.value.trim();
			if (!content || !currentChatUserID) return;

			WS.send('private_message', {
				receiver_id: currentChatUserID,
				content,
			});

			input.value = '';
			input.focus();
		});
	}

	function _bindLoadMore() {
		// Button click
		document.getElementById('btn-load-more').addEventListener('click', () => {
			if (!allLoaded) _fetchMessages(true);
		});

		const box = chatMsgsEl();
		if (box) {
			box.addEventListener('scroll', throttle(() => {
				if (box.scrollTop < 40 && !allLoaded && !isLoadingMore) {
					_fetchMessages(true);
				}
			}, 300));
		}
	}

	function _buildBubble(msg) {
		const isMine = msg.sender_id === me?.id;
		const div = document.createElement('div');
		div.className = `msg-bubble ${isMine ? 'mine' : 'theirs'}`;
		div.innerHTML = `
			<span class="msg-meta">${escHtml(msg.sender_nick)} · ${fmtDate(msg.created_at)}</span>
			<span class="msg-text">${escHtml(msg.content)}</span>
		`;
		return div;
	}

	function fmtDate(iso) {
		return new Date(iso).toLocaleString(undefined, {
			month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit',
		});
	}

	function escHtml(str) {
		return String(str)
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#39;');
	}

	function throttle(fn, ms) {
		let last = 0;
		return function (...args) {
			const now = Date.now();
			if (now - last >= ms) {
				last = now;
				fn.apply(this, args);
			}
		};
	}

	return { init, openChat, handleIncoming, updatePresence };
})();
