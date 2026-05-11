/**
 * posts.js
 * Handles post feed, single post view, creating posts and comments.
 * Exposes:
 *   Posts.loadFeed()
 *   Posts.loadPost(id)
 *   Posts.loadCategories()
 */
const Posts = (() => {

	function fmtDate(iso) {
		return new Date(iso).toLocaleString(undefined, {
			year: 'numeric', month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit',
		});
	}

	async function loadFeed() {
		const list = document.getElementById('post-list');
		list.innerHTML = '<li class="empty-state">Loading…</li>';

		let posts;
		try {
			const res = await fetch('/api/posts');
			posts = await res.json();
		} catch {
			list.innerHTML = '<li class="empty-state">Could not load posts.</li>';
			return;
		}

		if (!Array.isArray(posts) || posts.length === 0) {
			list.innerHTML = '<li class="empty-state">No posts yet. Be the first!</li>';
			return;
		}

		list.innerHTML = posts.map(p => `
			<li>
				<div class="post-card" data-id="${p.id}">
					<h3>${escHtml(p.title)}</h3>
					<div class="post-meta">
						<span>by <strong>${escHtml(p.nickname)}</strong></span>
						<span>${fmtDate(p.created_at)}</span>
					</div>
					<p class="post-content-preview">${escHtml(p.content)}</p>
					<div class="post-tags">
						${(p.categories || []).map(c => `<span class="tag">${escHtml(c.name)}</span>`).join('')}
					</div>
				</div>
			</li>
		`).join('');

		list.querySelectorAll('.post-card').forEach(card => {
			card.addEventListener('click', () => {
				App.navigate('post', { id: card.dataset.id });
			});
		});
	}

	let currentPostID = null;

	async function loadPost(id) {
		currentPostID = id;
		const detail = document.getElementById('post-detail');
		const commentList = document.getElementById('comment-list');
		detail.innerHTML = '<p class="empty-state">Loading…</p>';
		commentList.innerHTML = '';

		let post;
		try {
			const res = await fetch(`/api/posts/${id}`);
			if (!res.ok) { detail.innerHTML = '<p class="empty-state">Post not found.</p>'; return; }
			post = await res.json();
		} catch {
			detail.innerHTML = '<p class="empty-state">Could not load post.</p>';
			return;
		}

		detail.innerHTML = `
			<h2>${escHtml(post.title)}</h2>
			<div class="post-meta">
				<span>by <strong>${escHtml(post.nickname)}</strong></span>
				<span>${fmtDate(post.created_at)}</span>
			</div>
			<div class="post-tags">
				${(post.categories || []).map(c => `<span class="tag">${escHtml(c.name)}</span>`).join('')}
			</div>
			<p class="post-body" style="margin-top:16px">${escHtml(post.content)}</p>
		`;

		renderComments(post.comments || []);
	}

	function renderComments(comments) {
		const list = document.getElementById('comment-list');
		if (comments.length === 0) {
			list.innerHTML = '<li class="empty-state">No comments yet.</li>';
			return;
		}
		list.innerHTML = comments.map(c => `
			<li class="comment-item">
				<div class="comment-meta">
					<strong>${escHtml(c.nickname)}</strong> · ${fmtDate(c.created_at)}
				</div>
				<div class="comment-body">${escHtml(c.content)}</div>
			</li>
		`).join('');
	}

	document.getElementById('commentForm').addEventListener('submit', async (e) => {
		e.preventDefault();
		if (!currentPostID) return;

		const form = e.target;
		const content = form.content.value.trim();
		if (!content) return;

		try {
			const res = await fetch('/api/comments', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ post_id: currentPostID, content }),
			});
			if (!res.ok) return;

			form.reset();
			// Reload post to refresh comments
			loadPost(currentPostID);
		} catch { /* silent */ }
	});

	async function loadCategories() {
		const container = document.getElementById('category-list');
		let cats;
		try {
			const res = await fetch('/api/categories');
			cats = await res.json();
		} catch { return; }

		container.innerHTML = cats.map(c => `
			<span>
				<input type="checkbox" class="category-checkbox" id="cat-${c.id}" value="${c.id}" />
				<label class="category-label" for="cat-${c.id}">${escHtml(c.name)}</label>
			</span>
		`).join('');
	}

	document.getElementById('newPostForm').addEventListener('submit', async (e) => {
		e.preventDefault();
		const errEl = document.getElementById('new-post-error');
		errEl.classList.add('hidden');

		const form = e.target;
		const title   = form.title.value.trim();
		const content = form.content.value.trim();
		const cats    = [...document.querySelectorAll('.category-checkbox:checked')].map(el => parseInt(el.value));

		if (!title || !content) {
			errEl.textContent = 'Title and content are required.';
			errEl.classList.remove('hidden');
			return;
		}
		if (cats.length === 0) {
			errEl.textContent = 'Select at least one category.';
			errEl.classList.remove('hidden');
			return;
		}

		try {
			const res = await fetch('/api/posts', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ title, content, categories: cats }),
			});
			const data = await res.json();
			if (!res.ok) {
				errEl.textContent = data.error || 'Failed to create post.';
				errEl.classList.remove('hidden');
				return;
			}
			form.reset();
			document.querySelectorAll('.category-checkbox').forEach(el => el.checked = false);
			App.navigate('feed');
		} catch {
			errEl.textContent = 'Network error.';
			errEl.classList.remove('hidden');
		}
	});

	document.getElementById('cancel-post').addEventListener('click', () => App.navigate('feed'));
	document.getElementById('back-to-feed').addEventListener('click', () => App.navigate('feed'));

	function escHtml(str) {
		return String(str)
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#39;');
	}

	return { loadFeed, loadPost, loadCategories };
})();
