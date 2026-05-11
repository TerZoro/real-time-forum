package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"real-time-forum/database"
	"real-time-forum/models"

	"github.com/gofrs/uuid"
)

const messagesPageSize = 10

// GetMessages returns paginated messages between the current user and another user.
// Query params: with=<userID>, offset=<int> (default 0)
func GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	me, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	otherID := r.URL.Query().Get("with")
	if otherID == "" {
		jsonError(w, "with parameter required", http.StatusBadRequest)
		return
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		offset, _ = strconv.Atoi(v)
	}

	rows, err := database.DB.Query(
		`SELECT m.id, m.sender_id, m.receiver_id, m.content, m.created_at,
		        s.nickname AS sender_nick, rcv.nickname AS receiver_nick
		 FROM messages m
		 JOIN users s   ON m.sender_id   = s.id
		 JOIN users rcv ON m.receiver_id = rcv.id
		 WHERE (m.sender_id = ? AND m.receiver_id = ?)
		    OR (m.sender_id = ? AND m.receiver_id = ?)
		 ORDER BY m.created_at DESC
		 LIMIT ? OFFSET ?`,
		me.ID, otherID, otherID, me.ID, messagesPageSize, offset,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.CreatedAt,
			&m.SenderNick, &m.ReceiverNick)
		msgs = append(msgs, m)
	}

	// Reverse so oldest-first for display
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	if msgs == nil {
		msgs = []models.Message{}
	}
	jsonOK(w, msgs)
}

// GetConversations returns all users ordered by last message time, then alphabetically.
func GetConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	me, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query(
		`SELECT u.id, u.nickname,
		        MAX(m.created_at) AS last_at,
		        (SELECT content FROM messages
		         WHERE (sender_id = u.id AND receiver_id = ?)
		            OR (sender_id = ? AND receiver_id = u.id)
		         ORDER BY created_at DESC LIMIT 1) AS last_msg
		 FROM users u
		 LEFT JOIN messages m ON (m.sender_id = u.id AND m.receiver_id = ?)
		                      OR (m.sender_id = ? AND m.receiver_id = u.id)
		 WHERE u.id != ?
		 GROUP BY u.id
		 ORDER BY last_at DESC, u.nickname ASC`,
		me.ID, me.ID, me.ID, me.ID, me.ID,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []models.ConversationEntry
	for rows.Next() {
		var e models.ConversationEntry
		var lastAt *string
		var lastMsg *string
		rows.Scan(&e.UserID, &e.Nickname, &lastAt, &lastMsg)
		if lastMsg != nil {
			e.LastMessage = *lastMsg
			e.HasMessages = true
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []models.ConversationEntry{}
	}
	jsonOK(w, entries)
}

// SaveMessage persists a private message and returns the full populated record.
// Called by the WebSocket handler when it receives a chat message.
func SaveMessage(senderID, receiverID, content string) (models.Message, error) {
	content = strings.TrimSpace(content)
	id, _ := uuid.NewV4()

	_, err := database.DB.Exec(
		`INSERT INTO messages (id, sender_id, receiver_id, content) VALUES (?, ?, ?, ?)`,
		id.String(), senderID, receiverID, content,
	)
	if err != nil {
		return models.Message{}, err
	}

	var m models.Message
	err = database.DB.QueryRow(
		`SELECT m.id, m.sender_id, m.receiver_id, m.content, m.created_at,
		        s.nickname, rcv.nickname
		 FROM messages m
		 JOIN users s   ON m.sender_id   = s.id
		 JOIN users rcv ON m.receiver_id = rcv.id
		 WHERE m.id = ?`, id.String(),
	).Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.CreatedAt,
		&m.SenderNick, &m.ReceiverNick)
	return m, err
}
