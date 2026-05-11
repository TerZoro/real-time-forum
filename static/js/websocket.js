/**
 * websocket.js
 * Manages the single WebSocket connection for the entire app.
 * Exposes:
 *   WS.connect()
 *   WS.disconnect()
 *   WS.send(type, payload)
 *   WS.on(type, callback)      – subscribe to a message type
 *   WS.off(type, callback)     – unsubscribe
 */

const WS = (() => {
	let socket = null;
	const listeners = {}; // type → Set<fn>
	let reconnectTimer = null;
	let shouldReconnect = true;

	function connect() {
		shouldReconnect = true;
		_open();
	}

	function disconnect() {
		shouldReconnect = false;
		clearTimeout(reconnectTimer);
		if (socket) {
			socket.close();
			socket = null;
		}
	}

	function _open() {
		const proto = location.protocol === 'https:' ? 'wss' : 'ws';
		socket = new WebSocket(`${proto}://${location.host}/ws`);

		socket.addEventListener('open', () => {
			console.log('[WS] connected');
			clearTimeout(reconnectTimer);
		});

		socket.addEventListener('message', (ev) => {
			let msg;
			try { msg = JSON.parse(ev.data); } catch { return; }
			_dispatch(msg.type, msg.payload);
		});

		socket.addEventListener('close', () => {
			console.log('[WS] closed');
			socket = null;
			if (shouldReconnect) {
				reconnectTimer = setTimeout(_open, 3000);
			}
		});

		socket.addEventListener('error', () => {
			socket.close();
		});
	}

	function send(type, payload) {
		if (socket && socket.readyState === WebSocket.OPEN) {
			socket.send(JSON.stringify({ type, payload }));
		}
	}

	function on(type, cb) {
		if (!listeners[type]) listeners[type] = new Set();
		listeners[type].add(cb);
	}

	function off(type, cb) {
		listeners[type]?.delete(cb);
	}

	function _dispatch(type, payload) {
		listeners[type]?.forEach(cb => cb(payload));
		// Also dispatch to wildcard listeners
		listeners['*']?.forEach(cb => cb({ type, payload }));
	}

	return { connect, disconnect, send, on, off };
})();
